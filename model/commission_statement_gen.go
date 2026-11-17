package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Monthly statement generation engine (contract 01-CONTRACT.md §3.1.2/§3.3/§5 + appendix D2/D5.1, STMT-01/02/03).
//
// Generation semantics: statementPeriod = the calendar month before now (first-of-month minus one day); for each distributor that has pending flows
// (period <= statementPeriod) and no statement for that period, generate one statement in an independent transaction
// and flip the selected flows to available + backfill statement_id.
//
// Three dedup layers (contract §5/D2 frozen):
//  1. Scheduling-layer IsMasterNode gate (service layer, blocks slave nodes)
//  2. uk_stmt_period(distributor_id, period) unique index (final DB-level defense)
//  3. FOR UPDATE on the distributor's users row (serializes competition for the same distributor within the transaction)
//
// Boundary discipline (research ruling A1): statement boundaries always filter on the period field, never on creation-time columns -
// a cross-month reversal copies the original flow's Period (see the frozen comment in commission_reversal.go); filtering by flow creation time
// would split the cross-month reversal and the original flow across two statements. period is varchar(7) compared lexicographically, works on all three databases,
// and uses the idx_flow_period index.
//
// Gating discipline (appendix D5.1): statement generation is not gated by the commission master switch - correction and reconciliation are ledger-integrity operations;
// the switch only controls new bookkeeping entry points.

// statementPeriodOf computes the statement period label = the calendar month before now's month ("2006-01").
// First-of-month minus one day: take the 1st of the current month, subtract one day, then use that month.
// AddDate(0,-1,0) is forbidden: Go normalizes non-existent target-month dates by carrying over (3/31 -> 2/31 -> 3/3),
// producing a "2006-03" (current month) label for March 31 (04-RESEARCH Pitfall 1).
func statementPeriodOf(now time.Time) string {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -1).Format("2006-01")
}

// GenerateDueStatements is the statement generation entry point (now is parameterized for tests and catch-up runs).
// Generates one transaction per distributor; a single distributor failure aborts the batch and returns an error (already committed distributors are not rolled back -
// partial success is valid, and the next call resumes via the idempotent pre-check); skips/conflicts/empty flows are not counted in the return value.
// Returns the number of statements actually created in this run.
func GenerateDueStatements(now time.Time) (int, error) {
	stmtPeriod := statementPeriodOf(now)

	// Candidate distributors: those with pending flows (acceptance criterion 1 "distributors with available commission", Pitfall 5).
	// Filter on the period field (ruling A1; no creation-time column check).
	var distIds []int
	if err := DB.Model(&CommissionFlow{}).
		Where("status = ? AND period <= ?", CommissionFlowPending, stmtPeriod).
		Distinct("distributor_id").Pluck("distributor_id", &distIds).Error; err != nil {
		return 0, err
	}

	generated := 0
	for _, distId := range distIds {
		created := false
		err := DB.Transaction(func(tx *gorm.DB) error {
			var txErr error
			created, txErr = generateStatementTx(tx, distId, stmtPeriod)
			return txErr
		})
		if err != nil {
			return generated, err
		}
		if created {
			generated++
		}
	}
	return generated, nil
}

