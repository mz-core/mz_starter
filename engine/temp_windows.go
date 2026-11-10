//go:build windows

package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var securityDLL = syscall.NewLazyDLL("advapi32.dll")
var getNamedSecurityInfo = securityDLL.NewProc("GetNamedSecurityInfoW")
var getACE = securityDLL.NewProc("GetAce")
var convertSDDL = securityDLL.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
var getDescriptorControl = securityDLL.NewProc("GetSecurityDescriptorControl")
var kernelDLL = syscall.NewLazyDLL("kernel32.dll")
var getDriveType = kernelDLL.NewProc("GetDriveTypeW")
var getVolumeInformation = kernelDLL.NewProc("GetVolumeInformationW")

type aclHeader struct {
	Revision  byte
	Reserved  byte
	Size      uint16
	Count     uint16
	Reserved2 uint16
}
type aceHeader struct {
	Type  byte
	Flags byte
	Size  uint16
}

func defaultTempRoot() (string, error) { return os.UserCacheDir() }

func currentSID() (string, error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String()
}

// trustedDirectory rejects unknown writable principals, owners and reparse points.
func trustedDirectory(name, userSID string, direct bool) error {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return err
	}
	if attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("reparse point")
	}
	var owner, dacl, descriptor uintptr
	result, _, _ := getNamedSecurityInfo.Call(uintptr(unsafe.Pointer(p)), 1, 5, uintptr(unsafe.Pointer(&owner)), 0, uintptr(unsafe.Pointer(&dacl)), 0, uintptr(unsafe.Pointer(&descriptor)))
	if result != 0 {
		return syscall.Errno(result)
	}
	if descriptor == 0 {
		return fmt.Errorf("missing security descriptor")
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	if owner == 0 || dacl == 0 {
		return fmt.Errorf("missing owner or DACL")
	}
	ownerName, err := (*syscall.SID)(unsafe.Pointer(owner)).String()
	if err != nil {
		return err
	}
	trusted := func(s string) bool {
		return strings.EqualFold(s, userSID) || s == "S-1-5-18" || s == "S-1-5-32-544" || s == "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"
	}
	if !trusted(ownerName) {
		return fmt.Errorf("untrusted owner")
	}
	acl := (*aclHeader)(unsafe.Pointer(dacl))
	if acl.Size < 8 {
		return fmt.Errorf("invalid ACL")
	}
	for i := uint32(0); i < uint32(acl.Count); i++ {
		var ace uintptr
		ok, _, callErr := getACE.Call(dacl, uintptr(i), uintptr(unsafe.Pointer(&ace)))
		if ok == 0 {
			return callErr
		}
		if ace == 0 {
			return fmt.Errorf("missing ACE")
		}
		head := (*aceHeader)(unsafe.Pointer(ace))
		if head.Flags&0x08 != 0 || head.Type == 1 {
			continue
		}
		if head.Type != 0 || head.Size < 16 {
			return fmt.Errorf("unsupported ACE")
		}
		mask := *(*uint32)(unsafe.Pointer(ace + 4))
		sid, err := (*syscall.SID)(unsafe.Pointer(ace + 8)).String()
		if err != nil {
			return err
		}
		if trusted(sid) {
			continue
		}
		// DELETE_CHILD, DELETE, WRITE_DAC, WRITE_OWNER and GENERIC_ALL can replace an ancestor.
		dangerous := uint32(0x00000040 | 0x00010000 | 0x00040000 | 0x00080000 | 0x10000000 | 0x40000000 | 0x00000100)
		if direct {
			dangerous |= 0x00000002 | 0x00000004 | 0x00000010
		}
		if mask&dangerous != 0 {
			return fmt.Errorf("untrusted writable principal")
		}
	}
	return nil
}

