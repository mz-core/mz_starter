package engine

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type InstallProgress struct {
	State     string   `json:"state"`
	Resource  string   `json:"resource,omitempty"`
	Version   string   `json:"version,omitempty"`
	SHA256    string   `json:"sha256,omitempty"`
	Received  int64    `json:"received,omitempty"`
	Size      int64    `json:"size,omitempty"`
	Installed []string `json:"installed,omitempty"`
	Blocked   []string `json:"blocked,omitempty"`
	Files     []string `json:"files,omitempty"`
	Detail    string   `json:"detail,omitempty"`
}

type InstallOptions struct {
	GitHub   GitHub
	Verify   VerifyOptions
	Progress func(InstallProgress)
}

const partialBootstrapNotice = `# ---------------------------------------------------------------------------
# MZ STARTER - INSTALACAO PARCIAL EXPERIMENTAL
# Este diretorio NAO e um servidor pronto para iniciar.
# Apenas os resources selecionados foram instalados. FXServer, dependencias
# externas e demais resources obrigatorios podem estar ausentes.
# Linhas de boot de resources indisponiveis, nao instalados ou com dependencia
# ausente foram comentadas abaixo. Nao use este diretorio em producao.
# ---------------------------------------------------------------------------
`

// catalogDigest returns the lowercase digest of exactly one canonical sha256 value.
// Duplicated prefixes, truncation and non-hex values are rejected instead of trimmed.
func catalogDigest(value string) (string, bool) {
	trimmed := strings.TrimPrefix(value, "sha256:")
	if len(trimmed) != 64 || !hashPattern.MatchString(trimmed) {
		return "", false
	}
	return strings.ToLower(trimmed), true
}

// commentUnreadyBoot keeps the canonical order but never leaves a boot line active
// for a resource whose direct or transitive dependencies were not installed.
func commentUnreadyBoot(content string, bootReady map[string]bool, reason string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && (strings.EqualFold(fields[0], "ensure") || strings.EqualFold(fields[0], "start")) && !bootReady[fields[1]] {
			lines[i] = "# " + line + " # " + reason
		}
	}
	return strings.Join(lines, "\n")
}

// InstallPartial installs exactly the explicitly selected public resources, never a complete profile.
// The destination must not contain files; only a fully validated staging directory is published.
func InstallPartial(ctx context.Context, c Catalog, templates fs.FS, target string, ids []string, opts InstallOptions) (result InstallProgress, err error) {
	result.State = "preflight"
	emit := func() {
		if opts.Progress != nil {
			opts.Progress(result)
		}
	}
	defer func() {
		if err != nil {
			if ctx.Err() != nil {
				result.State = "cancelled"
			} else {
				result.State = "failed"
			}
			result.Detail = err.Error()
			emit()
		}
	}()
	if len(ids) == 0 {
		return result, fmt.Errorf("no_resources_selected")
	}
	if _, err = PlanInstallation(c, templates, target, "", "new", "", nil); err != nil {
		return result, err
	}
	absolute, err := validateInstallDestination(target)
	if err != nil {
		return result, err
	}
	parent := filepath.Dir(absolute)
	selected := map[string]bool{}
	var resources []Resource
	var prepared, files []string
	for _, id := range ids {
		if selected[id] {
			return result, fmt.Errorf("duplicate_resource")
		}
		selected[id] = true
	}
	for _, r := range c.Resources {
		if selected[r.ID] {
			version, _, validTag := ParseTag(r.ID, r.Tag)
			_, validDigest := catalogDigest(r.SHA256)
			if r.Type != "fivem_resource" || r.Source != "github_release" || r.Visibility != "public" || !validTag || version != r.Version || !validDigest || r.Availability == "provenance_conflict" || r.Availability == "version_conflict" {
				return result, fmt.Errorf("resource_unavailable:%s", r.ID)
			}
			resources = append(resources, r)
		} else {
			result.Blocked = append(result.Blocked, r.ID+":"+r.Availability)
		}
	}
	if len(resources) != len(ids) {
		return result, fmt.Errorf("resource_not_found")
	}
	emit()
	stage, err := makePrivateTemp(parent)
	if err != nil {
		return result, fmt.Errorf("staging_unavailable: %w", err)
	}
	defer func() {
		if stage != "" {
			if e := os.RemoveAll(stage); e != nil && err == nil {
				err = fmt.Errorf("staging_cleanup_failed")
			}
		}
	}()
	for _, r := range resources {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		expectedDigest, ok := catalogDigest(r.SHA256)
		if !ok {
			return result, fmt.Errorf("resource_unavailable:%s", r.ID)
		}
		result.State, result.Resource = "downloading", r.ID
		emit()
		v := opts.Verify
		v.ForbiddenRoots = append(append([]string{}, v.ForbiddenRoots...), absolute)
		v.Progress = func(p PackageReport) {
			result.State, result.Received, result.Size = p.State, p.Received, p.Size
			emit()
		}
		v.PermitIncomplete = func(p PackageReport) bool { return permittedPartial(r, p) }
		extracted := false
		v.OnVerified = func(ctx context.Context, archive string) error {
			file, e := os.Open(archive)
			if e != nil {
				return e
			}
			defer file.Close()
			info, e := file.Stat()
			if e != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("staging_archive_invalid")
			}
			h := sha256.New()
			read, e := io.Copy(h, contextReader{ctx, file})
			if e != nil {
				return e
			}
			if read != info.Size() {
				return fmt.Errorf("staging_archive_changed")
			}
			if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expectedDigest) {
				return fmt.Errorf("catalog_digest_mismatch")
			}
			// Extraction uses the same open handle that was just hashed, so the
			// executed bytes cannot be swapped by path after verification.
			z, e := zip.NewReader(file, info.Size())
			if e != nil {
				return e
			}
			result.State = "extracting"
			emit()
			if e := extractInstallArchive(ctx, z, stage, r, &files); e != nil {
				return e
			}
			extracted = true
			return nil
		}
		report := VerifyPackage(ctx, r, r.Tag, opts.GitHub, v)
		if !extracted || (report.State != "completed" && report.State != "structure_incomplete") {
			return result, fmt.Errorf("resource_verification_failed:%s:%s:%s", r.ID, report.State, strings.Join(report.Problems, ","))
		}
		// The release asset digest, the catalog digest and the executed archive must be the same value.
		if !strings.EqualFold(report.ExpectedSHA256, expectedDigest) || !strings.EqualFold(report.CalculatedSHA256, expectedDigest) {
			return result, fmt.Errorf("catalog_digest_mismatch:%s", r.ID)
		}
		result.Version, result.SHA256 = report.Version, report.CalculatedSHA256
		prepared = append(prepared, r.ID)
	}
	result.Resource = ""
	result.State = "bootstrap"
	emit()
	byID := map[string]Resource{}
	for _, resource := range c.Resources {
		byID[resource.ID] = resource
	}
	bootReady := bootReadyFor(byID, selected, nil)
	bootstrapFiles, e := writeBootstrap(stage, templates, bootReady, partialBootstrapNotice, "recurso ou dependencia ausente nesta instalacao parcial")
	if e != nil {
		return result, e
	}
	files = append(files, bootstrapFiles...)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	result.State = "publishing"
	emit()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if e := publishInstallStage(stage, absolute); e != nil {
		return result, e
	}
	stage = ""
	result.Installed = prepared
	result.Files = files
	result.State = "partial_installed"
	result.Detail = "partial_installation_not_ready"
	emit()
	return result, nil
}

