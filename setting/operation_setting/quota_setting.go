package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type QuotaSetting struct {
	EnableFreeModelPreConsume bool `json:"enable_free_model_pre_consume"` // Whether pre-consumption is enabled for free models
}

// Default configuration
var quotaSetting = QuotaSetting{
	EnableFreeModelPreConsume: true,
}

func init() {
	// Register with the global config manager
	config.GlobalConfig.Register("quota_setting", &quotaSetting)
}

func GetQuotaSetting() *QuotaSetting {
	return &quotaSetting
}
