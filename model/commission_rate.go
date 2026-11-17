package model

import (
	"errors"

	"gorm.io/gorm"
)

// CommissionRate is the rate configuration table (contract 01-CONTRACT.md §3.2 frozen).
// One current effective rate row per distributor; change history is written to CommissionRateHistory in the same transaction (RATE-03).
type CommissionRate struct {
	Id            int   `json:"id" gorm:"primaryKey"`
	DistributorId int   `json:"distributor_id" gorm:"type:int;uniqueIndex:uk_rate_distributor"` // One row per distributor
	RateBp        int   `json:"rate_bp" gorm:"type:int"`                                        // Current effective rate (basis points, 0-10000)
	UpdatedBy     int   `json:"updated_by" gorm:"type:int"`
	UpdatedAt     int64 `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at"`
}

// CommissionRateHistory is the rate change history table (contract §3.2 frozen).
// Changing the rate does not affect any existing flow (the snapshot lives on the flow, RATE-02).
type CommissionRateHistory struct {
	Id            int   `json:"id" gorm:"primaryKey"`
	DistributorId int   `json:"distributor_id" gorm:"type:int;index:idx_rate_hist_distributor"`
	OldRateBp     int   `json:"old_rate_bp" gorm:"type:int"`
	NewRateBp     int   `json:"new_rate_bp" gorm:"type:int"`
	OperatorId    int   `json:"operator_id" gorm:"type:int"`
	CreatedAt     int64 `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

// TableName explicitly specifies the table name (contract §3.2 frozen).
func (CommissionRate) TableName() string { return "commission_rates" }

// TableName explicitly specifies the table name (contract §3.2 frozen).
func (CommissionRateHistory) TableName() string { return "commission_rate_histories" }

// GetRateByDistributorId reads a distributor's current effective rate (03-02, RESEARCH frozen signature).
// gorm.ErrRecordNotFound is returned as-is - caller semantics: no rate config = not enabled (contract §7#3).
// Used by A2/A9/A12 and later phases; the caller must pass tx inside a transaction (Pitfall 2: no global DB).
func GetRateByDistributorId(tx *gorm.DB, distributorId int) (*CommissionRate, error) {
	var rate CommissionRate
	if err := tx.First(&rate, "distributor_id = ?", distributorId).Error; err != nil {
		return nil, err
	}
	return &rate, nil
}

// UpsertRateWithHistory writes the rate change and its history in the same transaction (contract §3.2 frozen:
// update the current value + insert a history row, ensuring RATE-03 is queryable and lossless).
// The caller must pass tx inside model.DB.Transaction (A2 controller transaction skeleton).
//
// Ordering semantics (the RESEARCH anti-pattern forbids reversing it): rate write succeeds first, then history -
// history is a subordinate record of the rate change; concurrent double writes are backed by the uk_rate_distributor unique index
// (contract §5 frozen design; no application-level check-then-insert two-step logic - TOCTOU window).
func UpsertRateWithHistory(tx *gorm.DB, distributorId, rateBp, operatorId int) error {
	// 1. Read the old row: ErrRecordNotFound -> oldBp=0 and use Create; other DB errors are propagated
	oldBp := 0
	var existing CommissionRate
	err := tx.First(&existing, "distributor_id = ?", distributorId).Error
	switch {
	case err == nil:
		oldBp = existing.RateBp
	case errors.Is(err, gorm.ErrRecordNotFound):
		// First-time setup: old=0
	default:
		return err
	}

	// 2. Write rate first: Updates when a row exists (same row), Create when none (unique index backs concurrent double writes)
	if oldBp != 0 || existing.Id != 0 {
		if err := tx.Model(&CommissionRate{}).Where("distributor_id = ?", distributorId).
			Updates(map[string]interface{}{
				"rate_bp":    rateBp,
				"updated_by": operatorId,
			}).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Create(&CommissionRate{
			DistributorId: distributorId,
			RateBp:        rateBp,
			UpdatedBy:     operatorId,
		}).Error; err != nil {
			return err
		}
	}

	// 3. Write history afterwards: the subordinate record of the rate change (the sole source of RATE-03 auditability)
	return tx.Create(&CommissionRateHistory{
		DistributorId: distributorId,
		OldRateBp:     oldBp,
		NewRateBp:     rateBp,
		OperatorId:    operatorId,
	}).Error
}

// CommissionRateDetail is an A1 list item: CommissionRate + redundant distributor username (contract §4.3 A1).
// Anonymous embedding produces flat JSON: id/distributor_id/rate_bp/updated_by/updated_at/username.
type CommissionRateDetail struct {
	CommissionRate
	Username string `json:"username"`
}

// ListRates paginates the rate list (contract §4.3 A1: items include the redundant distributor username for display).
// Clean count (no JOIN) + LEFT JOIN users for the redundant username, ordered by distributor_id ASC.
func ListRates(startIdx, num int) ([]CommissionRateDetail, int64, error) {
	var total int64
	if err := DB.Model(&CommissionRate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var details []CommissionRateDetail
	if err := DB.Model(&CommissionRate{}).
		Select("commission_rates.*, users.username").
		Joins("LEFT JOIN users ON users.id = commission_rates.distributor_id").
		Order("commission_rates.distributor_id ASC").
		Offset(startIdx).Limit(num).
		Find(&details).Error; err != nil {
		return nil, 0, err
	}
	return details, total, nil
}

// GetRateHistories paginates rate change history (contract §4.3 A3, RATE-03).
// Filter by distributorId, id DESC (newest changes first).
func GetRateHistories(distributorId, startIdx, num int) ([]CommissionRateHistory, int64, error) {
	var total int64
	query := DB.Model(&CommissionRateHistory{}).Where("distributor_id = ?", distributorId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var hists []CommissionRateHistory
	if err := query.Order("id DESC").Offset(startIdx).Limit(num).Find(&hists).Error; err != nil {
		return nil, 0, err
	}
	return hists, total, nil
}