func permittedPartial(r Resource, p PackageReport) bool {
	expected, ok := catalogDigest(r.SHA256)
	if !ok || !strings.EqualFold(expected, p.CalculatedSHA256) || !strings.EqualFold(expected, p.ExpectedSHA256) || p.Version != r.Version || p.Profile != "packaged_profile_only" {
		return false
	}
	conditional := map[string]bool{}
	for _, f := range p.FileFindings {
		if f.Classification == "conditional" && f.Condition != "" {
			conditional[f.Path] = true
		}
	}
	for _, problem := range p.Problems {
		switch {
		case problem == "legacy_tag", problem == "runtime_dependency_coverage_inconclusive":
		case strings.HasPrefix(problem, "external_reference_unverified:"), strings.HasPrefix(problem, "external_runtime_reference_unverified:"), strings.HasPrefix(problem, "runtime_dynamic_reference_unverified:"), strings.HasPrefix(problem, "web_dynamic_reference_unverified:"):
		case strings.HasPrefix(problem, "reference_glob_unmatched:") && conditional[strings.TrimPrefix(problem, "reference_glob_unmatched:")]:
		case strings.HasPrefix(problem, "runtime_reference_absent_unverified:") && conditional[strings.TrimPrefix(problem, "runtime_reference_absent_unverified:")]:
		default:
			return false
		}
	}
	return true
}

func extractInstallArchive(ctx context.Context, z *zip.Reader, stage string, r Resource, installedFiles *[]string) error {
	base := filepath.Join(stage, "resources", r.Group, r.ID)
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	for _, f := range z.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := strings.TrimPrefix(f.Name, r.ID+"/")
		if name == f.Name || name == "" || !fs.ValidPath(strings.TrimSuffix(name, "/")) || strings.ContainsAny(name, `\:`) {
			return fmt.Errorf("unsafe_archive_path")
		}
		lower := strings.ToLower("/" + name)
		if strings.Contains(lower, "/node_modules/") || strings.Contains(lower, "/web/src/") || strings.Contains(lower, "/tests/") || strings.Contains(lower, "/.git/") {
			return fmt.Errorf("development_file_in_archive")
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return fmt.Errorf("unsafe_archive_type")
		}
		out := filepath.Join(base, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
			return err
		}
		in, e := f.Open()
		if e != nil {
			return e
		}
		file, e := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			in.Close()
			return e
		}
		_, e = io.Copy(file, contextReader{ctx, in})
		ce := file.Close()
		in.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		*installedFiles = append(*installedFiles, filepath.ToSlash(filepath.Join("resources", r.Group, r.ID, filepath.FromSlash(name))))
	}
	manifest := filepath.Join(base, "fxmanifest.lua")
	if st, e := os.Stat(manifest); e != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("manifest_missing_after_extraction")
	}
	return nil
}
