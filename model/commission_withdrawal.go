package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Per-order withdrawal entries in domain C (contract 01-CONTRACT.md appendix E frozen, v1.4; 05-CONTEXT D-04 to D-14).
//
// Five-state machine (user ruling Q1=B): pending -> reviewing -> approved -> paid / rejected;
// Rejection can only start from reviewing (Q2=B); rejected/paid are terminal states.
//
// Deduplication design (appendix E4 frozen, with no unique index approach): conditional statement status machine update
// (payable -> withdrawing flip in the same transaction + RowsAffected==1 assertion) + statement row
// FOR UPDATE serialization - the statement status itself is the DB-level dedup state bit; a statement_id
// unique index is forbidden (§6 forbids partial indexes, and full-column uniqueness blocks re-application after a D-06 rejection).
//
// Write-path discipline: statement row updates always use conditional Updates (previous state in WHERE; never write back whole objects),
// RowsAffected != 1 -> return error and roll back everything (no half states, Pitfall 2);
// rejection rollback/payment settlement and withdrawal state transitions share one transaction (contract L308 forbids splitting linked operations).
// Gating semantics (D-14/Q5=A): the entire withdrawal path has no gated branch - the switch only governs new bookkeeping.
// Lock discipline: the statement row lock is the primary lock for all five withdrawal write paths (following the A8 master pattern).

// Withdrawal Status enum (appendix E1 frozen five values).
const (
	WithdrawalPending   = "pending"   // Requested (D-04)
	WithdrawalReviewing = "reviewing" // Under review (after acceptance)
	WithdrawalApproved  = "approved"  // Approved, awaiting payment (new state per Q1=B)
	WithdrawalPaid      = "paid"      // Paid (terminal; voucher number required)
	WithdrawalRejected  = "rejected"  // Rejected (terminal; reason required, D-06)
)

// CommissionWithdrawal is the withdrawal table (appendix E1 frozen shape).
//
// No amount field (D-08): the requested amount = statement.settle_amount_cents, read live via JOIN at query time;
// the five timeline columns (created_at/reviewed_at/approved_at/paid_at/rejected_at) support SC4 transition queries.
type CommissionWithdrawal struct {
	Id            int64  `json:"id" gorm:"primaryKey"`
	WithdrawalNo  string `json:"withdrawal_no" gorm:"type:varchar(64);uniqueIndex:uk_wd_no,unique"` // WD-{yyyyMMddHHmmss}-{4 random chars}
	StatementId   int64  `json:"statement_id" gorm:"type:bigint;index:idx_wd_statement"`
	DistributorId int    `json:"distributor_id" gorm:"type:int;index:idx_wd_distributor"`
	Status        string `json:"status" gorm:"type:varchar(20);index:idx_wd_status"`
	Reason        string `json:"reason" gorm:"type:varchar(255)"`                    // Rejection reason (required only for rejected)
	VoucherNo     string `json:"voucher_no" gorm:"type:varchar(255)"`                // Payment voucher number (required only for paid)
	OperatorId    int    `json:"operator_id" gorm:"type:int;default:0"`              // Last operating admin (0 at request time)
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"` // Request time
	ReviewedAt    int64  `json:"reviewed_at" gorm:"column:reviewed_at"`
	ApprovedAt    int64  `json:"approved_at" gorm:"column:approved_at"`
	PaidAt        int64  `json:"paid_at" gorm:"column:paid_at"`
	RejectedAt    int64  `json:"rejected_at" gorm:"column:rejected_at"`
}

// TableName explicitly specifies the table name (appendix E1 frozen).
func (CommissionWithdrawal) TableName() string { return "commission_withdrawals" }

// ErrWithdrawalNotOwner is the statement ownership sentinel (the controller maps it to 403; T-05-01-01).
var ErrWithdrawalNotOwner = errors.New("You are not allowed to operate on this statement")

