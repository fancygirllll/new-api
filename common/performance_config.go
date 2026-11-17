package common

import "sync/atomic"

// PerformanceMonitorConfig is the performance monitoring configuration
type PerformanceMonitorConfig struct {
	Enabled         bool
	CPUThreshold    int
	MemoryThreshold int
	DiskThreshold   int
}

var performanceMonitorConfig atomic.Value

func init() {
	// Initialize the default configuration
	performanceMonitorConfig.Store(PerformanceMonitorConfig{
		Enabled:         true,
		CPUThreshold:    90,
		MemoryThreshold: 90,
		DiskThreshold:   90,
	})
}

// GetPerformanceMonitorConfig returns the performance monitoring configuration
func GetPerformanceMonitorConfig() PerformanceMonitorConfig {
	return performanceMonitorConfig.Load().(PerformanceMonitorConfig)
}

// SetPerformanceMonitorConfig sets the performance monitoring configuration
func SetPerformanceMonitorConfig(config PerformanceMonitorConfig) {
	performanceMonitorConfig.Store(config)
}
