package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	errInvalidDirectory           = fmt.Errorf("invalid_directory")
	errTargetInsideExistingServer = fmt.Errorf("target_inside_existing_server")
	errDestinationUnavailable     = fmt.Errorf("destination_unavailable")
	errDestinationNotEmpty        = fmt.Errorf("destination_not_empty")
	errPublicationRestore         = fmt.Errorf("publication_failed_empty_destination_not_restored")
)

type publicationError struct{ err error }

func (e *publicationError) Error() string { return "publication_failed: " + e.err.Error() }
func (e *publicationError) Unwrap() error { return e.err }

// validateInstallDestination requires a target that is absent or an empty directory,
// outside any existing server tree. It returns the absolute target.
func validateInstallDestination(target string) (string, error) {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	if parent == absolute {
		return "", errInvalidDirectory
	}
	for p := parent; ; p = filepath.Dir(p) {
		if strings.EqualFold(filepath.Base(p), "resources") {
			return "", errTargetInsideExistingServer
		}
		for _, marker := range []string{"server.cfg", "fxmanifest.lua"} {
			if _, e := os.Lstat(filepath.Join(p, marker)); e == nil {
				return "", errTargetInsideExistingServer
			} else if !os.IsNotExist(e) {
				return "", errDestinationUnavailable
			}
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return absolute, nil
}

// writeBootstrap copies the canonical embedded models into the staging tree.
// Only cfg/resources.cfg is transformed; every other file stays byte-identical.
func writeBootstrap(staging string, templates fs.FS, bootReady map[string]bool, notice, reason string) ([]string, error) {
	var files []string
	for _, name := range BootstrapFiles {
		b, err := fs.ReadFile(templates, name)
		if err != nil {
			return nil, err
		}
		if name == "cfg/resources.cfg" {
			b = []byte(notice + commentUnreadyBoot(string(b), bootReady, reason))
		}
		p := filepath.Join(staging, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, err
		}
		if err = os.WriteFile(p, b, 0600); err != nil {
			return nil, err
		}
		files = append(files, name)
	}
	return files, nil
}

// publishInstallStage publishes a fully prepared staging directory with a no-replace
// rename. An empty destination may be removed, never a populated one.
func publishInstallStage(stage, absolute string) error {
	if st, e := os.Lstat(absolute); e == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
		return errDestinationNotEmpty
	} else if e != nil && !os.IsNotExist(e) {
		return errDestinationUnavailable
	}
	removed := false
	if e := removeEmptyDirectory(absolute); e == nil {
		removed = true
	} else if !os.IsNotExist(e) {
		return errDestinationNotEmpty
	}
	if e := publishNoReplace(stage, absolute); e != nil {
		if removed {
			if restore := os.Mkdir(absolute, 0700); restore != nil && !os.IsExist(restore) {
				return errPublicationRestore
			}
		}
		return &publicationError{e}
	}
	return nil
}

// bootReadyFor reports whether a boot line may stay active. A dependency is satisfied
// when it was installed or is provided by the FXServer artifacts.
func bootReadyFor(byID map[string]Resource, installed, system map[string]bool) map[string]bool {
	ready := map[string]bool{}
	var canBoot func(string, map[string]bool) bool
	canBoot = func(id string, visiting map[string]bool) bool {
		if system[id] {
			return true
		}
		if !installed[id] || visiting[id] {
			return false
		}
		resource, exists := byID[id]
		if !exists {
			return false
		}
		visiting[id] = true
		defer delete(visiting, id)
		for _, dependency := range resource.Dependencies {
			if !canBoot(dependency, visiting) {
				return false
			}
		}
		return true
	}
	for id := range installed {
		ready[id] = canBoot(id, map[string]bool{})
	}
	for id := range system {
		ready[id] = true
	}
	return ready
}
