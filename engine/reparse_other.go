//go:build !windows

package engine

// Symlinks are already rejected through Lstat on this platform.
func isReparsePoint(string) bool { return false }
