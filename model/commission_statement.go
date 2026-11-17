package model

// CommissionStatement is the statement table (contract 01-CONTRACT.md §3.3 frozen).
// The uk_stmt_period unique index prevents duplicate statement generation; locked on creation (STMT-03 immutable).
type CommissionStatement struct {
	Id                   int64  `json:"id" gorm:"primaryKey"`
	DistributorId        int    `json:"distributor_id" gorm:"type:int;uniqueIndex:uk_stmt_period,unique"`
	Period               string `json:"period" gorm:"type:varchar(7);uniqueIndex:uk_stmt_period,unique"` // "2026-08"; the unique index prevents duplicate statement generation
	TotalTopupCents      int64  `json:"total_topup_cents" gorm:"type:bigint"`
	TotalCommissionCents int64  `json:"total_commission_cents" gorm:"type:bigint"`
	AdjustedCents        int64  `json:"adjusted_cents" gorm:"type:bigint;default:0"` // Accumulated adjustment difference (COMM-06)
	SettleAmountCents    int64  `json:"settle_amount_cents" gorm:"type:bigint"`      // = TotalCommission + Adjusted; settled by this amount
	Status               string `json:"status" gorm:"type:varchar(20);index:idx_stmt_status"`
	CreatedAt            int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"` // Statement generation time
	LockedAt             int64  `json:"locked_at" gorm:"column:locked_at"`                  // Locked on creation (STMT-03, immutable)
}

// CommissionStatementItem is the statement item table (contract §3.3 frozen, STMT-02).
type CommissionStatementItem struct {
	Id              int64  `json:"id" gorm:"primaryKey"`
	StatementId     int64  `json:"statement_id" gorm:"type:bigint;index:idx_stmt_item_stmt"`
	FlowId          int64  `json:"flow_id" gorm:"type:bigint"`             // References commission_flows.id (audit traceability)
	BillingNo       string `json:"billing_no" gorm:"type:varchar(64)"`     // Original top-up information (STMT-02)
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(50)"` // Snapshot from TopUp
	PaymentProvider string `json:"payment_provider" gorm:"type:varchar(50)"`
	TopupMoneyCents int64  `json:"topup_money_cents" gorm:"type:bigint"`
	CommissionCents int64  `json:"commission_cents" gorm:"type:bigint"`
	CustomerId      int    `json:"customer_id" gorm:"type:int"`               // Customer identifier
	CompleteTime    int64  `json:"complete_time" gorm:"column:complete_time"` // Credit time
}

// StatementAdjustment is a statement adjustment entry (contract §3.3 frozen, COMM-06).
// Can only be created when status=payable (settled is rejected, COMM-07); accumulates statement.adjusted_cents in the same transaction.
type StatementAdjustment struct {
	Id          int64  `json:"id" gorm:"primaryKey"`
	StatementId int64  `json:"statement_id" gorm:"type:bigint;index:idx_stmt_adj_stmt"`
	DeltaCents  int64  `json:"delta_cents" gorm:"type:bigint"`  // Positive or negative difference correction (COMM-06)
	Reason      string `json:"reason" gorm:"type:varchar(255)"` // Required
	OperatorId  int    `json:"operator_id" gorm:"type:int"`     // RootAuth only
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

// Statement Status enum (contract §3.3 frozen).
const (
	StatementPayable     = "payable"     // Payable
	StatementWithdrawing = "withdrawing" // Withdrawing (set after a withdrawal request in domain C, WDRAW-04)
	StatementSettled     = "settled"     // Settled (set after payment in domain C)
)

// TableName explicitly specifies the table name (contract §3.3 frozen).
func (CommissionStatement) TableName() string { return "commission_statements" }

// TableName explicitly specifies the table name (contract §3.3 frozen).
func (CommissionStatementItem) TableName() string { return "commission_statement_items" }

// TableName explicitly specifies the table name (contract §3.3 frozen).
func (StatementAdjustment) TableName() string { return "commission_statement_adjustments" }
