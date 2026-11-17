package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Reversal path (contract 01-CONTRACT.md §3.5 ReverseCommissionTx + appendix C1 top-up void, COMM-04).
// ErrTopUpNotFound / ErrTopUpStatusInvalid reuse the existing sentinels in topup.go (do not redefine).

// ErrAlreadyStatemented is returned by reversal when the original commission flow is already statemented (available)
// (contract §3.5 + §7#4: refunds after statementing go through a manual adjustment; history is never changed automatically).
// The caller VoidTopUp therefore rolls back the entire void transaction; partial voids are forbidden.
var ErrAlreadyStatemented = errors.New("commission flow already statemented")

// ReverseCommissionTx creates a negative reversal flow (contract §3.5 frozen signature).
// Must run inside the caller's transaction (only touch the passed-in tx, never the global DB - Pitfall 2):
//   - No original commission flow (top-up during switch-off / unbooked order) -> no-op, return nil
//   - Original flow already statemented (available) -> return ErrAlreadyStatemented; the caller rolls back everything
//   - Pending original flow -> generate exactly one negative reversal flow, leaving the original record untouched (COMM-04 no-delete semantics)
//
// reversal BillingNo=tradeNo: same-order reversal uniqueness is backed solely by the uk_flow_billing(billing_no, flow_type) unique index (§3.1.1);
// Period copies the original flow: the same order nets out at statement generation, so voids during the cross-month buffer never split positive/negative flows across two statements.
func ReverseCommissionTx(tx *gorm.DB, tradeNo string, reason string) error {
	original := &CommissionFlow{}
	// Query the original commission flow in any status first (not just pending): statemented ones must be explicitly rejected rather than silently no-op'd
	err := tx.Where("trade_no = ? AND flow_type = ?", tradeNo, CommissionFlowCommission).
		Order("id DESC").First(original).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Not booked (top-up during switch-off / no commission attribution) -> no reversal needed
		}
		return err
	}
	if original.Status == CommissionFlowAvailable {
		return ErrAlreadyStatemented // Already statemented -> automatic history changes are forbidden (contract §7#4)
	}
	if original.Status != CommissionFlowPending {
		return nil // Defensive branch: unknown status produces no reversal
	}
	return tx.Create(&CommissionFlow{
		BillingNo:       tradeNo, // §3.1.1: reversal billing_no = original trade_no
		FlowType:        CommissionFlowReversal,
		TradeNo:         tradeNo,
		CustomerId:      original.CustomerId,
		DistributorId:   original.DistributorId,
		TopupMoneyCents: original.TopupMoneyCents,
		RateBp:          original.RateBp,
		AmountCents:     -original.AmountCents, // Negative reversal
		Status:          CommissionFlowPending,
		Period:          original.Period, // Net against the same period (do not use the current month)
		RelatedFlowId:   original.Id,
		Reason:          reason,
		OperatorId:      0, // System flow
	}).Error
	// No field of the original flow is modified (the original record is retained)
}

// VoidTopUp voids a top-up order (contract 01-CONTRACT.md appendix C1 frozen: single transaction, all-or-nothing).
// (1) Status -> refunded (2) Claw back the originally credited quota (mirroring each channel's credit formula in topupQuotaToCredit:
// Creem raw Amount, Stripe Money x QuotaPerUnit, others Amount x QuotaPerUnit; negative balances allowed, clamping forbidden)
// (3) ReverseCommissionTx reversal (ErrAlreadyStatemented -> full rollback, no partial void)
// (4) Audit trail outside the transaction (RecordLog uses the global DB internally, and the audit must not disappear with a rollback).
func VoidTopUp(tradeNo string, reason string, operatorId int) error {
	if tradeNo == "" {
		return errors.New("Order number is required")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Unified lock discipline (contract appendix D5.4 + checker m-3 fix): the statement transaction generateStatementTx
		// takes the distributor users row as its first lock - the void write path locks the same distributor row before any business write,
		// closing the millisecond race where a void concurrent with "statement SELECT pending but before commit" reads the pending original flow and produces a cross-period reversal
		// (T-04-03-06). The lock target is the original commission flow's DistributorId, not the current inviter attribution
		// column: a concurrent void and rebind may have rewritten that column, so locking by the current value cannot protect the original flow's distributor.
		// Order: lock-free read of tradeNo to locate topUp -> lock-free query of the original flow for DistributorId -> lock that users row
		// (skip if the original flow does not exist, i.e. no commission path) -> then load topUp with a row lock as before and continue.
		// The locating read is a lock-free snapshot used only to choose the lock target; the Status gate is evaluated after acquiring the row lock, so there is no TOCTOU risk.
		topUpPre := &TopUp{}
		if err := tx.Where(refCol+" = ?", tradeNo).Select("id, user_id").First(topUpPre).Error; err != nil {
			return ErrTopUpNotFound
		}
		lockDistId := 0
		var original CommissionFlow
		if err := tx.Where("trade_no = ? AND flow_type = ?", tradeNo, CommissionFlowCommission).
			Order("id DESC").First(&original).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// No original commission flow (top-up during switch-off / unbooked order) -> no commission path, skip the distributor lock
		} else {
			lockDistId = original.DistributorId
		}
		if lockDistId > 0 {
			// Anomalies such as a deleted distributor row do not block the void (failing to lock only loses mutual exclusion; business gates are unaffected)
			var lockUser User
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				Select("id").First(&lockUser, "id = ?", lockDistId).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		topUp := &TopUp{}
		// Row-level lock, serializes concurrent voids of the same order (T-03-03-05)
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		// Pre-check: only success can be voided (pending/failed/expired/refunded = duplicate void are all rejected, contract C1)
		if topUp.Status != common.TopUpStatusSuccess {
			return ErrTopUpStatusInvalid
		}

		// (1) Set status to refunded
		topUp.Status = common.TopUpStatusRefunded
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// (2) Claw back the originally credited quota: call the single source of truth topupQuotaToCredit, mirroring each channel's credit formula
		// (Creem=raw Amount, Stripe=Money x QuotaPerUnit, Epay/Waffo/WaffoPancake=Amount x QuotaPerUnit).
		// The former constant Amount x QuotaPerUnit formula over-clawed Creem orders by 500,000x and used the wrong discount basis for Stripe (6.3 Fix 1).
		// Contract C1: negative balances are allowed - trust pre-consumption can overdraw, SQL has no lower-bound clamp, and new requests after a negative balance are naturally blocked by the pre-consumption check;
		// max(0,...) clamping is forbidden - it would break the "clawback" semantics and is asymmetric with crediting.
		quotaToDeduct := topupQuotaToCredit(topUp)
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota - ?", quotaToDeduct)).Error; err != nil {
			return err
		}

		// (3) Reversal (contract §3.5): ErrAlreadyStatemented -> return error -> the entire void transaction rolls back
		// (status restored to success, no quota deducted, no flow - all three consistent, no partial void, T-03-03-03)
		if err := ReverseCommissionTx(tx, tradeNo, reason); err != nil {
			return err
		}

		userId = topUp.UserId
		return nil
	})
	if err != nil {
		return err
	}

	// (4) Audit trail (executed outside the transaction)
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("Top-up order voided trade_no=%s operator_id=%d reason=%s", tradeNo, operatorId, reason))
	return nil
}
