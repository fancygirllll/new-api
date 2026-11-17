package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

// Statement adjustment entry (contract 01-CONTRACT.md §4.3 A8 + §3.3 immutability rules + appendix D5.1, COMM-06/07).
//
// Transaction semantics (authoritative skeleton from RESEARCH Example 3): statement row FOR UPDATE serializes concurrent adjustments ->
// whitelist status==payable (reject withdrawing/settled too, COMM-07; no threshold == settled
// check, since future enum values are rejected by the whitelist by default) -> same-transaction INSERT adjustment + accumulate
// adjusted_cents and recompute settle_amount_cents (computed server-side, never trusting the client; read-then-compute under the row lock is safe).
//
// The original statement is unchanged (COMM-06): the only columns written here are adjusted_cents and settle_amount_cents;
// Total fields and details are untouched - immutability is achieved through "no write path" (§3.3 frozen; the only legal writes are
// adjusted_cents and status).
//
// Gating semantics (appendix D5.1 frozen): the adjustment entry is not gated by the commission master switch - correction is a ledger-integrity operation;
// this function has no gated branch.

// CreateStatementAdjustmentTx creates an adjustment for a payable statement and accumulates the difference in the same transaction.
// Model-side safety for the two-layer reason validation (the controller validates first).
func CreateStatementAdjustmentTx(statementId int64, deltaCents int64, reason string, operatorId int) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("Adjustment reason is required")
	}
	if utf8.RuneCountInString(reason) > 255 {
		return errors.New("Adjustment reason must not exceed 255 characters")
	}
	if deltaCents == 0 {
		return errors.New("Adjustment difference cannot be zero (meaningless adjustments are rejected)")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		stmt := &CommissionStatement{}
		// Statement row lock: serializes concurrent adjustments to the same statement (T-04-03-07)
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(stmt, "id = ?", statementId).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New("Statement not found")
			}
			return err
		}
		// Whitelist: only payable passes (Pitfall 6/COMM-07); messages distinguish statuses for audit readability
		if stmt.Status != StatementPayable {
			switch stmt.Status {
			case StatementWithdrawing:
				return errors.New("The statement is being withdrawn; adjustments are not allowed")
			case StatementSettled:
				return errors.New("The statement is settled; adjustments are not allowed (corrections appear in the next period via manual entry)")
			default:
				return fmt.Errorf("Statement status %q does not allow adjustments (only payable statements can be adjusted)", stmt.Status)
			}
		}
		if err := tx.Create(&StatementAdjustment{
			StatementId: statementId,
			DeltaCents:  deltaCents,
			Reason:      reason,
			OperatorId:  operatorId,
		}).Error; err != nil {
			return err
		}
		newAdjusted := stmt.AdjustedCents + deltaCents
		return tx.Model(&CommissionStatement{}).Where("id = ?", statementId).
			Updates(map[string]any{
				"adjusted_cents":      newAdjusted,
				"settle_amount_cents": stmt.TotalCommissionCents + newAdjusted, // Settle using the adjusted amount
			}).Error
	})
}
