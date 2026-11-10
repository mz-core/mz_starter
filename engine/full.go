package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const fullBootstrapNotice = `# ---------------------------------------------------------------------------
# MZ STARTER - INSTALACAO COMPLETA DE ARQUIVOS
# Os componentes planejados foram instalados e verificados.
# Este diretorio AINDA NAO esta configurado para iniciar: preencha licenca,
# banco, endpoints e permissoes em cfg/ e forneca os artifacts do FXServer.
# Linhas de boot comentadas indicam componente nao presente.
# ---------------------------------------------------------------------------
`

type FullOptions struct {
	GitHub          GitHub
	Target          string
	LocalRoot       string
	LocalAuthorized bool
	ArtifactsMode   string
	ArtifactsPath   string
	MaxDownload     int64
	Progress        func(FullProgress)
	Sources         *Sources
	ArtifactClient  *http.Client
}

// Component is one resolvable piece of the server-data tree.
type Component struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Version      string   `json:"version,omitempty"`
	Tag          string   `json:"tag,omitempty"`
	Repository   string   `json:"repository,omitempty"`
	Asset        string   `json:"asset,omitempty"`
	Digest       string   `json:"digest,omitempty"`
	Size         int64    `json:"size,omitempty"`
	Destination  string   `json:"destination,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	SourceKind   string   `json:"source_kind"`
	SourceDetail string   `json:"source_detail,omitempty"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
	Protected    bool     `json:"protected,omitempty"`
	Optional     bool     `json:"optional,omitempty"`

	kind       string
	assetID    int64
	resource   Resource
	local      LocalPackage
	contract   ExternalContract
	archiveURL string
}

type ArtifactPlan struct {
	Mode    string `json:"mode"`
	Path    string `json:"path,omitempty"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Digest  string `json:"digest,omitempty"`
	URL     string `json:"url,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type FullPlan struct {
	Target                string       `json:"target"`
	Profile               string       `json:"profile"`
	LocalRoot             string       `json:"local_root,omitempty"`
	LocalAuthorized       bool         `json:"local_authorized"`
	Artifacts             ArtifactPlan `json:"artifacts"`
	Components            []Component  `json:"components"`
	Required              int          `json:"required"`
	Resolved              int          `json:"resolved"`
	Blockers              []string     `json:"blockers"`
	Warnings              []string     `json:"warnings"`
	ConfigurationRequired []string     `json:"configuration_required"`
	FilesReady            bool         `json:"files_ready"`
	ServerReady           bool         `json:"server_ready"`
}

type FullProgress struct {
	State              string       `json:"state"`
	Resource           string       `json:"resource,omitempty"`
	Index              int          `json:"index,omitempty"`
	Total              int          `json:"total,omitempty"`
	Received           int64        `json:"received,omitempty"`
	Size               int64        `json:"size,omitempty"`
	Installed          int          `json:"installed,omitempty"`
	InstalledResources []string     `json:"installed_resources,omitempty"`
	Missing            []string     `json:"missing,omitempty"`
	Artifacts          ArtifactPlan `json:"artifacts"`
	Plan               *FullPlan    `json:"plan,omitempty"`
	Destination        string       `json:"destination,omitempty"`
	Detail             string       `json:"detail,omitempty"`
	Files              int          `json:"files,omitempty"`
	FilesReady         bool         `json:"files_ready"`
	ServerReady        bool         `json:"server_ready"`
}

func (o FullOptions) defaults() FullOptions {
	if o.MaxDownload <= 0 {
		o.MaxDownload = 512 << 20
	}
	if o.ArtifactsMode == "" {
		o.ArtifactsMode = "none"
	}
	defaults := defaultSources()
	if o.Sources == nil {
		o.Sources = &defaults
		return o
	}
	merged := defaults
	if o.Sources.CodeloadBase != "" {
		merged.CodeloadBase = o.Sources.CodeloadBase
	}
	if o.Sources.ChangelogBase != "" {
		merged.ChangelogBase = o.Sources.ChangelogBase
	}
	if o.Sources.RuntimeBase != "" {
		merged.RuntimeBase = o.Sources.RuntimeBase
	}
	for id, contract := range o.Sources.External {
		merged.External[id] = contract
	}
	o.Sources = &merged
	return o
}

