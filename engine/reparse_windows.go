//go:build windows

package engine

import "syscall"

func isReparsePoint(path string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return true
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return true
	}
	return attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