func verifyPrivateDirectory(name, userSID string) error {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	var owner, dacl, descriptor uintptr
	result, _, _ := getNamedSecurityInfo.Call(uintptr(unsafe.Pointer(p)), 1, 5, uintptr(unsafe.Pointer(&owner)), 0, uintptr(unsafe.Pointer(&dacl)), 0, uintptr(unsafe.Pointer(&descriptor)))
	if result != 0 {
		return syscall.Errno(result)
	}
	if descriptor == 0 {
		return fmt.Errorf("missing descriptor")
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	if owner == 0 || dacl == 0 {
		return fmt.Errorf("missing owner or DACL")
	}
	sid, err := (*syscall.SID)(unsafe.Pointer(owner)).String()
	if err != nil || !strings.EqualFold(sid, userSID) {
		return fmt.Errorf("unexpected owner")
	}
	var control uint16
	var revision uint32
	ok, _, callErr := getDescriptorControl.Call(descriptor, uintptr(unsafe.Pointer(&control)), uintptr(unsafe.Pointer(&revision)))
	if ok == 0 || control&0x1000 == 0 {
		return fmt.Errorf("unprotected DACL: %v", callErr)
	}
	acl := (*aclHeader)(unsafe.Pointer(dacl))
	if acl.Count != 1 {
		return fmt.Errorf("unexpected ACE count")
	}
	var ace uintptr
	ok, _, callErr = getACE.Call(dacl, 0, uintptr(unsafe.Pointer(&ace)))
	if ok == 0 || ace == 0 {
		return fmt.Errorf("missing ACE: %v", callErr)
	}
	head := (*aceHeader)(unsafe.Pointer(ace))
	if head.Type != 0 || head.Flags&0x08 != 0 || head.Size < 16 {
		return fmt.Errorf("unexpected ACE")
	}
	sid, err = (*syscall.SID)(unsafe.Pointer(ace + 8)).String()
	if err != nil || !strings.EqualFold(sid, userSID) {
		return fmt.Errorf("unexpected ACE principal")
	}
	return nil
}

func makePrivateTemp(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	volume := filepath.VolumeName(root)
	if len(volume) != 2 || volume[1] != ':' {
		return "", fmt.Errorf("unsupported volume")
	}
	volumePtr, err := syscall.UTF16PtrFromString(volume + `\`)
	if err != nil {
		return "", err
	}
	drive, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(volumePtr)))
	if drive != 3 {
		return "", fmt.Errorf("non-local volume")
	}
	var flags uint32
	ok, _, volumeErr := getVolumeInformation.Call(uintptr(unsafe.Pointer(volumePtr)), 0, 0, 0, 0, uintptr(unsafe.Pointer(&flags)), 0, 0)
	if ok == 0 || flags&0x8 == 0 {
		return "", fmt.Errorf("persistent ACL unavailable: %v", volumeErr)
	}
	sid, err := currentSID()
	if err != nil {
		return "", err
	}
	for p := root; ; p = filepath.Dir(p) {
		if err := trustedDirectory(p, sid, p == root); err != nil {
			return "", err
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
	}
	sddl, err := syscall.UTF16PtrFromString("O:" + sid + "D:P(A;OICI;FA;;;" + sid + ")")
	if err != nil {
		return "", err
	}
	var descriptor uintptr
	ok, _, callErr := convertSDDL.Call(uintptr(unsafe.Pointer(sddl)), 1, uintptr(unsafe.Pointer(&descriptor)), 0)
	if ok == 0 {
		return "", callErr
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	var control uint16
	var revision uint32
	ok, _, callErr = getDescriptorControl.Call(descriptor, uintptr(unsafe.Pointer(&control)), uintptr(unsafe.Pointer(&revision)))
	if ok == 0 || control&0x1000 == 0 {
		return "", fmt.Errorf("unprotected DACL: %v", callErr)
	}
	sa := syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(syscall.SecurityAttributes{})), SecurityDescriptor: descriptor}
	for attempt := 0; attempt < 8; attempt++ {
		var random [16]byte
		if _, err = rand.Read(random[:]); err != nil {
			return "", err
		}
		dir := filepath.Join(root, "mz-starter-verify-"+hex.EncodeToString(random[:]))
		p, e := syscall.UTF16PtrFromString(dir)
		if e != nil {
			return "", e
		}
		if e = syscall.CreateDirectory(p, &sa); e == nil {
			if e = verifyPrivateDirectory(dir, sid); e != nil {
				_ = os.Remove(dir)
				return "", e
			}
			return dir, nil
		} else if !os.IsExist(e) {
			return "", e
		}
	}
	return "", fmt.Errorf("temporary name collision")
}
