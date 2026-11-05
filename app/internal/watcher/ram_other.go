//go:build !linux && !windows && !darwin

package watcher

func totalRAMBytes() uint64 {
	return 0
}
