package model

import (
	"errors"
	"math/rand"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

// Checkin is a check-in record
type Checkin struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex:idx_user_checkin_date"`
	CheckinDate  string `json:"checkin_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_user_checkin_date"` // Format: YYYY-MM-DD
	QuotaAwarded int    `json:"quota_awarded" gorm:"not null"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
}

// CheckinRecord is the check-in record returned by the API (without sensitive fields)
type CheckinRecord struct {
	CheckinDate  string `json:"checkin_date"`
	QuotaAwarded int    `json:"quota_awarded"`
}

func (Checkin) TableName() string {
	return "checkins"
}

// GetUserCheckinRecords returns a user's check-in records within a date range
func GetUserCheckinRecords(userId int, startDate, endDate string) ([]Checkin, error) {
	var records []Checkin
	err := DB.Where("user_id = ? AND checkin_date >= ? AND checkin_date <= ?",
		userId, startDate, endDate).
		Order("checkin_date DESC").
		Find(&records).Error
	return records, err
}

// HasCheckedInToday checks whether the user has checked in today
func HasCheckedInToday(userId int) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var count int64
	err := DB.Model(&Checkin{}).
		Where("user_id = ? AND checkin_date = ?", userId, today).
		Count(&count).Error
	return count > 0, err
}

// UserCheckin performs the user check-in
// MySQL and PostgreSQL use transactions to guarantee atomicity
// SQLite does not support nested transactions, so use sequential operations + manual rollback
func UserCheckin(userId int) (*Checkin, error) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		return nil, errors.New("Check-in is not enabled")
	}

	// Check whether the user has already checked in today
	hasChecked, err := HasCheckedInToday(userId)
	if err != nil {
		return nil, err
	}
	if hasChecked {
		return nil, errors.New("Already checked in today")
	}

	// Compute the random quota reward
	quotaAwarded := setting.MinQuota
	if setting.MaxQuota > setting.MinQuota {
		quotaAwarded = setting.MinQuota + rand.Intn(setting.MaxQuota-setting.MinQuota+1)
	}

	today := time.Now().Format("2006-01-02")
	checkin := &Checkin{
		UserId:       userId,
		CheckinDate:  today,
		QuotaAwarded: quotaAwarded,
		CreatedAt:    time.Now().Unix(),
	}

	// Choose a strategy based on the database type
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		// SQLite does not support nested transactions, so use sequential operations + manual rollback
		return userCheckinWithoutTransaction(checkin, userId, quotaAwarded)
	}

	// MySQL and PostgreSQL support transactions, so use a transaction to guarantee atomicity
	return userCheckinWithTransaction(checkin, userId, quotaAwarded)
}

// userCheckinWithTransaction performs the check-in in a transaction (for MySQL and PostgreSQL)
func userCheckinWithTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: create the check-in record
		// The unique constraint (user_id, checkin_date) prevents duplicate concurrent check-ins
		if err := tx.Create(checkin).Error; err != nil {
			return errors.New("Check-in failed, please retry later")
		}

		// Step 2: increase the user quota in the transaction
		if err := tx.Model(&User{}).Where("id = ?", userId).
			Update("quota", gorm.Expr("quota + ?", quotaAwarded)).Error; err != nil {
			return errors.New("Check-in failed: error updating quota")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// After the transaction succeeds, update the cache asynchronously
	go func() {
		_ = cacheIncrUserQuota(userId, int64(quotaAwarded))
	}()

	return checkin, nil
}

// userCheckinWithoutTransaction performs the check-in without a transaction (for SQLite)
func userCheckinWithoutTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	// Step 1: create the check-in record
	// The unique constraint (user_id, checkin_date) prevents duplicate concurrent check-ins
	if err := DB.Create(checkin).Error; err != nil {
		return nil, errors.New("Check-in failed, please retry later")
	}

	// Step 2: increase the user quota
	// Use db=true to force a direct database write instead of batched updates
	if err := IncreaseUserQuota(userId, quotaAwarded, true); err != nil {
		// If increasing the quota fails, roll back the check-in record
		DB.Delete(checkin)
		return nil, errors.New("Check-in failed: error updating quota")
	}

	return checkin, nil
}

// GetUserCheckinStats returns the user's check-in statistics
func GetUserCheckinStats(userId int, month string) (map[string]interface{}, error) {
	// Get all check-in records for the given month
	startDate := month + "-01"
	endDate := month + "-31"

	records, err := GetUserCheckinRecords(userId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Convert to records without sensitive fields
	checkinRecords := make([]CheckinRecord, len(records))
	for i, r := range records {
		checkinRecords[i] = CheckinRecord{
			CheckinDate:  r.CheckinDate,
			QuotaAwarded: r.QuotaAwarded,
		}
	}

	// Check whether the user has already checked in today
	hasCheckedToday, _ := HasCheckedInToday(userId)

	// Get all-time check-in statistics for the user
	var totalCheckins int64
	var totalQuota int64
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Count(&totalCheckins)
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Select("COALESCE(SUM(quota_awarded), 0)").Scan(&totalQuota)

	return map[string]interface{}{
		"total_quota":      totalQuota,      // Total quota earned all-time
		"total_checkins":   totalCheckins,   // Total check-in count all-time
		"checkin_count":    len(records),    // Check-in count this month
		"checked_in_today": hasCheckedToday, // Whether checked in today
		"records":          checkinRecords,  // Check-in record details for this month (without id and user_id)
	}, nil
}
