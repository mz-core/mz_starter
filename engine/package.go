package engine

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type PackageReport struct {
	Resource         string        `json:"resource"`
	Repository       string        `json:"repository,omitempty"`
	Version          string        `json:"version,omitempty"`
	Tag              string        `json:"tag,omitempty"`
	Asset            string        `json:"asset,omitempty"`
	Size             int64         `json:"size,omitempty"`
	Received         int64         `json:"received"`
	ExpectedSHA256   string        `json:"expected_sha256,omitempty"`
	CalculatedSHA256 string        `json:"calculated_sha256,omitempty"`
	ManifestVersion  string        `json:"manifest_version,omitempty"`
	State            string        `json:"state"`
	Download         string        `json:"download"`
	Integrity        string        `json:"integrity"`
	Provenance       string        `json:"provenance"`
	Structure        string        `json:"structure"`
	References       string        `json:"references"`
	Profile          string        `json:"profile"`
	Installation     string        `json:"installation"`
	FileFindings     []FileFinding `json:"file_findings,omitempty"`
	Problems         []string      `json:"problems"`
}
type VerifyOptions struct {
	MaxDownload    int64
	MaxExpanded    uint64
	MaxEntries     int
	TempRoot       string
	ForbiddenRoots []string
	Client         *http.Client
	APIBase        string
	Progress       func(PackageReport)
}

func (o VerifyOptions) defaults() VerifyOptions {
	if o.MaxDownload <= 0 {
		o.MaxDownload = 512 << 20
	}
	if o.MaxExpanded == 0 {
		o.MaxExpanded = 1 << 30
	}
	if o.MaxEntries <= 0 {
		o.MaxEntries = 10000
	}
	return o
}
func (r *PackageReport) issue(s string) { r.Problems = append(r.Problems, s) }

