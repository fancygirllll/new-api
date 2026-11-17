package performance_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// PerformanceSetting is the performance configuration
type PerformanceSetting struct {
	// DiskCacheEnabled controls whether disk caching is enabled (using disk instead of memory)
	DiskCacheEnabled bool `json:"disk_cache_enabled"`
	// DiskCacheThresholdMB is the request body size threshold that triggers disk caching (MB)
	DiskCacheThresholdMB int `json:"disk_cache_threshold_mb"`
	// DiskCacheMaxSizeMB is the maximum total disk cache size (MB)
	DiskCacheMaxSizeMB int `json:"disk_cache_max_size_mb"`
	// DiskCachePath is the disk cache directory
	DiskCachePath string `json:"disk_cache_path"`

	// MonitorEnabled controls whether performance monitoring is enabled
	MonitorEnabled bool `json:"monitor_enabled"`
	// MonitorCPUThreshold is the CPU usage threshold (%)
	MonitorCPUThreshold int `json:"monitor_cpu_threshold"`
	// MonitorMemoryThreshold is the memory usage threshold (%)
	MonitorMemoryThreshold int `json:"monitor_memory_threshold"`
	// MonitorDiskThreshold is the disk usage threshold (%)
	MonitorDiskThreshold int `json:"monitor_disk_threshold"`
}

// Default configuration
var performanceSetting = PerformanceSetting{
	DiskCacheEnabled:     false,
	DiskCacheThresholdMB: 10,   // Use disk caching above 10MB
	DiskCacheMaxSizeMB:   1024, // Maximum disk cache of 1GB
	DiskCachePath:        "",   // Empty means the system temporary directory

	MonitorEnabled:         true,
	MonitorCPUThreshold:    90,
	MonitorMemoryThreshold: 90,
	MonitorDiskThreshold:   95,
}

func init() {
	// Register with the global config manager
	config.GlobalConfig.Register("performance_setting", &performanceSetting)
	// Sync the initial configuration to the common package
	syncToCommon()
}

// syncToCommon synchronizes the configuration to the common package
func syncToCommon() {
	common.SetDiskCacheConfig(common.DiskCacheConfig{
		Enabled:     performanceSetting.DiskCacheEnabled,
		ThresholdMB: performanceSetting.DiskCacheThresholdMB,
		MaxSizeMB:   performanceSetting.DiskCacheMaxSizeMB,
		Path:        performanceSetting.DiskCachePath,
	})

	common.SetPerformanceMonitorConfig(common.PerformanceMonitorConfig{
		Enabled:         performanceSetting.MonitorEnabled,
		CPUThreshold:    performanceSetting.MonitorCPUThreshold,
		MemoryThreshold: performanceSetting.MonitorMemoryThreshold,
		DiskThreshold:   performanceSetting.MonitorDiskThreshold,
	})
}

// GetPerformanceSetting returns the performance settings
func GetPerformanceSetting() *PerformanceSetting {
	return &performanceSetting
}

// UpdateAndSync updates the configuration and synchronizes it to the common package
// Call this after the configuration is loaded from the database to synchronize it
func UpdateAndSync() {
	syncToCommon()
}

// GetCacheStats returns cache statistics (delegates to the common package)
func GetCacheStats() common.DiskCacheStats {
	return common.GetDiskCacheStats()
}

// ResetStats resets the statistics
func ResetStats() {
	common.ResetDiskCacheStats()
}
