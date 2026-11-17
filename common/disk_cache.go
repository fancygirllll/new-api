package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// DiskCacheType is the disk cache type
type DiskCacheType string

const (
	DiskCacheTypeBody DiskCacheType = "body" // Request body cache
	DiskCacheTypeFile DiskCacheType = "file" // File data cache
)

// Unified cache directory name
const diskCacheDir = "new-api-body-cache"

// GetDiskCacheDir returns the unified disk cache directory
// Note: recomputed on every call to reflect configuration changes
func GetDiskCacheDir() string {
	cachePath := GetDiskCachePath()
	if cachePath == "" {
		cachePath = os.TempDir()
	}
	return filepath.Join(cachePath, diskCacheDir)
}

// EnsureDiskCacheDir ensures the cache directory exists
func EnsureDiskCacheDir() error {
	dir := GetDiskCacheDir()
	return os.MkdirAll(dir, 0755)
}

// CreateDiskCacheFile creates a disk cache file
// cacheType: cache type (body/file)
// Returns the file path and file handle
func CreateDiskCacheFile(cacheType DiskCacheType) (string, *os.File, error) {
	if err := EnsureDiskCacheDir(); err != nil {
		return "", nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dir := GetDiskCacheDir()
	filename := fmt.Sprintf("%s-%s-%d.tmp", cacheType, uuid.New().String()[:8], time.Now().UnixNano())
	filePath := filepath.Join(dir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cache file: %w", err)
	}

	return filePath, file, nil
}

// WriteDiskCacheFile writes data to a disk cache file
// Returns the file path
func WriteDiskCacheFile(cacheType DiskCacheType, data []byte) (string, error) {
	filePath, file, err := CreateDiskCacheFile(cacheType)
	if err != nil {
		return "", err
	}

	_, err = file.Write(data)
	if err != nil {
		file.Close()
		os.Remove(filePath)
		return "", fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("failed to close cache file: %w", err)
	}

	return filePath, nil
}

// WriteDiskCacheFileString writes a string to a disk cache file
func WriteDiskCacheFileString(cacheType DiskCacheType, data string) (string, error) {
	return WriteDiskCacheFile(cacheType, []byte(data))
}

// ReadDiskCacheFile reads a disk cache file
func ReadDiskCacheFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// ReadDiskCacheFileString reads a disk cache file as a string
func ReadDiskCacheFileString(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RemoveDiskCacheFile deletes a disk cache file
func RemoveDiskCacheFile(filePath string) error {
	return os.Remove(filePath)
}

// CleanupOldDiskCacheFiles removes old cache files
// maxAge: maximum file lifetime
// Note: this function only deletes files and does not update statistics (the original size of each file is unknown)
func CleanupOldDiskCacheFiles(maxAge time.Duration) error {
	dir := GetDiskCacheDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory does not exist; no cleanup needed
		}
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			// Note: when the background cleanup task deletes files, because the original base64Size is unknown,
			// it can only subtract the on-disk file size. This is accurate under the current base64 storage mode.
			if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
				DecrementDiskFiles(info.Size())
			}
		}
	}
	return nil
}

// GetDiskCacheInfo returns information about the disk cache directory
func GetDiskCacheInfo() (fileCount int, totalSize int64, err error) {
	dir := GetDiskCacheDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fileCount++
		totalSize += info.Size()
	}
	return fileCount, totalSize, nil
}

// ShouldUseDiskCache reports whether disk caching should be used
func ShouldUseDiskCache(dataSize int64) bool {
	if !IsDiskCacheEnabled() {
		return false
	}
	threshold := GetDiskCacheThresholdBytes()
	if dataSize < threshold {
		return false
	}
	return IsDiskCacheAvailable(dataSize)
}