// generateStatementTx is the seven-step statement transaction for one distributor (only use the passed-in tx, never the global DB -
// bypassing the global handle inside a transaction breaks atomicity and deadlocks on a single-connection test DB; same root as Phase 3 Pitfall 2/7).
// Returns created=true only when a statement was actually created; pre-check hit/empty flows/unique index conflict all return (false, nil).
func generateStatementTx(tx *gorm.DB, distributorId int, stmtPeriod string) (bool, error) {
	// (1) Distributor row lock (contract §5 frozen: the statement task locks the target distributor row to serialize;
	//    SQLite's single writer is naturally serialized; Set dialect follows the topup.go/commission_reversal.go precedent)
	var lockUser User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Select("id").First(&lockUser, "id = ?", distributorId).Error; err != nil {
		return false, nil // Distributor missing/deleted or similar anomaly -> skip without aborting the batch
	}

	// (2) Idempotent pre-check: already statemented for this period -> skip (the uk_stmt_period unique index is the final defense)
	var cnt int64
	if err := tx.Model(&CommissionStatement{}).
		Where("distributor_id = ? AND period = ?", distributorId, stmtPeriod).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	if cnt > 0 {
		return false, nil
	}

	// (3) Read pending flows: filter on period (ruling A1; no creation-time column check), ordered stably by id
	var flows []CommissionFlow
	if err := tx.Where("distributor_id = ? AND status = ? AND period <= ?",
		distributorId, CommissionFlowPending, stmtPeriod).
		Order("id ASC").Find(&flows).Error; err != nil {
		return false, err
	}
	if len(flows) == 0 {
		return false, nil // Statemented by another node within the race window between the candidate query and the row lock -> no empty statement (Pitfall 5)
	}

	// (4) Batch-fill the TopUp snapshot: PaymentMethod/PaymentProvider/CompleteTime exist only in top_ups
	//    (STMT-02); afterwards items never read top_ups again (locked-on-creation semantics)
	tradeNos := make([]string, 0, len(flows))
	for _, f := range flows {
		if f.TradeNo != "" {
			tradeNos = append(tradeNos, f.TradeNo)
		}
	}
	topupMap := make(map[string]TopUp, len(tradeNos))
	if len(tradeNos) > 0 {
		var topups []TopUp
		if err := tx.Where("trade_no IN ?", tradeNos).
			Select("trade_no, payment_method, payment_provider, complete_time").
			Find(&topups).Error; err != nil {
			return false, err
		}
		for _, tp := range topups {
			topupMap[tp.TradeNo] = tp
		}
	}

	// (5) Sum totals + INSERT statement (AmountCents includes negative reversals that offset naturally;
	//    SettleAmountCents = TotalCommission + AdjustedCents (initially 0))
	var totalTopup, totalCommission int64
	for _, f := range flows {
		totalTopup += f.TopupMoneyCents
		totalCommission += f.AmountCents
	}
	stmt := &CommissionStatement{
		DistributorId:        distributorId,
		Period:               stmtPeriod,
		TotalTopupCents:      totalTopup,
		TotalCommissionCents: totalCommission,
		SettleAmountCents:    totalCommission,
		Status:               StatementPayable,
		LockedAt:             common.GetTimestamp(), // Locked on creation (STMT-03)
	}
	if err := tx.Create(stmt).Error; err != nil {
		// uk_stmt_period conflict = another node already generated the statement (Pitfall 4): log at Info and skip, no retry
		common.SysLog(fmt.Sprintf("commission statement create conflict (already statemented by another node): node=%s distributor_id=%d period=%s err=%v",
			common.NodeName, distributorId, stmtPeriod, err))
		return false, nil
	}

	// (6) One item per flow (including negative reversals and manual flows, no netting, FlowId traceable);
	//    manual flows (trade_no="") have no TopUp row -> defaults (appendix D5.2):
	//    PaymentMethod="manual", PaymentProvider="", CompleteTime=flow bookkeeping time
	items := make([]CommissionStatementItem, 0, len(flows))
	for _, f := range flows {
		item := CommissionStatementItem{
			StatementId:     stmt.Id,
			FlowId:          f.Id,
			BillingNo:       f.BillingNo,
			TopupMoneyCents: f.TopupMoneyCents,
			CommissionCents: f.AmountCents,
			CustomerId:      f.CustomerId,
		}
		if tp, ok := topupMap[f.TradeNo]; ok {
			item.PaymentMethod = tp.PaymentMethod
			item.PaymentProvider = tp.PaymentProvider
			item.CompleteTime = tp.CompleteTime
		} else if f.TradeNo == "" {
			item.PaymentMethod = "manual"
			item.PaymentProvider = ""
			item.CompleteTime = f.CreatedAt
		}
		items = append(items, item)
	}
	if err := tx.CreateInBatches(items, 500).Error; err != nil {
		return false, err
	}

	// (7) Flip by the ID list selected (Pitfall 3: do not re-match with a distributor_id+status predicate -
	//    flows committed concurrently between the SELECT and UPDATE would be wrongly flipped into orphan flows without a statement);
	//    assert RowsAffected == len(ids); otherwise roll everything back
	ids := make([]int64, 0, len(flows))
	for _, f := range flows {
		ids = append(ids, f.Id)
	}
	res := tx.Model(&CommissionFlow{}).Where("id IN ?", ids).
		Updates(map[string]any{"status": CommissionFlowAvailable, "statement_id": stmt.Id})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected != int64(len(ids)) {
		return false, fmt.Errorf("commission flow flip mismatch: want %d got %d (distributor_id=%d period=%s)",
			len(ids), res.RowsAffected, distributorId, stmtPeriod)
	}

	// Reconciliation check (acceptance criterion 1): Go-side double identity assertion before commit; on mismatch -> return error and roll back everything
	var sumCommission, sumTopup int64
	for _, it := range items {
		sumCommission += it.CommissionCents
		sumTopup += it.TopupMoneyCents
	}
	if len(items) != len(flows) || sumCommission != stmt.TotalCommissionCents || sumTopup != stmt.TotalTopupCents {
		return false, fmt.Errorf("commission statement reconciliation mismatch: distributor_id=%d period=%s len(items)=%d len(flows)=%d sum_commission=%d total_commission=%d sum_topup=%d total_topup=%d",
			distributorId, stmtPeriod, len(items), len(flows), sumCommission, stmt.TotalCommissionCents, sumTopup, stmt.TotalTopupCents)
	}
	return true, nil
}
