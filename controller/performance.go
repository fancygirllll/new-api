package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
)

// PerformanceStats holds performance statistics
type PerformanceStats struct {
	// Cache statistics
	CacheStats common.DiskCacheStats `json:"cache_stats"`
	// System memory statistics
	MemoryStats MemoryStats `json:"memory_stats"`
	// Disk cache directory info
	DiskCacheInfo DiskCacheInfo `json:"disk_cache_info"`
	// Disk space info
	DiskSpaceInfo common.DiskSpaceInfo `json:"disk_space_info"`
	// Configuration info
	Config PerformanceConfig `json:"config"`
}

// MemoryStats holds memory statistics
type MemoryStats struct {
	// Allocated memory (bytes)
	Alloc uint64 `json:"alloc"`
	// Total allocated memory (bytes)
	TotalAlloc uint64 `json:"total_alloc"`
	// System memory (bytes)
	Sys uint64 `json:"sys"`
	// GC count
	NumGC uint32 `json:"num_gc"`
	// Goroutine count
	NumGoroutine int `json:"num_goroutine"`
}

// DiskCacheInfo holds disk cache directory info
type DiskCacheInfo struct {
	// Cache directory path
	Path string `json:"path"`
	// Whether the directory exists
	Exists bool `json:"exists"`
	// File count
	FileCount int `json:"file_count"`
	// Total size (bytes)
	TotalSize int64 `json:"total_size"`
}

// PerformanceConfig holds performance settings
type PerformanceConfig struct {
	// Whether disk caching is enabled
	DiskCacheEnabled bool `json:"disk_cache_enabled"`
	// Disk cache threshold (MB)
	DiskCacheThresholdMB int `json:"disk_cache_threshold_mb"`
	// Maximum disk cache size (MB)
	DiskCacheMaxSizeMB int `json:"disk_cache_max_size_mb"`
	// Disk cache path
	DiskCachePath string `json:"disk_cache_path"`
	// Whether running in a container
	IsRunningInContainer bool `json:"is_running_in_container"`

	// MonitorEnabled reports whether performance monitoring is enabled
	MonitorEnabled bool `json:"monitor_enabled"`
	// MonitorCPUThreshold is the CPU usage threshold (%)
	MonitorCPUThreshold int `json:"monitor_cpu_threshold"`
	// MonitorMemoryThreshold is the memory usage threshold (%)
	MonitorMemoryThreshold int `json:"monitor_memory_threshold"`
	// MonitorDiskThreshold is the disk usage threshold (%)
	MonitorDiskThreshold int `json:"monitor_disk_threshold"`
}

// GetPerformanceStats returns performance statistics
func GetPerformanceStats(c *gin.Context) {
	// No longer scans the disk on every stats request; atomic counters keep it fast
	// Sync only at startup or on explicit cleanup
	cacheStats := common.GetDiskCacheStats()

	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get disk cache directory info
	diskCacheInfo := getDiskCacheInfo()

	// Get configuration info
	diskConfig := common.GetDiskCacheConfig()
	monitorConfig := common.GetPerformanceMonitorConfig()
	config := PerformanceConfig{
		DiskCacheEnabled:       diskConfig.Enabled,
		DiskCacheThresholdMB:   diskConfig.ThresholdMB,
		DiskCacheMaxSizeMB:     diskConfig.MaxSizeMB,
		DiskCachePath:          diskConfig.Path,
		IsRunningInContainer:   common.IsRunningInContainer(),
		MonitorEnabled:         monitorConfig.Enabled,
		MonitorCPUThreshold:    monitorConfig.CPUThreshold,
		MonitorMemoryThreshold: monitorConfig.MemoryThreshold,
		MonitorDiskThreshold:   monitorConfig.DiskThreshold,
	}

	// Get disk space info
	// Use the cached system status to avoid frequent system API calls
	systemStatus := common.GetSystemStatus()
	diskSpaceInfo := common.DiskSpaceInfo{
		UsedPercent: systemStatus.DiskUsage,
	}
	// If detailed info is needed, fetch it on demand or extend SystemStatus
	// For interface compatibility we still call GetDiskSpaceInfo, but note the potential performance cost
	// Since GetPerformanceStats is a low-frequency admin endpoint, calling it directly is acceptable
	// For consistency, some info could also come from SystemStatus
	diskSpaceInfo = common.GetDiskSpaceInfo()

	stats := PerformanceStats{
		CacheStats: cacheStats,
		MemoryStats: MemoryStats{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			NumGC:        memStats.NumGC,
			NumGoroutine: runtime.NumGoroutine(),
		},
		DiskCacheInfo: diskCacheInfo,
		DiskSpaceInfo: diskSpaceInfo,
		Config:        config,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ClearDiskCache clears inactive disk cache
func ClearDiskCache(c *gin.Context) {
	// Remove cache files unused for more than 10 minutes
	// 10 minutes is a safe threshold that avoids deleting files used by in-flight requests
	err := common.CleanupOldDiskCacheFiles(10 * time.Minute)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Inactive disk cache cleared",
	})
}

