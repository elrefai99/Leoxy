//go:build darwin

package watcher

import "golang.org/x/sys/unix"

func totalRAMBytes() uint64 {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return total
}
