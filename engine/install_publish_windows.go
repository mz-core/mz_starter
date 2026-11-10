//go:build windows

package engine

import "syscall"

func removeEmptyDirectory(name string) error {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	return syscall.RemoveDirectory(p)
}

func publishNoReplace(from, to string) error {
	a, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	b, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return syscall.MoveFile(a, b)
}
