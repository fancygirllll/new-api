package model

// AttributionChange is the attribution change audit table (contract 01-CONTRACT.md §2.3 frozen).
// Covers audit trails for CUST-04 (admin rebind) and ATTR-05 (self-service rebind).
type AttributionChange struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"type:int;index:idx_attr_user"` // The customer whose attribution changed
	OldInviterId int    `json:"old_inviter_id" gorm:"type:int"`              // 0 = previously unbound
	NewInviterId int    `json:"new_inviter_id" gorm:"type:int;index:idx_attr_distributor"`
	Source       string `json:"source" gorm:"type:varchar(20)"`  // admin_bind | self_bind
	OperatorId   int    `json:"operator_id" gorm:"type:int"`     // Equals UserId for self_bind
	Reason       string `json:"reason" gorm:"type:varchar(255)"` // Required for admin_bind; "self" for self_bind
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

// TableName explicitly specifies the table name (contract §2.3 frozen).
func (AttributionChange) TableName() string { return "attribution_changes" }
