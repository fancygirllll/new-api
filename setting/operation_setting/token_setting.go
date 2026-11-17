package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// TokenSetting is the token-related configuration
type TokenSetting struct {
	MaxUserTokens int `json:"max_user_tokens"` // Maximum number of tokens per user
}

// Default configuration
var tokenSetting = TokenSetting{
	MaxUserTokens: 1000, // Default maximum of 1000 tokens per user
}

func init() {
	// Register with the global config manager
	config.GlobalConfig.Register("token_setting", &tokenSetting)
}

// GetTokenSetting returns the token configuration
func GetTokenSetting() *TokenSetting {
	return &tokenSetting
}

// GetMaxUserTokens returns the maximum number of tokens per user
func GetMaxUserTokens() int {
	return GetTokenSetting().MaxUserTokens
}