func componentDestination(r Resource) string {
	group := r.Group
	if group == "" {
		group = "[mz]"
	}
	return "resources/" + group + "/" + r.ID
}

func contractDestination(c ExternalContract) string {
	return "resources/" + c.DestinationGroup + "/" + c.ID
}

// ResolveFull computes the complete component plan without writing to any server.
func ResolveFull(ctx context.Context, c Catalog, opts FullOptions) (FullPlan, error) {
	opts = opts.defaults()
	plan := FullPlan{
		Target:                opts.Target,
		Profile:               c.Profile,
		LocalRoot:             opts.LocalRoot,
		LocalAuthorized:       opts.LocalAuthorized,
		Components:            []Component{},
		Blockers:              []string{},
		Warnings:              []string{},
		ConfigurationRequired: append([]string{}, BootstrapConfigurations...),
	}
	artifact, err := resolveArtifacts(ctx, opts)
	if err != nil {
		return plan, err
	}
	plan.Artifacts = artifact
	if artifact.Status == "blocked" {
		plan.Blockers = append(plan.Blockers, "artifacts:"+artifact.Reason)
	}
	if artifact.Mode == "none" {
		plan.Warnings = append(plan.Warnings, "system_artifacts_not_provided")
	}
	byID := map[string]Resource{}
	for _, r := range c.Resources {
		byID[r.ID] = r
	}
	seen := map[string]bool{}
	plan.Components = make([]Component, len(c.Resources))
	var wg sync.WaitGroup
	limit := make(chan struct{}, 6)
	for i, r := range c.Resources {
		wg.Add(1)
		go func(i int, r Resource) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			plan.Components[i] = resolveComponent(ctx, r, opts)
		}(i, r)
	}
	wg.Wait()
	for _, comp := range plan.Components {
		seen[comp.ID] = true
	}
	if !seen["baseevents"] {
		if contract, ok := externalContract(*opts.Sources, "baseevents"); ok {
			comp := Component{
				ID: "baseevents", Type: "external", Destination: contractDestination(contract),
				SourceKind: "pinned_archive", SourceDetail: "official cfx-server-data pinned snapshot",
				Status: "ready", Optional: true, kind: "pinned", contract: contract,
				Digest: contract.ArchiveDigest,
			}
			comp.archiveURL = archiveURL(*opts.Sources, contract)
			plan.Components = append(plan.Components, comp)
		}
	}
	plan.Required = len(c.Resources)
	components := map[string]*Component{}
	for i := range plan.Components {
		components[plan.Components[i].ID] = &plan.Components[i]
	}
	for i := range plan.Components {
		comp := &plan.Components[i]
		if comp.Type != "fivem_resource" {
			continue
		}
		for _, dep := range comp.Dependencies {
			if _, ok := byID[dep]; !ok {
				plan.Blockers = append(plan.Blockers, comp.ID+":unknown_dependency:"+dep)
				continue
			}
			if d := components[dep]; d == nil || d.Status == "blocked" {
				plan.Warnings = append(plan.Warnings, comp.ID+":dependency_blocked:"+dep)
			}
		}
	}
	for _, comp := range plan.Components {
		switch comp.Status {
		case "ready", "system":
			plan.Resolved++
		case "blocked":
			if !comp.Optional {
				plan.Blockers = append(plan.Blockers, comp.ID+":"+comp.Reason)
			}
		}
	}
	plan.Blockers = uniqueStrings(plan.Blockers)
	plan.Warnings = uniqueStrings(plan.Warnings)
	plan.FilesReady = len(plan.Blockers) == 0
	plan.ServerReady = plan.FilesReady && artifactReady(artifact.Status) && len(plan.ConfigurationRequired) == 0
	return plan, nil
}

