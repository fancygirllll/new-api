package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// CheckinSetting is the check-in feature configuration
type CheckinSetting struct {
	Enabled  bool `json:"enabled"`   // Whether the check-in feature is enabled
	MinQuota int  `json:"min_quota"` // Minimum check-in quota reward
	MaxQuota int  `json:"max_quota"` // Maximum check-in quota reward
}

// Default configuration
var checkinSetting = CheckinSetting{
	Enabled:  false, // Disabled by default
	MinQuota: 1000,  // Default minimum quota 1000 (about 0.002 USD)
	MaxQuota: 10000, // Default maximum quota 10000 (about 0.02 USD)
}

func init() {
	// Register with the global config manager
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

// GetCheckinSetting returns the check-in configuration
func GetCheckinSetting() *CheckinSetting {
	return &checkinSetting
}

// IsCheckinEnabled reports whether the check-in feature is enabled
func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

// GetCheckinQuotaRange returns the check-in quota range
func GetCheckinQuotaRange() (min, max int) {
	return checkinSetting.MinQuota, checkinSetting.MaxQuota
}
