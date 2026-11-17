package console_setting

import "github.com/QuantumNous/new-api/setting/config"

type ConsoleSetting struct {
	ApiInfo              string `json:"api_info"`              // Console API information (JSON array string)
	UptimeKumaGroups     string `json:"uptime_kuma_groups"`    // Uptime Kuma group configuration (JSON array string)
	Announcements        string `json:"announcements"`         // System announcements (JSON array string)
	FAQ                  string `json:"faq"`                   // Frequently asked questions (JSON array string)
	ApiInfoEnabled       bool   `json:"api_info_enabled"`      // Whether the API information panel is enabled
	UptimeKumaEnabled    bool   `json:"uptime_kuma_enabled"`   // Whether the Uptime Kuma panel is enabled
	AnnouncementsEnabled bool   `json:"announcements_enabled"` // Whether the announcements panel is enabled
	FAQEnabled           bool   `json:"faq_enabled"`           // Whether the FAQ panel is enabled
}

// Default configuration
var defaultConsoleSetting = ConsoleSetting{
	ApiInfo:              "",
	UptimeKumaGroups:     "",
	Announcements:        "",
	FAQ:                  "",
	ApiInfoEnabled:       true,
	UptimeKumaEnabled:    true,
	AnnouncementsEnabled: true,
	FAQEnabled:           true,
}

// Global instance
var consoleSetting = defaultConsoleSetting

func init() {
	// Register with the global config manager under the key console_setting
	config.GlobalConfig.Register("console_setting", &consoleSetting)
}

// GetConsoleSetting returns the ConsoleSetting configuration instance
func GetConsoleSetting() *ConsoleSetting {
	return &consoleSetting
}