// VerifyPackage downloads only an asset selected from the authorized repository's release metadata.
func VerifyPackage(ctx context.Context, r Resource, tag string, g GitHub, opts VerifyOptions) (out PackageReport) {
	opts = opts.defaults()
	out = PackageReport{Resource: r.ID, Repository: r.Repository, State: "resolving", Download: "download_not_started", Integrity: "integrity_unverified", Provenance: "provenance_unverified", Structure: "structure_unverified", References: "references_unverified", Profile: "profile_inconclusive", Installation: "installation_unavailable", Problems: []string{}}
	emit := func() {
		if opts.Progress != nil {
			snapshot := out
			snapshot.Problems = append([]string{}, out.Problems...)
			opts.Progress(snapshot)
		}
	}
	emit()
	fail := func(code string) PackageReport {
		if out.Download == "downloading" {
			if code == "cancelled" {
				out.Download = "cancelled"
			} else {
				out.Download = "download_failed"
			}
		}
		out.State = code
		out.issue(code)
		emit()
		return out
	}
	if r.Source == "protected" {
		return fail("protected_distribution")
	}
	if r.Source != "github_release" && r.Source != "private_release_requires_authorization" {
		return fail("distribution_unavailable")
	}
	if r.Repository == "" {
		return fail("repository_unavailable")
	}
	q := g.Releases(ctx, r.ID, r.Repository)
	if ctx.Err() == context.DeadlineExceeded {
		return fail("timeout")
	}
	if ctx.Err() == context.Canceled {
		return fail("cancelled")
	}
	if q.Status != "ok" {
		if q.Status == "not_found_or_no_access" && r.Visibility == "private" && g.Token == "" {
			return fail("authorization_required_or_not_found")
		}
		return fail(q.Status)
	}
	var rel *Release
	for i := range q.Releases {
		x := &q.Releases[i]
		if tag != "" && x.Tag != tag {
			continue
		}
		if tag == "" && (x.Prerelease && r.Channel != "prerelease") {
			continue
		}
		if rel != nil && tag != "" {
			return fail("release_ambiguous")
		}
		rel = x
		if tag == "" {
			break
		}
	}
	if rel == nil {
		return fail("release_missing")
	}
	out.Version = rel.Version
	out.Tag = rel.Tag
	if r.Version != "" && r.Version != rel.Version {
		out.issue("catalog_version_differs")
	}
	if r.Tag != "" && r.Tag != rel.Tag {
		out.issue("catalog_tag_differs")
	}
	if rel.LegacyTag {
		out.issue("legacy_tag")
	}
	if rel.Status == "asset_missing" || rel.Status == "asset_ambiguous" {
		return fail(rel.Status)
	}
	var asset *Asset
	for i := range rel.Assets {
		a := &rel.Assets[i]
		if a.Name == r.ID+"-v"+rel.Version+".zip" {
			if asset != nil {
				return fail("asset_ambiguous")
			}
			asset = a
		}
	}
	if asset == nil {
		return fail("asset_missing")
	}
	if asset.State != "uploaded" {
		return fail("asset_unavailable")
	}
	out.Asset = asset.Name
	out.Size = asset.Size
	out.ExpectedSHA256 = strings.ToLower(strings.TrimPrefix(asset.Digest, "sha256:"))
	if rel.Status != "integrity_unverified" {
		return fail(rel.Status)
	}
	if asset.ID <= 0 || asset.Size <= 0 {
		return fail("invalid_asset_metadata")
	}
	if asset.Size > opts.MaxDownload {
		return fail("download_too_large")
	}
	if !strings.HasPrefix(asset.Digest, "sha256:") || len(out.ExpectedSHA256) != 64 || !hashPattern.MatchString(out.ExpectedSHA256) {
		return fail("integrity_invalid")
	}
	repo, err := url.Parse(r.Repository)
	if err != nil || repo.Scheme != "https" || repo.Host != "github.com" || repo.User != nil {
		return fail("invalid_repository")
	}
	parts := strings.Split(strings.Trim(repo.Path, "/"), "/")
	if len(parts) != 2 || parts[1] != r.ID || !idPattern.MatchString(parts[0]) {
		return fail("invalid_repository")
	}
	base := opts.APIBase
	if base == "" {
		base = "https://api.github.com"
	}
	if g.Token != "" && base != "https://api.github.com" {
		return fail("invalid_api_origin")
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil || req.URL.Port() != "" || !(req.URL.Hostname() == "release-assets.githubusercontent.com" || req.URL.Hostname() == "objects.githubusercontent.com") {
				return http.ErrUseLastResponse
			}
			req.Header.Del("Authorization")
			return nil
		}}
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", strings.TrimRight(base, "/"), parts[0], parts[1], asset.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fail("invalid_request")
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "mz-starter-verifier/1")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	out.State = "downloading"
	out.Download = "downloading"
	emit()
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fail("timeout")
		}
		var n net.Error
		if errors.As(err, &n) && n.Timeout() {
			return fail("timeout")
		}
		if ctx.Err() != nil {
			return fail("cancelled")
		}
		return fail("network_error")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fail("download_http_error")
	}
	if response.ContentLength > opts.MaxDownload {
		return fail("download_too_large")
	}
	if response.ContentLength >= 0 && response.ContentLength != asset.Size {
		return fail("download_size_mismatch")
	}
	tempRoot := opts.TempRoot
	if tempRoot == "" {
		tempRoot, err = defaultTempRoot()
		if err != nil {
			return fail("temporary_security_unavailable")
		}
	}
	tempRoot, err = filepath.Abs(tempRoot)
	if err != nil {
		return fail("temporary_storage_error")
	}
	if p, e := filepath.EvalSymlinks(tempRoot); e == nil {
		tempRoot = p
	} else {
		return fail("temporary_storage_error")
	}
	for _, root := range opts.ForbiddenRoots {
		root, e := filepath.Abs(root)
		if e != nil {
			return fail("temporary_storage_error")
		}
		if p, e := filepath.EvalSymlinks(root); e == nil {
			root = p
		}
		if !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(tempRoot)) {
			continue
		}
		rel, e := filepath.Rel(root, tempRoot)
		if e != nil {
			return fail("temporary_storage_error")
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return fail("temporary_path_conflict")
		}
	}
	dir, err := makePrivateTemp(tempRoot)
	if err != nil {
		return fail("temporary_security_unavailable")
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			out.State = "cleanup_failed"
			out.issue("cleanup_failed")
			emit()
		}
	}()
	f, err := os.OpenFile(filepath.Join(dir, "asset.zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fail("temporary_storage_error")
	}
	hash := sha256.New()
	buf := make([]byte, 64<<10)
	var readErr error
	for {
		if ctx.Err() != nil {
			readErr = ctx.Err()
			break
		}
		n, e := response.Body.Read(buf)
		if n > 0 {
			out.Received += int64(n)
			if out.Received > opts.MaxDownload || out.Received > asset.Size {
				readErr = fmt.Errorf("download_size_mismatch")
				break
			}
			if _, w := f.Write(buf[:n]); w != nil {
				readErr = fmt.Errorf("temporary_storage_error")
				break
			}
			_, _ = hash.Write(buf[:n])
			emit()
		}
		if e != nil {
			if e != io.EOF {
				readErr = e
			}
			break
		}
	}
	closeErr := f.Close()
	if readErr != nil || closeErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fail("timeout")
		}
		var n net.Error
		if errors.As(readErr, &n) && n.Timeout() {
			return fail("timeout")
		}
		if ctx.Err() != nil {
			return fail("cancelled")
		}
		if readErr != nil && readErr.Error() == "temporary_storage_error" {
			return fail("temporary_storage_error")
		}
		return fail("download_interrupted")
	}
	if out.Received != asset.Size {
		return fail("download_interrupted")
	}
	out.Download = "download_completed"
	out.CalculatedSHA256 = hex.EncodeToString(hash.Sum(nil))
	if out.CalculatedSHA256 != out.ExpectedSHA256 {
		out.Integrity = "sha256_mismatch"
		return fail("sha256_mismatch")
	}
	out.Integrity = "integrity_confirmed"
	out.State = "inspecting"
	emit()
	version, problems, incomplete := inspectZIPDetailed(ctx, filepath.Join(dir, "asset.zip"), r, rel.Version, opts)
	out.ManifestVersion = version
	out.Problems = append(out.Problems, problems...)
	out.Problems = append(out.Problems, incomplete...)
	out.FileFindings = inspectFileRules(filepath.Join(dir, "asset.zip"), r, rel.Version, out.CalculatedSHA256)
	for _, finding := range out.FileFindings {
		if finding.Classification == "required" && !finding.Present {
			out.Problems = append(out.Problems, "required_file_missing:"+finding.Path)
			incomplete = append(incomplete, "required_file_missing:"+finding.Path)
		}
	}
	if len(out.FileFindings) > 0 {
		out.Profile = "packaged_profile_only"
	}
	out.References = "references_complete"
	if len(incomplete) > 0 || len(problems) > 0 {
		out.References = "references_inconclusive"
	}
	if ctx.Err() != nil {
		return fail("cancelled")
	}
	if len(problems) > 0 {
		out.Structure = "structure_invalid"
		out.State = "structure_invalid"
	} else if len(incomplete) > 0 {
		out.Structure = "structure_incomplete"
		out.State = "structure_incomplete"
	} else {
		out.Structure = "structure_valid"
		out.State = "completed"
	}
	emit()
	return out
}

