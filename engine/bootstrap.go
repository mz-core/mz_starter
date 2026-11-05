package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var BootstrapFiles = []string{"server.cfg", "cfg/base.cfg", "cfg/database.cfg", "cfg/endpoints.cfg", "cfg/onesync.cfg", "cfg/permissions.cfg", "cfg/phone.cfg", "cfg/resources.cfg", "cfg/tags.cfg", "logo.png"}
var BootstrapConfigurations = []string{"license_key", "database_connection", "server_endpoints", "owner_identifiers", "phone_urls_and_secret"}
var PlanningPhases = []string{"preflight", "bootstrap", "configuration", "external_dependencies", "mz_resources", "final_validation"}

// ValidateBootstrap verifies the embedded canonical inputs and rejects changed sensitive defaults.
func ValidateBootstrap(c Catalog, files fs.FS) error {
	for _, name := range append(append([]string{}, BootstrapFiles...), "patches/pma-voice-server-authoritative-calls.patch") {
		b, err := fs.ReadFile(files, name)
		if err != nil || len(b) == 0 {
			return fmt.Errorf("bootstrap_file_unavailable: %s", name)
		}
	}
	checks := map[string][]string{
		"cfg/base.cfg":        {`set sv_licenseKey "COLOQUE_SUA_KEY"`},
		"cfg/database.cfg":    {`set mysql_connection_string "mysql://root:senha@127.0.0.1/mz_starter?charset=utf8mb4"`},
		"cfg/permissions.cfg": {`add_principal identifier.fivem:SEU_ID_FIVEM_AQUI group.mz_owner`, `add_principal identifier.fivem:ID_DO_SOCIO_AQUI group.mz_owner`},
		"cfg/phone.cfg":       {`set mz_phone_media_issuer_secret "ALTERE_PARA_UM_SEGREDO_FORTE_COM_32_CARACTERES"`},
	}
	for name, required := range checks {
		b, _ := fs.ReadFile(files, name)
		content := string(b)
		for _, line := range required {
			if !strings.Contains(content, line) {
				return fmt.Errorf("bootstrap_sensitive_default_changed: %s", name)
			}
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				continue
			}
			lower := strings.ToLower(line)
			sensitive := strings.Contains(lower, "sv_licensekey") || strings.Contains(lower, "mysql_connection_string") || strings.HasPrefix(lower, "add_principal ") || strings.Contains(lower, "mz_phone_media_issuer_secret") || strings.Contains(lower, "mz_phone_pin_pepper")
			if sensitive {
				allowed := false
				for _, expected := range required {
					if line == expected {
						allowed = true
					}
				}
				if !allowed {
					return fmt.Errorf("bootstrap_sensitive_default_changed: %s", name)
				}
			}
		}
	}
	for _, name := range BootstrapFiles {
		if !strings.HasSuffix(name, ".cfg") {
			continue
		}
		b, _ := fs.ReadFile(files, name)
		for _, raw := range strings.Split(string(b), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
				continue
			}
			lower := strings.ToLower(line)
			sensitive := strings.Contains(lower, "password") || strings.Contains(lower, "passwd") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "licensekey") || strings.Contains(lower, "connection_string") || strings.HasPrefix(lower, "add_principal ") || strings.Contains(lower, "api_key")
			if sensitive {
				allowed := false
				for _, expected := range checks[name] {
					if line == expected {
						allowed = true
					}
				}
				if !allowed {
					return fmt.Errorf("bootstrap_sensitive_default_changed: %s", name)
				}
			}
		}
	}
	f, err := files.Open("cfg/resources.cfg")
	if err != nil {
		return err
	}
	defer f.Close()
	boot, err := ParseBoot(f)
	if err != nil {
		return err
	}
	if len(boot) != len(c.Resources) {
		return fmt.Errorf("bootstrap_catalog_mismatch")
	}
	for i, id := range boot {
		if id != c.Resources[i].ID {
			return fmt.Errorf("bootstrap_catalog_mismatch: %s", id)
		}
	}
	return nil
}

// PlanInstallation only observes the target; explicit modes never overwrite existing files.
func PlanInstallation(c Catalog, files fs.FS, server, workspace, mode, profile string, remote map[string]QueryResult) (Plan, error) {
	if strings.TrimSpace(server) == "" {
		return Plan{}, fmt.Errorf("invalid_request")
	}
	if profile == "" {
		profile = c.Profile
	}
	if profile != c.Profile {
		return Plan{}, fmt.Errorf("unsupported_profile")
	}
	if mode != "new" && mode != "existing" {
		return Plan{}, fmt.Errorf("installation_mode_required")
	}
	if err := ValidateBootstrap(c, files); err != nil {
		return Plan{}, err
	}
	root, err := filepath.Abs(server)
	if err != nil {
		return Plan{}, fmt.Errorf("invalid_directory")
	}
	st, err := os.Lstat(root)
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, fmt.Errorf("directory_unavailable")
	}
	if err == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
		return Plan{}, fmt.Errorf("directory_unavailable")
	}
	var boot []string
	if mode == "new" {
		if err == nil {
			entries, e := os.ReadDir(root)
			if e != nil {
				return Plan{}, fmt.Errorf("directory_unavailable")
			}
			if len(entries) != 0 {
				return Plan{}, fmt.Errorf("directory_not_empty")
			}
		}
		for _, r := range c.Resources {
			boot = append(boot, r.ID)
		}
	} else {
		if err != nil {
			return Plan{}, fmt.Errorf("existing_server_missing")
		}
		main, e := noSymlinkPath(root, "server.cfg")
		if e != nil {
			return Plan{}, fmt.Errorf("installation_ambiguous")
		}
		st, e := os.Stat(main)
		if e != nil || !st.Mode().IsRegular() {
			return Plan{}, fmt.Errorf("installation_ambiguous")
		}
		boot, e = ReadServerBoot(root)
		if e != nil || len(boot) == 0 {
			return Plan{}, fmt.Errorf("installation_ambiguous")
		}
	}
	p := BuildPlan(c, boot, root, workspace, remote)
	p.InstallationMode = mode
	p.Profile = profile
	p.Phases = append([]string{}, PlanningPhases...)
	if mode == "new" {
		p.BootstrapFiles = append([]string{}, BootstrapFiles...)
		p.RequiredConfiguration = append([]string{}, BootstrapConfigurations...)
	}
	return p, nil
}
