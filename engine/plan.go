package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Issue struct {
	Code     string `json:"code"`
	Resource string `json:"resource,omitempty"`
	Detail   string `json:"detail"`
}
type Step struct {
	ID               string   `json:"id"`
	Action           string   `json:"action"`
	Status           string   `json:"status"`
	Found            bool     `json:"found"`
	InstalledVersion string   `json:"installed_version,omitempty"`
	DeclaredVersion  string   `json:"declared_version,omitempty"`
	AvailableVersion string   `json:"available_version,omitempty"`
	Provenance       string   `json:"provenance"`
	RemoteStatus     string   `json:"remote_status"`
	Blockers         []string `json:"blockers"`
}
type Plan struct {
	SchemaVersion         int      `json:"schema_version"`
	Mode                  string   `json:"mode"`
	Server                string   `json:"server"`
	Steps                 []Step   `json:"steps"`
	Issues                []Issue  `json:"issues"`
	Ready                 bool     `json:"ready"`
	InstallationMode      string   `json:"installation_mode,omitempty"`
	Profile               string   `json:"profile,omitempty"`
	BootstrapFiles        []string `json:"bootstrap_files,omitempty"`
	RequiredConfiguration []string `json:"required_configuration,omitempty"`
	Phases                []string `json:"phases,omitempty"`
}

type managedRecord struct {
	SchemaVersion int    `json:"schema_version"`
	ID            string `json:"id"`
	Version       string `json:"version"`
	SHA256        string `json:"sha256"`
}

