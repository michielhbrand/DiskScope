//go:build windows

package scan

import (
	"os"

	"golang.org/x/sys/windows"
)

func platformAllocated(info os.FileInfo) (int64, bool) { return 0, false }

func platformID(info os.FileInfo) (dev, ino uint64, ok bool) { return 0, 0, false }

func platformNlink(info os.FileInfo) uint64 { return 1 }

// DiskUsage returns capacity information for the volume containing path.
func DiskUsage(path string) DiskInfo {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return DiskInfo{Err: err.Error()}
	}
	var freeBytes, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeBytes, &totalBytes, &totalFree); err != nil {
		return DiskInfo{Err: err.Error()}
	}
	return DiskInfo{
		Total: int64(totalBytes),
		Used:  int64(totalBytes - totalFree),
		Free:  int64(freeBytes),
		Ok:    true,
	}
}
