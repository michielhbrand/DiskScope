//go:build !linux && !darwin && !windows

package scan

import "os"

func platformAllocated(info os.FileInfo) (int64, bool) { return 0, false }

func platformID(info os.FileInfo) (dev, ino uint64, ok bool) { return 0, 0, false }

func platformNlink(info os.FileInfo) uint64 { return 1 }

// DiskUsage is unavailable on this platform.
func DiskUsage(path string) DiskInfo {
	return DiskInfo{Err: "disk capacity unavailable on this platform"}
}
