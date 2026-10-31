//go:build linux || darwin

package scan

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func platformAllocated(info os.FileInfo) (int64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int64(st.Blocks) * 512, true
}

func platformID(info os.FileInfo) (dev, ino uint64, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return uint64(st.Dev), uint64(st.Ino), true
}

func platformNlink(info os.FileInfo) uint64 {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 1
	}
	return uint64(st.Nlink)
}

// DiskUsage returns capacity information for the filesystem containing path.
func DiskUsage(path string) DiskInfo {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return DiskInfo{Err: err.Error()}
	}
	bsize := int64(st.Bsize)
	total := int64(st.Blocks) * bsize
	used := int64(st.Blocks-st.Bfree) * bsize
	free := int64(st.Bavail) * bsize
	return DiskInfo{Total: total, Used: used, Free: free, Ok: true}
}
