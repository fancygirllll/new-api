package model

import (
	"time"
)

// Distributor commission dashboard aggregation (06-01; contract 01-CONTRACT.md §4.3 A9/A12 frozen shape + v1.5 appendix F2 count extension).
// All values are aggregated server-side (int64 cents, contract §3.0 precision decision); the frontend performs no amount recomputation (D-04 user ruling).
// Four metrics (OQ-1/D-09 ruling) and the four-card reconciliation identity: statemented = withdrawn + pending withdrawal + withdrawing.
// This file is the sole location for A9 aggregation (aggregate SQL must not be added to distributor.go, keeping the aggregation point count unique).

// CommissionDashboard is the A9 response body: four frozen cents fields from §4.3 (names unchanged) + four additive count fields from appendix F2.
type CommissionDashboard struct {
	CurrentPendingCents  int64 `json:"current_pending_cents"`
	TotalStatementCents  int64 `json:"total_statement_cents"`
	TotalWithdrawnCents  int64 `json:"total_withdrawn_cents"`
	PendingWithdrawCents int64 `json:"pending_withdraw_cents"`

	CurrentPendingCount  int64 `json:"current_pending_count"`
	TotalStatementCount  int64 `json:"total_statement_count"`
	TotalWithdrawnCount  int64 `json:"total_withdrawn_count"`
	PendingWithdrawCount int64 `json:"pending_withdraw_count"`
}

// dashboardAggregateRow is an aggregate query row: amount totals + counts (total columns wrapped to avoid NULL serialization).
type dashboardAggregateRow struct {
	Total int64
	Cnt   int64
}

// GetCommissionDashboard runs four independent A9 aggregations (readable and easy to assert; follows the B4 aggregation precedent).
// Status predicates always use status constants, never string literals; JOIN clauses are static literals, never concatenated input (T-06-01-02).
// (1) Current unstatemented: sum commission flow amounts for pending flows only (negative reversals offset naturally; same predicate as A12)
// (2) Historical statemented: sum settlement amounts across all statement statuses (including final payable adjustments, OQ-1 settle basis)
// (3) Withdrawn: sum the settlement amount of the related statement for paid withdrawals only (amount read live from the statement; appendix E D-08 has no amount column)
// (4) Pending withdrawal: sum settlement amounts for payable statements only
func GetCommissionDashboard(distributorId int) (*CommissionDashboard, error) {
	d := &CommissionDashboard{}

	// (1) Current unstatemented
	var flowRow dashboardAggregateRow
	if err := DB.Model(&CommissionFlow{}).
		Select("COALESCE(SUM(amount_cents),0) AS total, COUNT(*) AS cnt").
		Where("distributor_id = ? AND status = ?", distributorId, CommissionFlowPending).
		Scan(&flowRow).Error; err != nil {
		return nil, err
	}
	d.CurrentPendingCents = flowRow.Total
	d.CurrentPendingCount = flowRow.Cnt

	// (2) Historical statemented (all statement statuses, settle basis)
	var stmtRow dashboardAggregateRow
	if err := DB.Model(&CommissionStatement{}).
		Select("COALESCE(SUM(settle_amount_cents),0) AS total, COUNT(*) AS cnt").
		Where("distributor_id = ?", distributorId).
		Scan(&stmtRow).Error; err != nil {
		return nil, err
	}
	d.TotalStatementCents = stmtRow.Total
	d.TotalStatementCount = stmtRow.Cnt

	// (3) Withdrawn (paid withdrawals JOIN statements for the settlement amount)
	var wdRow dashboardAggregateRow
	if err := DB.Model(&CommissionWithdrawal{}).
		Select("COALESCE(SUM(commission_statements.settle_amount_cents),0) AS total, COUNT(*) AS cnt").
		Joins("JOIN commission_statements ON commission_statements.id = commission_withdrawals.statement_id").
		Where("commission_withdrawals.distributor_id = ? AND commission_withdrawals.status = ?", distributorId, WithdrawalPaid).
		Scan(&wdRow).Error; err != nil {
		return nil, err
	}
	d.TotalWithdrawnCents = wdRow.Total
	d.TotalWithdrawnCount = wdRow.Cnt

	// (4) Pending withdrawal (payable statements only)
	var payableRow dashboardAggregateRow
	if err := DB.Model(&CommissionStatement{}).
		Select("COALESCE(SUM(settle_amount_cents),0) AS total, COUNT(*) AS cnt").
		Where("distributor_id = ? AND status = ?", distributorId, StatementPayable).
		Scan(&payableRow).Error; err != nil {
		return nil, err
	}
	d.PendingWithdrawCents = payableRow.Total
	d.PendingWithdrawCount = payableRow.Cnt

	return d, nil
}

// CurrentPendingSummary is the A12 response body (§4.3 three frozen fields verbatim; appendix F3 confirms no extension).
type CurrentPendingSummary struct {
	Period       string `json:"period"`
	PendingFlows int64  `json:"pending_flows"`
	PendingCents int64  `json:"pending_cents"`
}

// GetCurrentPending is the A12 live unstatemented preview (same predicate and source as A9-(1); period is the server's current calendar month,
// with the time zone baseline TZ=Asia/Shanghai from docker-compose, following the commission_manual.go precedent).
func GetCurrentPending(distributorId int) (*CurrentPendingSummary, error) {
	var row dashboardAggregateRow
	if err := DB.Model(&CommissionFlow{}).
		Select("COALESCE(SUM(amount_cents),0) AS total, COUNT(*) AS cnt").
		Where("distributor_id = ? AND status = ?", distributorId, CommissionFlowPending).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	return &CurrentPendingSummary{
		Period:       time.Now().Format("2006-01"),
		PendingFlows: row.Cnt,
		PendingCents: row.Total,
	}, nil
}
