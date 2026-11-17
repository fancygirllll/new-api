//go:build !windows

package common

import (
	"os"

	"golang.org/x/sys/unix"
)

// GetDiskSpaceInfo returns disk space information for the disk containing the cache directory (Unix/Linux/macOS)
func GetDiskSpaceInfo() DiskSpaceInfo {
	cachePath := GetDiskCachePath()
	if cachePath == "" {
		cachePath = os.TempDir()
	}

	info := DiskSpaceInfo{}

	var stat unix.Statfs_t
	err := unix.Statfs(cachePath, &stat)
	if err != nil {
		return info
	}

	// Compute disk space (explicit conversion for FreeBSD compatibility, where the field type is int64)
	bsize := uint64(stat.Bsize)
	info.Total = uint64(stat.Blocks) * bsize
	info.Free = uint64(stat.Bavail) * bsize
	info.Used = info.Total - uint64(stat.Bfree)*bsize

	if info.Total > 0 {
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}

	return info
}