func artifactReady(status string) bool {
	return status == "ready" || status == "existing" || status == "installed"
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func resolveComponent(ctx context.Context, r Resource, opts FullOptions) Component {
	base := Component{
		ID: r.ID, Type: r.Type, Version: r.Version, Tag: r.Tag, Repository: r.Repository,
		Destination: componentDestination(r), Dependencies: append([]string{}, r.Dependencies...),
		resource: r, Status: "blocked",
	}
	switch r.Type {
	case "artifact":
		base.Status = "system"
		base.SourceKind = "system_artifact"
		base.SourceDetail = "provided by the FXServer artifacts"
		base.Destination = "citizen/system_resources/" + r.ID
		return base
	case "cfx":
		contract, ok := externalContract(*opts.Sources, r.ID)
		if !ok {
			base.Reason = "cfx_contract_unknown"
			return base
		}
		return resolvePinned(r, contract, opts, base)
	case "external":
		contract, ok := externalContract(*opts.Sources, r.ID)
		if !ok {
			base.Reason = "external_contract_unknown"
			return base
		}
		if contract.Kind == ExternalRelease {
			return resolveExternalRelease(ctx, r, contract, opts, base)
		}
		return resolvePinned(r, contract, opts, base)
	case "fivem_resource":
		return resolveMZ(ctx, r, opts, base)
	default:
		base.Reason = "unsupported_resource_type"
		return base
	}
}

func resolveMZ(ctx context.Context, r Resource, opts FullOptions, comp Component) Component {
	if r.Source == "protected" && !opts.LocalAuthorized {
		comp.Status = "blocked"
		comp.Reason = "protected_local_authorization_required"
		comp.Protected = true
		return comp
	}
	if r.Visibility == "public" && r.Source == "github_release" && r.Tag != "" && r.Asset != "" {
		if expected, ok := catalogDigest(r.SHA256); ok {
			if direct, err := directReleaseComponent(r, expected); err == nil {
				return direct
			}
		}
	}
	localFirst := r.Source == "protected" || r.Visibility != "public" || r.Source != "github_release" || r.Tag == "" || r.Asset == "" || r.SHA256 == ""
	if localFirst {
		if local, err := resolveLocalPackage(opts.LocalRoot, r, opts.LocalAuthorized); err == nil {
			return applyLocal(comp, local, r)
		} else {
			comp.Reason = err.Error()
		}
		if r.Source == "protected" {
			comp.Status = "blocked"
			comp.Reason = "protected_local_package_required"
			comp.Protected = true
			return comp
		}
		if r.Visibility != "public" && opts.GitHub.Token == "" {
			comp.Status = "blocked"
			comp.Reason = "private_release_or_local_package_required"
			return comp
		}
	}
	if r.Source == "github_release" || r.Source == "private_release_requires_authorization" {
		if r.Visibility == "public" || opts.GitHub.Token != "" {
			if remote, err := resolveGitHubRelease(ctx, r, opts); err == nil {
				return remote
			} else if comp.Reason == "" {
				comp.Reason = "github:" + err.Error()
			}
		}
	}
	if local, err := resolveLocalPackage(opts.LocalRoot, r, opts.LocalAuthorized); err == nil {
		return applyLocal(comp, local, r)
	}
	comp.Status = "blocked"
	if comp.Reason == "" {
		if r.Source == "unavailable" || r.Repository == "" {
			comp.Reason = "repository_missing_or_unknown"
		} else {
			comp.Reason = "release_or_local_package_missing"
		}
	}
	return comp
}

func directReleaseComponent(r Resource, digest string) (Component, error) {
	u, err := url.Parse(r.Repository)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil {
		return Component{}, fmt.Errorf("invalid_repository")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[1] != r.ID || !idPattern.MatchString(parts[0]) {
		return Component{}, fmt.Errorf("invalid_repository")
	}
	comp := Component{
		ID: r.ID, Type: r.Type, Version: r.Version, Tag: r.Tag, Repository: r.Repository,
		Destination: componentDestination(r), Dependencies: append([]string{}, r.Dependencies...),
		Status: "ready", resource: r, kind: "direct_release", Digest: digest, Asset: r.Asset,
		SourceKind: "github_release", SourceDetail: "pinned GitHub release " + r.Tag,
	}
	comp.archiveURL = "https://github.com/" + parts[0] + "/" + parts[1] + "/releases/download/" + r.Tag + "/" + r.Asset
	return comp, nil
}

func resolveGitHubRelease(ctx context.Context, r Resource, opts FullOptions) (Component, error) {
	comp := Component{
		ID: r.ID, Type: r.Type, Version: r.Version, Tag: r.Tag, Repository: r.Repository,
		Destination: componentDestination(r), Dependencies: append([]string{}, r.Dependencies...),
		Status: "blocked", resource: r, kind: "github",
	}
	if r.Repository == "" {
		return comp, fmt.Errorf("repository_missing")
	}
	q := opts.GitHub.Releases(ctx, r.ID, r.Repository)
	if q.Status != "ok" {
		return comp, fmt.Errorf("%s", q.Status)
	}
	var matched []Release
	for _, rel := range q.Releases {
		if rel.Version != r.Version {
			continue
		}
		if r.Tag != "" && rel.Tag != r.Tag {
			continue
		}
		matched = append(matched, rel)
	}
	if len(matched) == 0 {
		return comp, fmt.Errorf("release_missing")
	}
	if len(matched) > 1 {
		return comp, fmt.Errorf("release_ambiguous")
	}
	rel := matched[0]
	if rel.Status != "integrity_unverified" {
		return comp, fmt.Errorf("%s", rel.Status)
	}
	var asset *Asset
	for i := range rel.Assets {
		a := &rel.Assets[i]
		if a.Name == r.ID+"-v"+rel.Version+".zip" {
			if asset != nil {
				return comp, fmt.Errorf("asset_ambiguous")
			}
			asset = a
		}
	}
	if asset == nil {
		return comp, fmt.Errorf("asset_missing")
	}
	if asset.State != "" && asset.State != "uploaded" {
		return comp, fmt.Errorf("asset_unavailable")
	}
	digest := strings.ToLower(strings.TrimPrefix(asset.Digest, "sha256:"))
	if len(digest) != 64 || !hashPattern.MatchString(digest) {
		return comp, fmt.Errorf("integrity_missing")
	}
	if expected, ok := catalogDigest(r.SHA256); ok && !strings.EqualFold(digest, expected) {
		return comp, fmt.Errorf("catalog_digest_changed")
	}
	if asset.ID <= 0 || asset.Size <= 0 {
		return comp, fmt.Errorf("invalid_asset_metadata")
	}
	comp.Status = "ready"
	comp.SourceKind = "github_release"
	comp.SourceDetail = "GitHub release " + rel.Tag
	comp.Tag = rel.Tag
	comp.Asset = asset.Name
	comp.Digest = digest
	comp.Size = asset.Size
	comp.assetID = asset.ID
	return comp, nil
}

func applyLocal(comp Component, local LocalPackage, r Resource) Component {
	comp.Status = "ready"
	comp.Reason = ""
	comp.kind = "local_" + local.Kind
	comp.local = local
	comp.Digest = local.Digest
	comp.Size = local.Total
	comp.SourceKind = "local_" + local.Kind
	comp.SourceDetail = "operator-authorized local package"
	comp.Protected = r.Source == "protected"
	if local.Kind == "folder" && r.SHA256 != "" {
		comp.SourceDetail = "operator-authorized local tree (release digest differs)"
	}
	return comp
}

func resolveExternalRelease(ctx context.Context, r Resource, contract ExternalContract, opts FullOptions, comp Component) Component {
	comp.contract = contract
	if local, err := resolveLocalPackage(opts.LocalRoot, r, opts.LocalAuthorized); err == nil {
		return applyLocal(comp, local, r)
	}
	if contract.URL != "" && contract.ArchiveDigest != "" {
		comp.Status = "ready"
		comp.kind = "pinned"
		comp.SourceKind = "external_release"
		comp.SourceDetail = "pinned GitHub release " + contract.Version
		comp.Version = contract.Version
		comp.Digest = contract.ArchiveDigest
		comp.archiveURL = contract.URL
		return comp
	}
	q := opts.GitHub.Releases(ctx, r.ID, contract.Repository)
	if q.Status != "ok" {
		comp.Reason = "external_release_" + q.Status
		return comp
	}
	for _, rel := range q.Releases {
		if rel.Prerelease {
			continue
		}
		for _, a := range rel.Assets {
			if a.Name != contract.Asset {
				continue
			}
			digest := strings.ToLower(strings.TrimPrefix(a.Digest, "sha256:"))
			if len(digest) != 64 || !hashPattern.MatchString(digest) || a.ID <= 0 || a.Size <= 0 || (a.State != "" && a.State != "uploaded") {
				continue
			}
			comp.Status = "ready"
			comp.kind = "external_release"
			comp.SourceKind = "external_release"
			comp.SourceDetail = "GitHub release " + rel.Tag
			comp.Version = rel.Version
			comp.Tag = rel.Tag
			comp.Asset = a.Name
			comp.Digest = digest
			comp.Size = a.Size
			comp.assetID = a.ID
			return comp
		}
	}
	comp.Reason = "external_release_asset_missing"
	return comp
}

func resolvePinned(r Resource, contract ExternalContract, opts FullOptions, comp Component) Component {
	comp.contract = contract
	if local, err := resolveLocalPackage(opts.LocalRoot, r, opts.LocalAuthorized); err == nil {
		if contract.Patch == "pma-voice" {
			if err := verifyPMAVoiceLocal(local); err != nil {
				comp.Reason = err.Error()
				return comp
			}
		}
		return applyLocal(comp, local, r)
	}
	comp.Status = "ready"
	comp.kind = "pinned"
	comp.SourceKind = "pinned_archive"
	comp.SourceDetail = "official pinned snapshot"
	comp.Version = contract.Version
	comp.Digest = contract.ArchiveDigest
	if contract.URL != "" {
		comp.archiveURL = contract.URL
	} else {
		comp.archiveURL = archiveURL(*opts.Sources, contract)
	}
	return comp
}

func verifyPMAVoiceLocal(local LocalPackage) error {
	if local.Kind == "folder" {
		b, err := os.ReadFile(filepath.Join(local.Root, "server", "module", "phone.lua"))
		if err != nil {
			return fmt.Errorf("pma_voice_file_missing")
		}
		return verifyPMAVoiceText(string(b))
	}
	z, err := openZip(local.Root)
	if err != nil {
		return err
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name != local.ID+"/server/module/phone.lua" {
			continue
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(io.LimitReader(in, 2<<20))
		_ = in.Close()
		if err != nil {
			return err
		}
		return verifyPMAVoiceText(string(b))
	}
	return fmt.Errorf("pma_voice_file_missing")
}

func allowedInconclusiveProblems(problems []string) bool {
	for _, problem := range problems {
		switch {
		case problem == "legacy_tag", problem == "runtime_dependency_coverage_inconclusive":
		case problem == "manifest_reference_limit":
		case strings.HasPrefix(problem, "external_reference_unverified:"),
			strings.HasPrefix(problem, "external_runtime_reference_unverified:"),
			strings.HasPrefix(problem, "runtime_dynamic_reference_unverified:"),
			strings.HasPrefix(problem, "web_dynamic_reference_unverified:"),
			strings.HasPrefix(problem, "web_script_too_large_to_inspect:"),
			strings.HasPrefix(problem, "runtime_script_too_large_to_inspect:"),
			strings.HasPrefix(problem, "reference_glob_unmatched:"),
			strings.HasPrefix(problem, "runtime_reference_absent_unverified:"):
		default:
			return false
		}
	}
	return true
}

func InstallFull(ctx context.Context, c Catalog, templates fs.FS, target string, opts FullOptions) (result FullProgress, err error) {
	opts = opts.defaults()
	opts.Target = target
	result.State = "preflight"
	result.Artifacts = ArtifactPlan{Mode: opts.ArtifactsMode}
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
	if _, err = PlanInstallation(c, templates, target, "", "new", "", nil); err != nil {
		return result, err
	}
	absolute, err := validateInstallDestination(target)
	if err != nil {
		return result, err
	}
	parent := filepath.Dir(absolute)
	result.State = "resolving"
	result.Destination = absolute
	emit()
	plan, err := ResolveFull(ctx, c, opts)
	if err != nil {
		return result, err
	}
	result.Plan = &plan
	result.Artifacts = plan.Artifacts
	result.Total = 0
	for _, comp := range plan.Components {
		if comp.Status == "ready" {
			result.Total++
		}
	}
	if !plan.FilesReady {
		result.Missing = blockedIDs(plan)
		emit()
		return result, fmt.Errorf("full_installation_blocked")
	}
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
	downloads := filepath.Join(stage, ".mz-starter-downloads")
	if err = os.MkdirAll(downloads, 0700); err != nil {
		return result, err
	}
	byID := map[string]Resource{}
	for _, r := range c.Resources {
		byID[r.ID] = r
	}
	installedSet := map[string]bool{}
	systemSet := map[string]bool{}
	archiveCache := map[string]string{}
	index := 0
	for i := range plan.Components {
		comp := &plan.Components[i]
		if comp.Status == "system" {
			if artifactReady(plan.Artifacts.Status) {
				systemSet[comp.ID] = true
			}
			continue
		}
		if comp.Status != "ready" {
			continue
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		index++
		result.State, result.Resource, result.Index = "downloading", comp.ID, index
		emit()
		var count int
		switch comp.kind {
		case "github", "direct_release":
			count, err = installReleaseComponent(ctx, comp, stage, downloads, opts, &result, emit)
		case "external_release":
			count, err = installExternalRelease(ctx, comp, stage, downloads, opts, &result, emit)
		case "pinned":
			count, err = installPinned(ctx, comp, stage, downloads, archiveCache, opts, &result, emit)
		case "local_zip", "local_folder":
			count, err = copyLocalPackage(ctx, comp.local, filepath.Join(stage, filepath.FromSlash(comp.Destination)), comp.Protected)
			if err == nil && comp.contract.Patch == "pma-voice" {
				err = verifyPMAVoiceFile(filepath.Join(stage, filepath.FromSlash(comp.Destination), "server", "module", "phone.lua"))
			}
		default:
			err = fmt.Errorf("component_source_unknown:%s", comp.ID)
		}
		if err != nil {
			return result, fmt.Errorf("component_failed:%s:%w", comp.ID, err)
		}
		if err = verifyInstalledResource(filepath.Join(stage, filepath.FromSlash(comp.Destination)), comp); err != nil {
			return result, fmt.Errorf("component_failed:%s:%w", comp.ID, err)
		}
		if err = writeManagedMarker(stage, comp); err != nil {
			return result, err
		}
		result.Files += count
		installedSet[comp.ID] = true
		result.Installed++
		result.InstalledResources = append(result.InstalledResources, comp.ID)
		result.State, result.Resource = "installed", comp.ID
		emit()
	}
	result.Resource = ""
	result.State = "bootstrap"
	emit()
	bootReady := bootReadyFor(byID, installedSet, systemSet)
	bootstrapFiles, err := writeBootstrap(stage, templates, bootReady, fullBootstrapNotice, "recurso ou dependencia ausente nesta instalacao")
	if err != nil {
		return result, err
	}
	result.Files += len(bootstrapFiles)
	if err = os.RemoveAll(downloads); err != nil {
		return result, fmt.Errorf("staging_cleanup_failed")
	}
	result.State = "publishing"
	emit()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err = publishInstallStage(stage, absolute); err != nil {
		return result, err
	}
	stage = ""
	for i := range plan.Components {
		if installedSet[plan.Components[i].ID] {
			plan.Components[i].Status = "installed"
		}
	}
	result.Plan = &plan
	result.FilesReady = true
	artifactResult, artifactErr := installArtifacts(ctx, opts, absolute, &result, emit)
	result.Artifacts = artifactResult
	if artifactErr != nil {
		result.State = "completed_files_only"
		result.Detail = "artifacts_failed:" + artifactErr.Error()
		result.ServerReady = false
		emit()
		return result, nil
	}
	result.ServerReady = plan.FilesReady && artifactReady(artifactResult.Status)
	if result.ServerReady && len(plan.ConfigurationRequired) > 0 {
		result.ServerReady = false
		result.Detail = "configuration_required"
	}
	if artifactResult.Status == "ready" || artifactResult.Status == "installed" || artifactResult.Status == "existing" {
		result.State = "completed"
		if result.Detail == "" {
			result.Detail = "configuration_required"
		}
	} else {
		result.State = "completed_files_only"
		result.Detail = "server_artifacts_not_provided"
	}
	emit()
	return result, nil
}

func blockedIDs(plan FullPlan) []string {
	var ids []string
	for _, comp := range plan.Components {
		if comp.Status == "blocked" && !comp.Optional {
			ids = append(ids, comp.ID)
		}
	}
	return ids
}

func installReleaseComponent(ctx context.Context, comp *Component, stage, downloads string, opts FullOptions, result *FullProgress, emit func()) (int, error) {
	archive := filepath.Join(downloads, comp.ID+"-"+comp.Digest[:12]+".zip")
	progress := func(received, size int64) {
		result.Received, result.Size = received, size
		emit()
	}
	var err error
	if comp.kind == "direct_release" {
		_, err = downloadVerified(ctx, downloadSpec{URL: comp.archiveURL, Digest: comp.Digest, MaxSize: opts.MaxDownload, Client: opts.ArtifactClient, Progress: progress}, archive)
	} else {
		apiBase := ""
		if opts.GitHub.BaseURL != "" {
			apiBase = opts.GitHub.BaseURL
		}
		_, err = downloadGitHubAsset(ctx, opts.GitHub, apiBase, comp.resource.Repository, Asset{ID: comp.assetID, Size: comp.Size, Digest: "sha256:" + comp.Digest}, archive, progress)
	}
	if err != nil {
		return 0, err
	}
	return extractReleaseArchive(ctx, comp, archive, stage)
}

func extractReleaseArchive(ctx context.Context, comp *Component, archive, stage string) (int, error) {
	version, invalid, incomplete := inspectZIPDetailed(ctx, archive, comp.resource, comp.Version, VerifyOptions{MaxDownload: 512 << 20})
	if version != comp.Version {
		return 0, fmt.Errorf("manifest_version_mismatch")
	}
	if len(invalid) > 0 {
		return 0, fmt.Errorf("structure_invalid:%s", strings.Join(invalid, ","))
	}
	if !allowedInconclusiveProblems(incomplete) {
		return 0, fmt.Errorf("structure_invalid:%s", strings.Join(incomplete, ","))
	}
	z, err := openZip(archive)
	if err != nil {
		return 0, fmt.Errorf("zip_corrupt")
	}
	defer z.Close()
	var installed []string
	dest := filepath.Join(stage, filepath.FromSlash(comp.Destination))
	return extractZipTree(ctx, &z.Reader, dest, comp.resource.ID, "", comp.resource.RequiredFiles, comp.Protected, &installed)
}

func installExternalRelease(ctx context.Context, comp *Component, stage, downloads string, opts FullOptions, result *FullProgress, emit func()) (int, error) {
	archive := filepath.Join(downloads, comp.ID+".zip")
	apiBase := ""
	if opts.GitHub.BaseURL != "" {
		apiBase = opts.GitHub.BaseURL
	}
	_, err := downloadGitHubAsset(ctx, opts.GitHub, apiBase, comp.contract.Repository, Asset{ID: comp.assetID, Size: comp.Size, Digest: "sha256:" + comp.Digest}, archive, func(received, size int64) {
		result.Received, result.Size = received, size
		emit()
	})
	if err != nil {
		return 0, err
	}
	z, err := openZip(archive)
	if err != nil {
		return 0, fmt.Errorf("zip_corrupt")
	}
	defer z.Close()
	if comp.Version != "" {
		version, err := readZipManifestVersion(&z.Reader, comp.ID)
		if err != nil {
			return 0, err
		}
		if version != "" && version != comp.Version {
			return 0, fmt.Errorf("manifest_version_mismatch")
		}
	}
	var installed []string
	return extractZipTree(ctx, &z.Reader, filepath.Join(stage, filepath.FromSlash(comp.Destination)), comp.ID, "", comp.contract.RequiredFiles, false, &installed)
}

func installPinned(ctx context.Context, comp *Component, stage, downloads string, cache map[string]string, opts FullOptions, result *FullProgress, emit func()) (int, error) {
	archive, ok := cache[comp.archiveURL]
	if !ok {
		archive = filepath.Join(downloads, comp.ID+"-"+comp.contract.ArchiveDigest[:12]+".zip")
		_, err := downloadVerified(ctx, downloadSpec{URL: comp.archiveURL, Digest: comp.contract.ArchiveDigest, MaxSize: 128 << 20, Client: opts.ArtifactClient, Progress: func(received, size int64) {
			result.Received, result.Size = received, size
			emit()
		}}, archive)
		if err != nil {
			return 0, err
		}
		cache[comp.archiveURL] = archive
	}
	z, err := openZip(archive)
	if err != nil {
		return 0, fmt.Errorf("zip_corrupt")
	}
	defer z.Close()
	dest := filepath.Join(stage, filepath.FromSlash(comp.Destination))
	var installed []string
	count, err := extractZipTree(ctx, &z.Reader, dest, comp.contract.ArchiveRoot, comp.contract.ArchiveSubdir, comp.contract.RequiredFiles, false, &installed)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("archive_root_mismatch")
	}
	if comp.contract.Patch == "pma-voice" {
		if err = applyPMAVoicePatch(filepath.Join(dest, "server", "module", "phone.lua")); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func verifyPMAVoiceFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("pma_voice_file_missing")
	}
	return verifyPMAVoiceText(string(b))
}

func verifyInstalledResource(dest string, comp *Component) error {
	if _, err := os.Stat(filepath.Join(dest, "fxmanifest.lua")); err != nil {
		return fmt.Errorf("manifest_missing_after_extraction")
	}
	if comp.Version == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(dest, "fxmanifest.lua"))
	if err != nil {
		return err
	}
	v, _ := ParseManifest(b)
	if v != "" && v != comp.Version {
		return fmt.Errorf("installed_version_mismatch")
	}
	return nil
}

func writeManagedMarker(stage string, comp *Component) error {
	if comp.Digest == "" {
		return nil
	}
	dest := filepath.Join(stage, filepath.FromSlash(comp.Destination))
	marker := filepath.Join(dest, ".mz-starter.json")
	if _, err := os.Lstat(marker); err == nil {
		return nil
	}
	record := map[string]any{
		"schema_version": 1,
		"id":             comp.ID,
		"version":        comp.Version,
		"sha256":         comp.Digest,
		"source_kind":    comp.SourceKind,
	}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return os.WriteFile(marker, b, 0600)
}
