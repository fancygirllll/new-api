package model

// Statement query surface (04-02; contract 01-CONTRACT.md §4.3 A4/A5/A6 + appendix D3 A14 frozen shape).
// All amounts are int64 cents (contract §3.0 forbids floating-point math); aggregate predicates do not depend on time columns (consistent with the 04-01 A1 ruling).

// StatementItemDetail is an A6 item list entry: CommissionStatementItem embedded flat + redundant customer username
// (contract §4.3 A6; JOIN precedent model/commission_rate.go ListRates).
// RateBp is a v1.5 appendix F1 addition: rate snapshot (basis points); 0 when there is no related flow.
type StatementItemDetail struct {
	CommissionStatementItem
	Username string `json:"username"`
	RateBp   int    `json:"rate_bp"`
}

// ListStatements paginates the admin statement list (contract §4.3 A4).
// Three optional filters: skip the predicate when distributorId=0, period or status is an empty string.
// Order by period DESC, distributor_id ASC (newest period first, then by distributor); COUNT uses the same predicate as the query.
func ListStatements(distributorId int, period, status string, page, pageSize int) ([]CommissionStatement, int64, error) {
	query := DB.Model(&CommissionStatement{})
	if distributorId != 0 {
		query = query.Where("distributor_id = ?", distributorId)
	}
	if period != "" {
		query = query.Where("period = ?", period)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var statements []CommissionStatement
	if err := query.Order("period DESC, distributor_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&statements).Error; err != nil {
		return nil, 0, err
	}
	return statements, total, nil
}

// GetStatementWithAdjustments is the A5 statement detail: the statement itself + the adjustments array (contract §4.3 A5).
// ErrRecordNotFound is passed through; the controller converts emptiness into business semantics.
func GetStatementWithAdjustments(id int64) (*CommissionStatement, []StatementAdjustment, error) {
	var stmt CommissionStatement
	if err := DB.First(&stmt, "id = ?", id).Error; err != nil {
		return nil, nil, err
	}
	var adjustments []StatementAdjustment
	if err := DB.Where("statement_id = ?", id).Order("id ASC").Find(&adjustments).Error; err != nil {
		return nil, nil, err
	}
	return &stmt, adjustments, nil
}

// ListStatementItems paginates statement items (contract §4.3 A6).
// LEFT JOIN users for the redundant customer username (via customer_id); v1.5 appendix F1 addition
// Join the commission flow table for the redundant rate snapshot (via flow_id; 0 when no row matches).
// Clean COUNT + pagination, ordered by id ASC; one change applies the same shape to both A6/A11 endpoints.
func ListStatementItems(statementId int64, page, pageSize int) ([]StatementItemDetail, int64, error) {
	var total int64
	if err := DB.Model(&CommissionStatementItem{}).
		Where("statement_id = ?", statementId).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var details []StatementItemDetail
	if err := DB.Model(&CommissionStatementItem{}).
		Select("commission_statement_items.*, users.username, commission_flows.rate_bp").
		Joins("LEFT JOIN users ON users.id = commission_statement_items.customer_id").
		Joins("LEFT JOIN commission_flows ON commission_flows.id = commission_statement_items.flow_id").
		Where("commission_statement_items.statement_id = ?", statementId).
		Order("commission_statement_items.id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&details).Error; err != nil {
		return nil, 0, err
	}
	return details, total, nil
}

// StatementSummaryItem is an A14 summary list item (contract appendix D3 frozen field names).
type StatementSummaryItem struct {
	DistributorId        int    `json:"distributor_id"`
	Username             string `json:"username"`
	TotalCommissionCents int64  `json:"total_commission_cents"`
	AdjustedCents        int64  `json:"adjusted_cents"`
	StatementStatus      string `json:"statement_status"`
	StatementId          int64  `json:"statement_id"`
}

// StatementSummaryTotals is the A14 summary totals (accumulated in Go, D3).
type StatementSummaryTotals struct {
	TotalCommissionCents int64 `json:"total_commission_cents"`
	AdjustedCents        int64 `json:"adjusted_cents"`
	DistributorCount     int   `json:"distributor_count"`
}

// StatementSummary is the A14 summary response body (contract appendix D3 frozen shape).
type StatementSummary struct {
	Period string                 `json:"period"`
	Items  []StatementSummaryItem `json:"items"`
	Totals StatementSummaryTotals `json:"totals"`
}

// SummarizeStatements is the A14 cross-distributor summary (contract appendix D3: GROUP BY aggregating all statements for the period).
// uk_stmt_period guarantees at most one row per (distributor, period), so each row is one distributor;
// detail columns come directly from the statement snapshot columns (no second aggregation); totals are accumulated in Go (exact int64).
// Ordered by total_commission_cents DESC (largest first, the usual admin reporting convention).
func SummarizeStatements(period string) (*StatementSummary, error) {
	// Explicitly make an empty slice: an empty period serializes as [] instead of null (contract appendix D3 freezes "items empty array")
	items := make([]StatementSummaryItem, 0)
	if err := DB.Model(&CommissionStatement{}).
		Select("commission_statements.distributor_id, users.username, commission_statements.total_commission_cents, commission_statements.adjusted_cents, commission_statements.status AS statement_status, commission_statements.id AS statement_id").
		Joins("LEFT JOIN users ON users.id = commission_statements.distributor_id").
		Where("commission_statements.period = ?", period).
		Order("commission_statements.total_commission_cents DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	summary := &StatementSummary{Period: period, Items: items}
	for _, it := range items {
		summary.Totals.TotalCommissionCents += it.TotalCommissionCents
		summary.Totals.AdjustedCents += it.AdjustedCents
	}
	summary.Totals.DistributorCount = len(items)
	return summary, nil
}
