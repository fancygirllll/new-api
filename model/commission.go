package model

import (
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// RecordCommissionTx is the commission bookkeeping engine (contract 01-CONTRACT.md §3.5 frozen signature and call timing).
//
// Transaction discipline (Pitfall 2): only use the passed-in tx for queries/writes (tx.Select / tx.First / tx.Create);
// do not use the user lookup helpers in model/user.go that use the global DB (get user by invitation code, get user by ID) -
// they run on a connection outside the transaction: the main transaction rollback creates a half-commit illusion, and combined with a MaxOpenConns(1) test DB they always deadlock.
//
// Gating semantics (Pitfall 3): gated branches (switch off / customer missing / no attribution / owner not a distributor / no rate configured)
// always return nil - a gated error would turn "no bookkeeping needed" into "the entire top-up transaction rolls back"
// (the user has paid but receives no quota); only a genuine DB error writing the flow returns an error.
//
// Idempotency (contract §5 frozen): backed by the uk_flow_billing(billing_no, flow_type) unique index;
// no distributed lock, Redis lock, or "check-then-insert" two-step logic is allowed.
func RecordCommissionTx(tx *gorm.DB, topUp *TopUp) error {
	// 1. Master switch gate (contract §3.4; see common/option_vars.go for the hot-update path)
	if !common.CommissionEnabled {
		return nil
	}
	// 2. Customer attribution: query inviter_id directly (following attribution.go:66-71; only tx is used)
	var customer User
	if err := tx.Select("id, inviter_id").First(&customer, "id = ?", topUp.UserId).Error; err != nil {
		return nil // Customer missing (abnormal data) -> no bookkeeping, no blocking
	}
	if customer.InviterId == 0 {
		return nil // No attribution (ATTR-03)
	}
	// 3. The owner must be a distributor: exact == check, no >=5 threshold (otherwise admin=10/root=100 would trigger bookkeeping, T-03-01-02)
	var inviter User
	if err := tx.Select("id, role").First(&inviter, "id = ?", customer.InviterId).Error; err != nil {
		return nil
	}
	if inviter.Role != common.RoleDistributorUser {
		return nil // Regular invitations remain as before, with no commission flow (ATTR-04)
	}
	// 4. Rate snapshot: no config = not enabled, return nil without creating a flow (contract §7#3 frozen, no zero-amount flows)
	var rate CommissionRate
	if err := tx.Select("id, distributor_id, rate_bp").First(&rate, "distributor_id = ?", inviter.Id).Error; err != nil {
		return nil
	}
	// 5. Amount integerization: a single math.Round float->int conversion and snapshot (contract §3.0;
	//    commission base = TopUp.Money paid amount (F4); everything downstream is int64, no floats)
	moneyCents := int64(math.Round(topUp.Money * 100))
	flow := &CommissionFlow{
		BillingNo:       topUp.TradeNo, // Idempotency key (contract §3.1.1: commission uses TradeNo)
		FlowType:        CommissionFlowCommission,
		TradeNo:         topUp.TradeNo,
		CustomerId:      customer.Id,
		DistributorId:   inviter.Id,
		TopupMoneyCents: moneyCents,
		RateBp:          rate.RateBp,
		AmountCents:     moneyCents * int64(rate.RateBp) / 10000, // int64 integer division rounds down; the remainder stays with the platform (contract §3.0)
		Status:          CommissionFlowPending,                   // Flipped to available at Phase 4 statement generation
		// Period follows the server-local calendar month (contract C3, consistent with the checkin.go:24 precedent;
		// operations must keep the TZ config stable; docker-compose already sets TZ=Asia/Shanghai).
		// No CompleteTime==0 fallback - contract C2 decided that EpayNotify backfills it at the source.
		Period: time.Unix(topUp.CompleteTime, 0).Format("2006-01"),
	}
	// 6. Write the flow: idempotency backed by the uk_flow_billing(billing_no, flow_type) unique index (contract §5 frozen)
	if err := tx.Create(flow).Error; err != nil {
		return err // Only a genuine DB error writing the flow returns an error (on replay pass-through this is the unique index conflict)
	}
	return nil
}
