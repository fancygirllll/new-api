// Legacy keys used for migration detection; this file will be removed in the next release

package controller

import (
	"encoding/json"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// MigrateConsoleSetting migrates legacy console settings to console_setting.*
func MigrateConsoleSetting(c *gin.Context) {
	// Read all options
	opts, err := model.AllOption()
	if err != nil {
		common.SysError("failed to get all options: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get configuration, please try again later"})
		return
	}
	// Build the map
	valMap := map[string]string{}
	for _, o := range opts {
		valMap[o.Key] = o.Value
	}

	// Handle APIInfo
	if v := valMap["ApiInfo"]; v != "" {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			if len(arr) > 50 {
				arr = arr[:50]
			}
			bytes, _ := json.Marshal(arr)
			model.UpdateOption("console_setting.api_info", string(bytes))
		}
		model.UpdateOption("ApiInfo", "")
	}
	// Copy Announcements as-is
	if v := valMap["Announcements"]; v != "" {
		model.UpdateOption("console_setting.announcements", v)
		model.UpdateOption("Announcements", "")
	}
	// Convert FAQ
	if v := valMap["FAQ"]; v != "" {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			out := []map[string]interface{}{}
			for _, item := range arr {
				q, _ := item["question"].(string)
				if q == "" {
					q, _ = item["title"].(string)
				}
				a, _ := item["answer"].(string)
				if a == "" {
					a, _ = item["content"].(string)
				}
				if q != "" && a != "" {
					out = append(out, map[string]interface{}{"question": q, "answer": a})
				}
			}
			if len(out) > 50 {
				out = out[:50]
			}
			bytes, _ := json.Marshal(out)
			model.UpdateOption("console_setting.faq", string(bytes))
		}
		model.UpdateOption("FAQ", "")
	}
	// Migrate Uptime Kuma to the new groups structure (console_setting.uptime_kuma_groups)
	url := valMap["UptimeKumaUrl"]
	slug := valMap["UptimeKumaSlug"]
	if url != "" && slug != "" {
		// Migrate only when both URL and Slug are present
		groups := []map[string]interface{}{
			{
				"id":           1,
				"categoryName": "old",
				"url":          url,
				"slug":         slug,
				"description":  "",
			},
		}
		bytes, _ := json.Marshal(groups)
		model.UpdateOption("console_setting.uptime_kuma_groups", string(bytes))
	}
	// Clear the contents of the old keys
	if url != "" {
		model.UpdateOption("UptimeKumaUrl", "")
	}
	if slug != "" {
		model.UpdateOption("UptimeKumaSlug", "")
	}

	// Delete the old key records
	oldKeys := []string{"ApiInfo", "Announcements", "FAQ", "UptimeKumaUrl", "UptimeKumaSlug"}
	model.DB.Where("key IN ?", oldKeys).Delete(&model.Option{})

	// Reload OptionMap
	model.InitOptionMap()
	common.SysLog("console setting migrated")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "migrated"})
}