var reservedWindows = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\..*)?$`)
var loadResourceCall = regexp.MustCompile(`(?s)LoadResourceFile\s*\(\s*(GetCurrentResourceName\s*\(\s*\)|['"][^'"]+['"])\s*,\s*['"]([^'"]+)['"]\s*(?:,|\))`)
var anyLoadResourceCall = regexp.MustCompile(`\bLoadResourceFile\s*\(`)
var otherDynamicLuaCall = regexp.MustCompile(`\b(?:io\s*\.\s*open|dofile|loadfile|require)\s*\(`)

// shortcut: detects common dynamic web calls only; use a bounded JS parser for broader coverage.
var dynamicWebAssetCall = regexp.MustCompile("(?s)\\b(?:fetch|assetHref|import|require)\\s*\\(\\s*(?:[A-Za-z_$]|`[^`]*\\$\\{)|\\bXMLHttpRequest\\b")

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func InspectZIP(ctx context.Context, filename string, r Resource, version string, opts VerifyOptions) (string, []string) {
	versionFound, invalid, incomplete := inspectZIPDetailed(ctx, filename, r, version, opts)
	return versionFound, append(invalid, incomplete...)
}

func inspectZIPDetailed(ctx context.Context, filename string, r Resource, version string, opts VerifyOptions) (string, []string, []string) {
	opts = opts.defaults()
	z, e := zip.OpenReader(filename)
	if e != nil {
		return "", []string{"zip_corrupt"}, nil
	}
	defer z.Close()
	if len(z.File) > opts.MaxEntries {
		return "", []string{"too_many_entries"}, nil
	}
	seen := map[string]bool{}
	files := map[string]bool{}
	var total uint64
	manifest := ""
	manifestSource := ""
	var runtimeRefs []string
	var runtimeUnknown []string
	runtimeCodePresent := false
	problems := []string{}
	add := func(s string) { problems = append(problems, s) }
	for _, f := range z.File {
		if ctx.Err() != nil {
			return manifest, append(problems, "cancelled"), nil
		}
		n := f.Name
		trimmed := strings.TrimSuffix(n, "/")
		parts := strings.Split(trimmed, "/")
		bad := trimmed == "" || strings.HasPrefix(n, "/") || strings.ContainsAny(n, `\:<>"|?*`) || parts[0] != r.ID
		for _, p := range parts {
			for _, c := range p {
				if c < 32 {
					bad = true
				}
			}
			if p == "" || p == "." || p == ".." || strings.HasSuffix(p, ".") || strings.HasSuffix(p, " ") || reservedWindows.MatchString(p) {
				bad = true
			}
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			add("duplicate_or_case_collision")
		} else {
			seen[key] = true
		}
		if bad {
			add("unsafe_path_or_root")
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			add("unsafe_file_type")
			continue
		}
		for _, part := range parts {
			s := strings.ToLower(part)
			if s == "node_modules" || s == ".git" || s == ".env" || strings.HasPrefix(s, ".env.") && s != ".env.example" {
				add("forbidden_file")
			}
		}
		if f.FileInfo().IsDir() {
			continue
		}
		lowerName := strings.ToLower(n)
		if n != r.ID+"/fxmanifest.lua" && (strings.HasSuffix(lowerName, ".lua") || strings.HasSuffix(lowerName, ".js")) {
			runtimeCodePresent = true
		}
		files[strings.Join(parts[1:], "/")] = true
		size := f.UncompressedSize64
		if size > 256<<20 || size > opts.MaxExpanded-total {
			add("expanded_size_limit")
			continue
		}
		total += size
		if size > 1<<20 && (f.CompressedSize64 == 0 || size/f.CompressedSize64 > 200) {
			add("compression_ratio_limit")
			continue
		}
		rd, err := f.Open()
		if err != nil {
			add("zip_corrupt")
			continue
		}
		if n == r.ID+"/fxmanifest.lua" {
			if size > 2<<20 {
				add("manifest_too_large")
			} else {
				b, e := io.ReadAll(io.LimitReader(contextReader{ctx, rd}, 2<<20+1))
				if e != nil {
					add("zip_corrupt")
				} else {
					manifestSource = string(b)
					manifest, _ = ParseManifest(b)
				}
			}
		} else if strings.HasSuffix(strings.ToLower(n), ".lua") && size <= 2<<20 {
			b, e := io.ReadAll(io.LimitReader(contextReader{ctx, rd}, 2<<20+1))
			if e != nil {
				add("zip_corrupt")
			} else {
				text := string(b)
				if otherDynamicLuaCall.MatchString(text) {
					runtimeUnknown = append(runtimeUnknown, "runtime_dynamic_reference_unverified:"+strings.TrimPrefix(n, r.ID+"/"))
				}
				matches := loadResourceCall.FindAllStringSubmatch(text, -1)
				if len(anyLoadResourceCall.FindAllString(text, -1)) > len(matches) {
					runtimeUnknown = append(runtimeUnknown, "runtime_dynamic_reference_unverified:"+strings.TrimPrefix(n, r.ID+"/"))
				}
				for _, m := range matches {
					target := strings.Trim(m[1], "'\"")
					if target != "GetCurrentResourceName()" && !strings.HasPrefix(target, "GetCurrentResourceName") && target != r.ID {
						runtimeUnknown = append(runtimeUnknown, "external_runtime_reference_unverified:"+target)
						continue
					}
					runtimeRefs = append(runtimeRefs, m[2])
				}
			}
		} else if strings.HasSuffix(strings.ToLower(n), ".js") && size <= 2<<20 {
			b, e := io.ReadAll(io.LimitReader(contextReader{ctx, rd}, 2<<20+1))
			if e != nil {
				add("zip_corrupt")
			} else if dynamicWebAssetCall.Match(b) {
				runtimeUnknown = append(runtimeUnknown, "web_dynamic_reference_unverified:"+strings.TrimPrefix(n, r.ID+"/"))
			}
		} else {
			if strings.HasSuffix(strings.ToLower(n), ".lua") {
				runtimeUnknown = append(runtimeUnknown, "runtime_script_too_large_to_inspect:"+strings.TrimPrefix(n, r.ID+"/"))
			}
			if strings.HasSuffix(strings.ToLower(n), ".js") {
				runtimeUnknown = append(runtimeUnknown, "web_script_too_large_to_inspect:"+strings.TrimPrefix(n, r.ID+"/"))
			}
			if _, e := io.Copy(io.Discard, contextReader{ctx, rd}); e != nil {
				add("zip_corrupt")
			}
		}
		_ = rd.Close()
	}
	if !files["fxmanifest.lua"] {
		add("manifest_missing")
	} else if manifest == "" {
		add("manifest_version_unknown")
	} else if manifest != version {
		add("manifest_version_mismatch")
	}
	var requiredMissing []string
	for _, f := range r.RequiredFiles {
		if !files[f] {
			requiredMissing = append(requiredMissing, "required_file_missing:"+f)
		}
	}
	missing, unknown := inspectReferences(manifestSource, files)
	if runtimeCodePresent {
		runtimeUnknown = append(runtimeUnknown, "runtime_dependency_coverage_inconclusive")
	}
	missing = append(missing, requiredMissing...)
	for _, ref := range runtimeRefs {
		matched, valid := matchResourcePattern(ref, ref)
		if !valid || !matched {
			runtimeUnknown = append(runtimeUnknown, "runtime_reference_unsupported:"+ref)
		} else if !files[ref] {
			// shortcut: execution conditions are unknown; upgrade to sound control-flow analysis before classifying as required.
			runtimeUnknown = append(runtimeUnknown, "runtime_reference_absent_unverified:"+ref)
		}
	}
	return manifest, problems, append(append(missing, unknown...), runtimeUnknown...)
}
