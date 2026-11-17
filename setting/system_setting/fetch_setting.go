package system_setting

import "github.com/QuantumNous/new-api/setting/config"

type FetchSetting struct {
	EnableSSRFProtection   bool     `json:"enable_ssrf_protection"` // Whether SSRF protection is enabled
	AllowPrivateIp         bool     `json:"allow_private_ip"`
	DomainFilterMode       bool     `json:"domain_filter_mode"`         // Domain filter mode: true for allowlist, false for blocklist
	IpFilterMode           bool     `json:"ip_filter_mode"`             // IP filter mode: true for allowlist, false for blocklist
	DomainList             []string `json:"domain_list"`                // domain format, e.g. example.com, *.example.com
	IpList                 []string `json:"ip_list"`                    // CIDR format
	AllowedPorts           []string `json:"allowed_ports"`              // port range format, e.g. 80, 443, 8000-9000
	ApplyIPFilterForDomain bool     `json:"apply_ip_filter_for_domain"` // Apply IP filtering to domains (experimental)
}

var defaultFetchSetting = FetchSetting{
	EnableSSRFProtection:   true, // SSRF protection enabled by default
	AllowPrivateIp:         false,
	DomainFilterMode:       false,
	IpFilterMode:           false,
	DomainList:             []string{},
	IpList:                 []string{},
	AllowedPorts:           []string{"80", "443", "8080", "8443"},
	ApplyIPFilterForDomain: true,
}

func init() {
	// Register with the global config manager
	config.GlobalConfig.Register("fetch_setting", &defaultFetchSetting)
}

func GetFetchSetting() *FetchSetting {
	return &defaultFetchSetting
}
