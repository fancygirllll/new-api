package model

import (
	"strings"
)

// Simplified vendor mapping rules
var defaultVendorRules = map[string]string{
	"gpt":      "OpenAI",
	"dall-e":   "OpenAI",
	"whisper":  "OpenAI",
	"o1":       "OpenAI",
	"o3":       "OpenAI",
	"claude":   "Anthropic",
	"gemini":   "Google",
	"moonshot": "Moonshot",
	"kimi":     "Moonshot",
	"chatglm":  "Zhipu",
	"glm-":     "Zhipu",
	"qwen":     "Alibaba",
	"deepseek": "DeepSeek",
	"abab":     "MiniMax",
	"ernie":    "Baidu",
	"spark":    "iFlytek",
	"hunyuan":  "Tencent",
	"command":  "Cohere",
	"@cf/":     "Cloudflare",
	"360":      "360",
	"yi":       "Yi",
	"jina":     "Jina",
	"mistral":  "Mistral",
	"grok":     "xAI",
	"llama":    "Meta",
	"doubao":   "ByteDance",
	"kling":    "Kuaishou",
	"jimeng":   "Jimeng",
	"vidu":     "Vidu",
}

// Default vendor icon mapping
var defaultVendorIcons = map[string]string{
	"OpenAI":          "OpenAI",
	"Anthropic":       "Claude.Color",
	"Google":          "Gemini.Color",
	"Moonshot":        "Moonshot",
	"Zhipu":           "Zhipu.Color",
	"Alibaba":         "Qwen.Color",
	"DeepSeek":        "DeepSeek.Color",
	"MiniMax":         "Minimax.Color",
	"Baidu":           "Wenxin.Color",
	"iFlytek":         "Spark.Color",
	"Tencent":         "Hunyuan.Color",
	"Cohere":          "Cohere.Color",
	"Cloudflare":      "Cloudflare.Color",
	"360":             "Ai360.Color",
	"Yi":              "Yi.Color",
	"Jina":            "Jina",
	"Mistral":         "Mistral.Color",
	"xAI":             "XAI",
	"Meta":            "Ollama",
	"ByteDance":       "Doubao.Color",
	"Kuaishou":        "Kling.Color",
	"Jimeng":          "Jimeng.Color",
	"Vidu":            "Vidu",
	"Microsoft Azure": "AzureAI",
	"Microsoft":       "AzureAI",
	"Azure":           "AzureAI",
}

// initDefaultVendorMapping performs the simplified default vendor mapping
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	for _, ability := range enableAbilities {
		modelName := ability.Model
		if _, exists := metaMap[modelName]; exists {
			continue
		}

		// Match a vendor
		vendorID := 0
		modelLower := strings.ToLower(modelName)
		for pattern, vendorName := range defaultVendorRules {
			if strings.Contains(modelLower, pattern) {
				vendorID = getOrCreateVendor(vendorName, vendorMap)
				break
			}
		}

		// Create model metadata
		metaMap[modelName] = &Model{
			ModelName: modelName,
			VendorID:  vendorID,
			Status:    1,
			NameRule:  NameRuleExact,
		}
	}
}

// Find or create the vendor
func getOrCreateVendor(vendorName string, vendorMap map[int]*Vendor) int {
	// Look up an existing vendor
	for id, vendor := range vendorMap {
		if vendor.Name == vendorName {
			return id
		}
	}

	// Create a new vendor
	newVendor := &Vendor{
		Name:   vendorName,
		Status: 1,
		Icon:   getDefaultVendorIcon(vendorName),
	}

	if err := newVendor.Insert(); err != nil {
		return 0
	}

	vendorMap[newVendor.Id] = newVendor
	return newVendor.Id
}

// Get the vendor's default icon
func getDefaultVendorIcon(vendorName string) string {
	if icon, exists := defaultVendorIcons[vendorName]; exists {
		return icon
	}
	return ""
}
