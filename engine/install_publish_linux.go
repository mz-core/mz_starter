//go:build linux

package engine

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

func removeEmptyDirectory(name string) error { return syscall.Rmdir(name) }

func publishNoReplace(from, to string) error {
	var nr uintptr
	switch runtime.GOARCH {
	case "amd64":
		nr = 316
	case "arm64":
		nr = 276
	default:
		return fmt.Errorf("atomic_publish_unsupported")
	}
	a, err := syscall.BytePtrFromString(from)
	if err != nil {
		return err
	}
	b, err := syscall.BytePtrFromString(to)
	if err != nil {
		return err
	}
	_, _, e := syscall.Syscall6(nr, ^uintptr(99), uintptr(unsafe.Pointer(a)), ^uintptr(99), uintptr(unsafe.Pointer(b)), 1, 0)
	if e != 0 {
		return e
	}
	return nil
}