// ResetPerformanceStats resets performance statistics
func ResetPerformanceStats(c *gin.Context) {
	common.ResetDiskCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Statistics reset",
	})
}

// ForceGC forces a garbage collection
func ForceGC(c *gin.Context) {
	runtime.GC()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "GC executed",
	})
}

// LogFileInfo describes a log file
type LogFileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// LogFilesResponse is the log file list response
type LogFilesResponse struct {
	LogDir     string        `json:"log_dir"`
	Enabled    bool          `json:"enabled"`
	FileCount  int           `json:"file_count"`
	TotalSize  int64         `json:"total_size"`
	OldestTime *time.Time    `json:"oldest_time,omitempty"`
	NewestTime *time.Time    `json:"newest_time,omitempty"`
	Files      []LogFileInfo `json:"files"`
}

// getLogFiles reads the list of log files in the log directory
func getLogFiles() ([]LogFileInfo, error) {
	if *common.LogDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(*common.LogDir)
	if err != nil {
		return nil, err
	}
	var files []LogFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "oneapi-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, LogFileInfo{
			Name:    name,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	// Sort by file name descending (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name > files[j].Name
	})
	return files, nil
}

// GetLogFiles returns the log file list
func GetLogFiles(c *gin.Context) {
	if *common.LogDir == "" {
		common.ApiSuccess(c, LogFilesResponse{Enabled: false})
		return
	}
	files, err := getLogFiles()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var totalSize int64
	var oldest, newest time.Time
	for i, f := range files {
		totalSize += f.Size
		if i == 0 || f.ModTime.Before(oldest) {
			oldest = f.ModTime
		}
		if i == 0 || f.ModTime.After(newest) {
			newest = f.ModTime
		}
	}
	resp := LogFilesResponse{
		LogDir:    *common.LogDir,
		Enabled:   true,
		FileCount: len(files),
		TotalSize: totalSize,
		Files:     files,
	}
	if len(files) > 0 {
		resp.OldestTime = &oldest
		resp.NewestTime = &newest
	}
	common.ApiSuccess(c, resp)
}

// CleanupLogFiles removes expired log files
func CleanupLogFiles(c *gin.Context) {
	mode := c.Query("mode")
	valueStr := c.Query("value")
	if mode != "by_count" && mode != "by_days" {
		common.ApiErrorMsg(c, "invalid mode, must be by_count or by_days")
		return
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 1 {
		common.ApiErrorMsg(c, "invalid value, must be a positive integer")
		return
	}
	if *common.LogDir == "" {
		common.ApiErrorMsg(c, "log directory not configured")
		return
	}

	files, err := getLogFiles()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	activeLogPath := logger.GetCurrentLogPath()
	var toDelete []LogFileInfo

	switch mode {
	case "by_count":
		// files is sorted by name descending (newest first); keep the first value entries
		for i, f := range files {
			if i < value {
				continue
			}
			fullPath := filepath.Join(*common.LogDir, f.Name)
			if fullPath == activeLogPath {
				continue
			}
			toDelete = append(toDelete, f)
		}
	case "by_days":
		cutoff := time.Now().AddDate(0, 0, -value)
		for _, f := range files {
			if f.ModTime.Before(cutoff) {
				fullPath := filepath.Join(*common.LogDir, f.Name)
				if fullPath == activeLogPath {
					continue
				}
				toDelete = append(toDelete, f)
			}
		}
	}

	var deletedCount int
	var freedBytes int64
	var failedFiles []string
	for _, f := range toDelete {
		fullPath := filepath.Join(*common.LogDir, f.Name)
		if err := os.Remove(fullPath); err != nil {
			failedFiles = append(failedFiles, f.Name)
			continue
		}
		deletedCount++
		freedBytes += f.Size
	}

	result := gin.H{
		"deleted_count": deletedCount,
		"freed_bytes":   freedBytes,
		"failed_files":  failedFiles,
	}

	if len(failedFiles) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to delete some files (%d/%d)", len(failedFiles), len(toDelete)),
			"data":    result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// getDiskCacheInfo returns disk cache directory info
func getDiskCacheInfo() DiskCacheInfo {
	// Use the shared cache directory
	dir := common.GetDiskCacheDir()

	info := DiskCacheInfo{
		Path:   dir,
		Exists: false,
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return info
	}

	info.Exists = true
	info.FileCount = 0
	info.TotalSize = 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info.FileCount++
		if fileInfo, err := entry.Info(); err == nil {
			info.TotalSize += fileInfo.Size()
		}
	}

	return info
}