// CreateWithdrawalTx is the A15 request transaction (WDRAW-01/SC1, following the RESEARCH Example 1 authoritative skeleton).
// (1) statement row FOR UPDATE -> (2) ownership check -> (3) payable whitelist (distinct messages, Pitfall 6) ->
// (4) INSERT pending (D-08, no amount field) -> (5) conditional statement flip + RowsAffected assertion.
func CreateWithdrawalTx(statementId int64, distributorId int) (*CommissionWithdrawal, error) {
	var created *CommissionWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		stmt := &CommissionStatement{}
		// (1) statement row lock: serializes request/adjustment/rejection rollback/payment settlement for the same statement (appendix E4)
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(stmt, "id = ?", statementId).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New("Statement not found")
			}
			return err
		}
		// (2) Ownership check (WDRAW-01/CUST-05; middleware only guarantees "is a distributor"; this guarantees "is the distributor of this statement")
		if stmt.DistributorId != distributorId {
			return ErrWithdrawalNotOwner
		}
		// (3) Whitelist: only payable passes (no blacklist-style checks; distinct messages help distributors understand)
		if stmt.Status != StatementPayable {
			switch stmt.Status {
			case StatementWithdrawing:
				return errors.New("The statement is being withdrawn; duplicate requests are not allowed")
			case StatementSettled:
				return errors.New("The statement is already settled")
			default:
				return fmt.Errorf("Statement status %q does not allow a withdrawal request", stmt.Status)
			}
		}
		// (4) Create the entry (D-08: no amount field; amount read live from the statement via JOIN)
		wd := &CommissionWithdrawal{
			WithdrawalNo:  fmt.Sprintf("WD-%s-%s", time.Now().Format("20060102150405"), common.GetRandomString(4)),
			StatementId:   statementId,
			DistributorId: distributorId,
			Status:        WithdrawalPending,
		}
		if err := tx.Create(wd).Error; err != nil {
			return err
		}
		// (5) Statement linkage (D-05): conditional update + assertion; the previous state in WHERE also acts as the whitelist
		res := tx.Model(&CommissionStatement{}).
			Where("id = ? AND status = ?", statementId, StatementPayable).
			Updates(map[string]any{"status": StatementWithdrawing})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("statement status flip mismatch: want 1 got %d", res.RowsAffected)
		}
		created = wd
		return nil
	})
	return created, err
}

// AcceptWithdrawalTx is the A18 acceptance (pending -> reviewing, no statement operation - D-05 keeps withdrawing).
func AcceptWithdrawalTx(id int64, operatorId int) (*CommissionWithdrawal, error) {
	var updated *CommissionWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		wd, err := lockWithdrawal(tx, id)
		if err != nil {
			return err
		}
		if wd.Status != WithdrawalPending {
			return fmt.Errorf("Withdrawal status %q cannot be accepted (only requested withdrawals can be accepted)", wd.Status)
		}
		if err := updateWithdrawalStatus(tx, id, WithdrawalPending, map[string]any{
			"status":      WithdrawalReviewing,
			"reviewed_at": common.GetTimestamp(),
			"operator_id": operatorId,
		}); err != nil {
			return err
		}
		wd.Status = WithdrawalReviewing
		updated = wd
		return nil
	})
	return updated, err
}

// RejectWithdrawalTx is the A19 rejection (only from reviewing, Q2=B; terminal state keeps the reason, D-06) +
// same-transaction statement rollback withdrawing -> payable (D-05; a missing assertion means a permanently stuck withdrawing entry, Pitfall 2).
// Returns the transitioned withdrawal row (the controller audits outside the transaction embedding withdrawal_no, consistent with the other three admin Tx signatures).
func RejectWithdrawalTx(id int64, reason string, operatorId int) (*CommissionWithdrawal, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("Rejection reason is required") // Model-side safety (the controller already blocks earlier)
	}
	if utf8.RuneCountInString(reason) > 255 {
		return nil, errors.New("Rejection reason must not exceed 255 characters")
	}
	var updated *CommissionWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		wd, err := lockWithdrawal(tx, id)
		if err != nil {
			return err
		}
		if wd.Status != WithdrawalReviewing {
			return fmt.Errorf("Only withdrawals under review can be rejected; accept it first (current status %q)", wd.Status)
		}
		if err := updateWithdrawalStatus(tx, id, WithdrawalReviewing, map[string]any{
			"status":      WithdrawalRejected,
			"reason":      reason,
			"rejected_at": common.GetTimestamp(),
			"operator_id": operatorId,
		}); err != nil {
			return err
		}
		// Statement rollback (D-05): same-transaction conditional update + assertion
		if err := flipStatementStatus(tx, wd.StatementId, StatementWithdrawing, StatementPayable); err != nil {
			return err
		}
		wd.Status = WithdrawalRejected
		wd.Reason = reason
		updated = wd
		return nil
	})
	return updated, err
}

// ApproveWithdrawalTx is the A20 approval (reviewing -> approved, no statement operation - D-12 keeps withdrawing during approved).
func ApproveWithdrawalTx(id int64, operatorId int) (*CommissionWithdrawal, error) {
	var updated *CommissionWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		wd, err := lockWithdrawal(tx, id)
		if err != nil {
			return err
		}
		if wd.Status != WithdrawalReviewing {
			return fmt.Errorf("Withdrawal status %q cannot be approved (only reviewed withdrawals can be approved)", wd.Status)
		}
		if err := updateWithdrawalStatus(tx, id, WithdrawalReviewing, map[string]any{
			"status":      WithdrawalApproved,
			"approved_at": common.GetTimestamp(),
			"operator_id": operatorId,
		}); err != nil {
			return err
		}
		wd.Status = WithdrawalApproved
		updated = wd
		return nil
	})
	return updated, err
}

