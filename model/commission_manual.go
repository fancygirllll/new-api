package model

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Super-admin manual credit/debit flows (contract 01-CONTRACT.md §4.3 A7 + §3.1.1 + appendix D1/D5.1, COMM-05).
//
// Idempotency key: BillingNo = "MAN-" prefix + 14-digit timestamp + 4 random alphanumerics (§3.1.1 frozen format;
// manual flows have no trade_no, and uk_flow_billing(billing_no, flow_type) is naturally unique).
//
// Persistence semantics (appendix D1 frozen defaults): manual_debit amounts are always positive on the request side and stored negative;
// TradeNo empty, CustomerId=0, TopupMoneyCents=0, RateBp=0;
// Period = current month at record time -> automatically picked up by the next statement sweep (the D1 "next statement inclusion" mechanism;
// backfilling historical periods is a contract addendum, not part of this phase).
//
// Gating semantics (appendix D5.1 frozen): the adjustment entry is not gated by the commission master switch - correction is a ledger-integrity operation;
// the switch only controls new bookkeeping entry points; this function has no gated branch.

// CreateManualFlow lets a super-admin manually credit (manual_credit) or debit (manual_debit) a commission flow.
// Fail-fast validation order following the A2 precedent: flow_type whitelist -> amount positive -> reason required/width ->
// target exists in the transaction and its role is exactly a distributor (no thresholds - admin/root cannot be adjusted).
func CreateManualFlow(distributorId int, flowType string, amountCents int64, reason string, operatorId int) error {
	if flowType != CommissionFlowManualCredit && flowType != CommissionFlowManualDebit {
		return fmt.Errorf("Invalid flow type %q", flowType)
	}
	if amountCents <= 0 {
		// manual_debit is stored negative while always positive on the request side (contract A7)
		return fmt.Errorf("Amount must be positive")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("Reason is required")
	}
	if utf8.RuneCountInString(reason) > 255 {
		return fmt.Errorf("Reason must not exceed 255 characters")
	}

	signed := amountCents
	if flowType == CommissionFlowManualDebit {
		signed = -amountCents // Reversals are stored negative (contract A7/D1)
	}
	billingNo := fmt.Sprintf("MAN-%s-%s", time.Now().Format("20060102150405"), common.GetRandomString(4))
	return DB.Transaction(func(tx *gorm.DB) error {
		var dist User
		if err := tx.Select("id, role").First(&dist, "id = ?", distributorId).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("Target distributor does not exist")
			}
			return err
		}
		if dist.Role != common.RoleDistributorUser { // Exact == (A2 precedent, no thresholds)
			return fmt.Errorf("Target user is not a distributor")
		}
		return tx.Create(&CommissionFlow{
			BillingNo:       billingNo,
			FlowType:        flowType,
			TradeNo:         "", // Manual flows have no top-up order (D1)
			CustomerId:      0,
			DistributorId:   distributorId,
			TopupMoneyCents: 0,
			RateBp:          0,
			AmountCents:     signed,
			Status:          CommissionFlowPending,        // Consumed by the next statement sweep
			Period:          time.Now().Format("2006-01"), // Current month -> automatically included in the next statement (D1)
			Reason:          reason,
			OperatorId:      operatorId,
		}).Error
	})
}