func noSymlinkPath(root string, parts ...string) (string, error) {
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if st, e := os.Lstat(base); e == nil && st.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("raiz simbolica recusada")
	} else if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	current := base
	all := append([]string{}, parts...)
	for _, part := range all {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\`) || filepath.VolumeName(part) != "" {
			return "", fmt.Errorf("segmento inseguro")
		}
		current = filepath.Join(current, part)
		st, e := os.Lstat(current)
		if e != nil {
			if os.IsNotExist(e) {
				continue
			}
			return "", e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("link simbolico recusado")
		}
	}
	rel, e := filepath.Rel(base, current)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("caminho fora da raiz")
	}
	return current, nil
}

func BuildPlan(c Catalog, boot []string, server, workspace string, remote map[string]QueryResult) Plan {
	p := Plan{SchemaVersion: 1, Mode: "dry_run", Server: server, Steps: []Step{}, Issues: []Issue{}, Ready: false}
	byID := map[string]Resource{}
	position := map[string]int{}
	bootSeen := map[string]bool{}
	for _, r := range c.Resources {
		byID[r.ID] = r
	}
	for i, id := range boot {
		if bootSeen[id] {
			p.Issues = append(p.Issues, Issue{"duplicate_boot", id, "recurso duplicado no boot"})
		} else {
			position[id] = i
		}
		bootSeen[id] = true
		if _, ok := byID[id]; !ok {
			p.Issues = append(p.Issues, Issue{"missing_catalog", id, "recurso do boot ausente no catalogo"})
		} else if i >= len(c.Resources) || c.Resources[i].ID != id {
			p.Issues = append(p.Issues, Issue{"boot_order_mismatch", id, "ordem diverge do catalogo; nenhuma alteracao aplicada"})
		}
	}
	for _, r := range c.Resources {
		if !bootSeen[r.ID] {
			p.Issues = append(p.Issues, Issue{"missing_boot", r.ID, "recurso do catalogo ausente no boot"})
		}
	}
	colors := map[string]int{}
	stack := []string{}
	reported := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		colors[id] = 1
		stack = append(stack, id)
		r := byID[id]
		for _, d := range r.Dependencies {
			if _, ok := byID[d]; !ok {
				p.Issues = append(p.Issues, Issue{"dependency_missing", id, d})
				continue
			}
			if !bootSeen[d] {
				p.Issues = append(p.Issues, Issue{"dependency_not_in_boot", id, d})
			}
			if bootSeen[d] && bootSeen[id] && position[d] > position[id] {
				p.Issues = append(p.Issues, Issue{"dependency_order", id, d + " aparece depois no boot"})
			}
			if colors[d] == 1 && !reported[d] {
				p.Issues = append(p.Issues, Issue{"dependency_cycle", id, strings.Join(append(append([]string{}, stack...), d), " -> ")})
				reported[d] = true
			} else if colors[d] == 0 {
				visit(d)
			}
		}
		stack = stack[:len(stack)-1]
		colors[id] = 2
	}
	for _, r := range c.Resources {
		if colors[r.ID] == 0 {
			visit(r.ID)
		}
	}
	for _, id := range boot {
		r, ok := byID[id]
		if !ok {
			continue
		}
		step := Step{ID: id, Action: "inspect", Status: "not_installed", DeclaredVersion: r.Version, Provenance: "unknown", RemoteStatus: "offline", Blockers: []string{}}
		if r.Tag != "" && r.Asset != "" {
			step.AvailableVersion = r.Version
		}
		if q, ok := remote[id]; ok {
			step.RemoteStatus = q.Status
		}
		path, err := noSymlinkPath(server, "resources", r.Group, id, "fxmanifest.lua")
		if err != nil {
			step.Status = "configuration_conflict"
			step.Blockers = append(step.Blockers, "unsafe_path")
		} else if st, e := os.Lstat(path); e == nil && st.Mode().IsRegular() {
			step.Found = true
			data, e := os.ReadFile(path)
			if e != nil {
				step.Status = "configuration_conflict"
				step.Blockers = append(step.Blockers, "manifest_unreadable")
			} else {
				v, _ := ParseManifest(data)
				if HasDynamicDependencies(data) {
					step.Blockers = append(step.Blockers, "manifest_dynamic_unverified")
				}
				step.InstalledVersion = v
				if v == "" {
					step.Status = "version_unknown"
				} else if step.AvailableVersion == "" {
					step.Status = "available_version_unknown"
				} else if v == step.AvailableVersion {
					step.Status = "version_matches_unverified"
				} else if semverPattern.MatchString(v) && semverPattern.MatchString(step.AvailableVersion) && semverGreater(step.AvailableVersion, v) {
					step.Status = "update_available"
				} else {
					step.Status = "version_incompatible"
				}
			}
		} else if e != nil && !os.IsNotExist(e) {
			step.Status = "configuration_conflict"
			step.Blockers = append(step.Blockers, "manifest_unreadable")
		} else if e == nil {
			step.Status = "configuration_conflict"
			step.Blockers = append(step.Blockers, "manifest_not_regular")
		}
		if step.InstalledVersion != "" {
			if marker, e := noSymlinkPath(server, "resources", r.Group, id, ".mz-starter.json"); e == nil {
				if st, e := os.Lstat(marker); e == nil && st.Mode().IsRegular() && st.Size() <= 16384 {
					if data, e := os.ReadFile(marker); e == nil {
						var m managedRecord
						if json.Unmarshal(data, &m) == nil && m.SchemaVersion == 1 && m.ID == id && m.Version == step.InstalledVersion && hashPattern.MatchString(m.SHA256) {
							step.Provenance = "managed_record_unverified"
						}
					}
				}
			}
		}
		if r.Source == "protected" {
			step.Blockers = append(step.Blockers, "protected_distribution_required")
		} else if r.Visibility == "private" {
			step.Blockers = append(step.Blockers, "private_distribution_authorization_unverified")
			q, ok := remote[id]
			if !ok || (q.Status != "ok" && q.Status != "release_missing" && q.Status != "no_recognized_releases") {
				step.Blockers = append(step.Blockers, "authorization_required")
			}
		} else if r.Source == "unavailable" {
			step.Blockers = append(step.Blockers, "distribution_unavailable")
		}
		if r.Type != "fivem_resource" {
			step.Status = "external_unverified"
			step.Blockers = append(step.Blockers, "external_contract_unverified")
		}
		if r.Type == "fivem_resource" && step.Status != "not_installed" {
			for _, file := range r.RequiredFiles {
				parts := append([]string{"resources", r.Group, id}, strings.Split(file, "/")...)
				path, e := noSymlinkPath(server, parts...)
				if e != nil {
					step.Blockers = append(step.Blockers, "unsafe_required_file:"+file)
				} else if st, e := os.Lstat(path); e != nil || !st.Mode().IsRegular() {
					step.Blockers = append(step.Blockers, "required_file_missing:"+file)
				}
			}
		}
		if _, online := remote[id]; !online && (r.Availability == "release_missing" || r.Availability == "asset_missing_or_ambiguous" || r.Availability == "repository_missing_or_unknown") {
			step.Blockers = append(step.Blockers, "catalog_snapshot_"+r.Availability)
		}
		if r.Availability == "provenance_conflict" || r.Availability == "version_conflict" {
			p.Issues = append(p.Issues, Issue{"catalog_audit_snapshot", id, r.Availability + "; confirmar revisao atual"})
		}
		if r.Type == "fivem_resource" {
			step.Blockers = append(step.Blockers, "integrity_and_provenance_not_proven")
			if strings.HasPrefix(r.Tag, id+"-v") {
				step.Blockers = append(step.Blockers, "legacy_tag_requires_review")
			}
		}
		if q, ok := remote[id]; ok {
			if q.Status != "ok" {
				if q.Status == "release_missing" || q.Status == "no_recognized_releases" {
					step.AvailableVersion = ""
					if step.InstalledVersion != "" {
						step.Status = "release_unavailable"
					}
				}
				step.Blockers = append(step.Blockers, "github_"+q.Status)
			} else {
				step.AvailableVersion = ""
				found := 0
				chosen := false
				versions := map[string]int{}
				for _, rel := range q.Releases {
					versions[rel.Version]++
				}
				for _, v := range q.Releases {
					if v.Version == r.Version {
						found++
						if v.Status != "integrity_unverified" {
							step.Blockers = append(step.Blockers, v.Status)
						}
					}
					if !chosen && (r.Channel == "prerelease" || !v.Prerelease) && v.Status == "integrity_unverified" {
						if v.LegacyTag {
							step.Blockers = append(step.Blockers, "legacy_tag_requires_review")
						}
						if versions[v.Version] > 1 {
							step.AvailableVersion = ""
							step.Blockers = append(step.Blockers, "release_ambiguous")
						} else {
							step.AvailableVersion = v.Version
						}
						chosen = true
					}
				}
				if found == 0 {
					step.Blockers = append(step.Blockers, "declared_release_not_found")
				}
				if found > 1 {
					step.Blockers = append(step.Blockers, "release_ambiguous")
				}
				if r.Availability == "version_conflict" && found == 1 && step.AvailableVersion == r.Version {
					p.Issues = append(p.Issues, Issue{"metadata_conflict_changed", id, "release correspondente observado; procedencia ainda requer revisao"})
				}
				if step.InstalledVersion != "" && step.AvailableVersion == "" {
					step.Status = "release_unavailable"
				}
				if step.InstalledVersion != "" && semverPattern.MatchString(step.InstalledVersion) && semverPattern.MatchString(step.AvailableVersion) {
					if semverGreater(step.AvailableVersion, step.InstalledVersion) {
						step.Status = "update_available"
					} else if semverGreater(step.InstalledVersion, step.AvailableVersion) {
						step.Status = "version_incompatible"
					} else {
						step.Status = "version_matches_unverified"
					}
				}
			}
		}
		if q, ok := remote[id]; ok && r.SHA256 != "" {
			for _, rel := range q.Releases {
				if rel.Tag == r.Tag {
					for _, asset := range rel.Assets {
						if asset.Name == r.Asset && asset.Digest != "" && !strings.EqualFold(asset.Digest, r.SHA256) {
							step.Blockers = append(step.Blockers, "catalog_digest_changed")
						}
					}
				}
			}
		}
		if workspace != "" && r.Type == "fivem_resource" {
			mp, e := noSymlinkPath(workspace, id, "fxmanifest.lua")
			if e != nil {
				p.Issues = append(p.Issues, Issue{"workspace_manifest_unavailable", id, "caminho nao verificavel"})
			} else if data, e := os.ReadFile(mp); e != nil {
				p.Issues = append(p.Issues, Issue{"workspace_manifest_unavailable", id, "manifesto indisponivel"})
			} else {
				v, deps := ParseManifest(data)
				if HasDynamicDependencies(data) {
					p.Issues = append(p.Issues, Issue{"manifest_dynamic_unverified", id, "dependencias dinamicas exigem revisao"})
				}
				if v != r.Version {
					p.Issues = append(p.Issues, Issue{"catalog_manifest_version", id, "versao declarada difere do catalogo"})
				}
				a := append([]string{}, deps...)
				b := append([]string{}, r.Dependencies...)
				sort.Strings(a)
				sort.Strings(b)
				if strings.Join(a, "\x00") != strings.Join(b, "\x00") {
					p.Issues = append(p.Issues, Issue{"catalog_manifest_dependencies", id, "dependencias diferem do catalogo"})
				}
			}
		}
		switch step.Status {
		case "not_installed":
			step.Action = "would_install"
		case "update_available":
			step.Action = "would_update"
		case "version_matches_unverified":
			step.Action = "inspect_existing"
		}
		p.Steps = append(p.Steps, step)
	}
	states := map[string]string{}
	for _, s := range p.Steps {
		states[s.ID] = s.Status
	}
	for _, s := range p.Steps {
		if s.InstalledVersion == "" {
			continue
		}
		for _, d := range byID[s.ID].Dependencies {
			if states[d] == "not_installed" {
				p.Issues = append(p.Issues, Issue{"dependency_not_installed", s.ID, d})
			}
		}
	}
	return p
}
