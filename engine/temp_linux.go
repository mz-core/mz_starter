//go:build linux

package engine

import (
	"fmt"
	"os"
	"syscall"
)

func defaultTempRoot() (string, error) { return os.TempDir(), nil }

// makePrivateTemp only trusts a root-owned sticky shared directory or a private user-owned directory.
func makePrivateTemp(root string) (string, error) {
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("untrusted temporary root")
	}
	info, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("temporary ownership unavailable")
	}
	uid := uint32(os.Geteuid())
	if info.Uid != uid && info.Uid != 0 {
		return "", fmt.Errorf("untrusted temporary owner")
	}
	if st.Mode().Perm()&0022 != 0 && st.Mode()&os.ModeSticky == 0 {
		return "", fmt.Errorf("untrusted temporary permissions")
	}
	return os.MkdirTemp(root, "mz-starter-verify-")
}
