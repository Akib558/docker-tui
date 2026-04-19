//go:build !linux

package docker

// SystemMemory returns zero values on non-Linux platforms.
// TODO: add gopsutil or syscall-based implementation for macOS/Windows.
func GetSystemMemory() SystemMemory { return SystemMemory{} }

// GetSystemLoad returns zero values on non-Linux platforms.
func GetSystemLoad() SystemLoad { return SystemLoad{} }