// MarkWithdrawalPaidTx is the A21 payment registration (only from approved, Q1=B; voucher_no required and <= 255) +
// same-transaction statement settlement withdrawing -> settled (D-05).
func MarkWithdrawalPaidTx(id int64, voucherNo string, operatorId int) (*CommissionWithdrawal, error) {
	voucherNo = strings.TrimSpace(voucherNo)
	if voucherNo == "" {
		return nil, errors.New("Payment voucher number is required")
	}
	if utf8.RuneCountInString(voucherNo) > 255 {
		return nil, errors.New("Payment voucher number must not exceed 255 characters")
	}
	var updated *CommissionWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		wd, err := lockWithdrawal(tx, id)
		if err != nil {
			return err
		}
		if wd.Status != WithdrawalApproved {
			return fmt.Errorf("Withdrawal status %q cannot be marked as paid (only approved withdrawals can be paid)", wd.Status)
		}
		if err := updateWithdrawalStatus(tx, id, WithdrawalApproved, map[string]any{
			"status":      WithdrawalPaid,
			"voucher_no":  voucherNo,
			"paid_at":     common.GetTimestamp(),
			"operator_id": operatorId,
		}); err != nil {
			return err
		}
		// Statement settlement (D-05): same-transaction conditional update + assertion
		if err := flipStatementStatus(tx, wd.StatementId, StatementWithdrawing, StatementSettled); err != nil {
			return err
		}
		wd.Status = WithdrawalPaid
		updated = wd
		return nil
	})
	return updated, err
}

// lockWithdrawal locks the withdrawal row (serializes concurrent admin actions on the same entry: accept/reject/approve/pay are mutually exclusive).
func lockWithdrawal(tx *gorm.DB, id int64) (*CommissionWithdrawal, error) {
	wd := &CommissionWithdrawal{}
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(wd, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Withdrawal not found")
		}
		return nil, err
	}
	return wd, nil
}

// updateWithdrawalStatus transitions a withdrawal conditionally: the previous state in WHERE also acts as the whitelist,
// RowsAffected != 1 -> explicit error and full rollback (no silent handling).
func updateWithdrawalStatus(tx *gorm.DB, id int64, fromStatus string, updates map[string]any) error {
	res := tx.Model(&CommissionWithdrawal{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("Withdrawal status has changed, please refresh and retry")
	}
	return nil
}

// flipStatementStatus flips the statement conditionally (shared by rejection rollback/payment settlement): previous state in WHERE +
// RowsAffected assertion; on failure roll back everything (Pitfall 2 forbids a missing assertion that leaves a stuck entry).
func flipStatementStatus(tx *gorm.DB, statementId int64, fromStatus, toStatus string) error {
	res := tx.Model(&CommissionStatement{}).
		Where("id = ? AND status = ?", statementId, fromStatus).
		Updates(map[string]any{"status": toStatus})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("statement flip mismatch: want 1 got %d", res.RowsAffected)
	}
	return nil
}

// WithdrawalDetail is an A16/A17 list item: withdrawal embedded flat + redundant username / period / settle_amount_cents
// (D-08 reads the amount live from the statement; 03-02 ListRates flat-embedding precedent).
type WithdrawalDetail struct {
	CommissionWithdrawal
	Username          string `json:"username"`
	Period            string `json:"period"`
	SettleAmountCents int64  `json:"settle_amount_cents"`
}

// ListWithdrawals paginates the withdrawal list (appendix E2 frozen shape).
// selfId > 0 (A16): the server forces WHERE distributor_id = selfId and ignores the distributorId filter argument
// (never trust the client, ROLE-02/CUST-05); selfId = 0 (A17): distributorId/status are optional filters (0/empty string skips the predicate).
// Double LEFT JOIN for the redundant username (users) and period/settle_amount_cents (statement); ordered by id DESC (newest requests first).
func ListWithdrawals(selfId int, distributorId int, status string, page, pageSize int) ([]WithdrawalDetail, int64, error) {
	query := DB.Model(&CommissionWithdrawal{})
	if selfId != 0 {
		query = query.Where("commission_withdrawals.distributor_id = ?", selfId)
	} else if distributorId != 0 {
		query = query.Where("commission_withdrawals.distributor_id = ?", distributorId)
	}
	if status != "" {
		query = query.Where("commission_withdrawals.status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	details := make([]WithdrawalDetail, 0) // Empty results serialize as [] instead of null (04-02 A14 precedent)
	if err := query.
		Select("commission_withdrawals.*, users.username, commission_statements.period, commission_statements.settle_amount_cents").
		Joins("LEFT JOIN users ON users.id = commission_withdrawals.distributor_id").
		Joins("LEFT JOIN commission_statements ON commission_statements.id = commission_withdrawals.statement_id").
		Order("commission_withdrawals.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&details).Error; err != nil {
		return nil, 0, err
	}
	return details, total, nil
}
