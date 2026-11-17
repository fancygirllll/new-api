package common

import (
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// DiskSpaceInfo describes disk space information
type DiskSpaceInfo struct {
	// Total space (bytes)
	Total uint64 `json:"total"`
	// Available space (bytes)
	Free uint64 `json:"free"`
	// Used space (bytes)
	Used uint64 `json:"used"`
	// Usage percentage
	UsedPercent float64 `json:"used_percent"`
}

// SystemStatus describes system status information
type SystemStatus struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
}

var latestSystemStatus atomic.Value

func init() {
	latestSystemStatus.Store(SystemStatus{})
}

// StartSystemMonitor starts system monitoring
func StartSystemMonitor() {
	go func() {
		for {
			config := GetPerformanceMonitorConfig()
			if !config.Enabled {
				time.Sleep(30 * time.Second)
				continue
			}

			updateSystemStatus()
			time.Sleep(5 * time.Second)
		}
	}()
}

func updateSystemStatus() {
	var status SystemStatus

	// CPU
	// Note: cpu.Percent(0, false) returns CPU usage since the previous call
	// The first call may return an error or an inaccurate value, but it becomes reliable within the loop
	percents, err := cpu.Percent(0, false)
	if err == nil && len(percents) > 0 {
		status.CPUUsage = percents[0]
	}

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		status.MemoryUsage = memInfo.UsedPercent
	}

	// Disk
	diskInfo := GetDiskSpaceInfo()
	if diskInfo.Total > 0 {
		status.DiskUsage = diskInfo.UsedPercent
	}

	latestSystemStatus.Store(status)
}

// GetSystemStatus returns the current system status
func GetSystemStatus() SystemStatus {
	return latestSystemStatus.Load().(SystemStatus)
}
