package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type changelogVersions struct {
	Recommended         string `json:"recommended"`
	RecommendedDownload string `json:"recommended_download"`
}

func artifactOverlap(target, path string) bool {
	a, err := filepath.Abs(target)
	if err != nil {
		return true
	}
	b, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	check := func(root, other string) bool {
		rel, err := filepath.Rel(root, other)
		if err != nil {
			return true
		}
		return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
	}
	return check(a, b) || check(b, a)
}

func artifactOS(opts FullOptions) string {
	if opts.ArtifactOS != "" {
		return opts.ArtifactOS
	}
	return runtime.GOOS
}

func resolveArtifacts(ctx context.Context, opts FullOptions) (ArtifactPlan, error) {
	plan := ArtifactPlan{Mode: opts.ArtifactsMode, Path: opts.ArtifactsPath}
	switch opts.ArtifactsMode {
	case "none":
		plan.Status = "not_provided"
		return plan, nil
	case "existing":
		abs, err := validateExistingArtifacts(opts.ArtifactsPath, artifactOS(opts))
		if err != nil {
			plan.Status, plan.Reason = "blocked", err.Error()
			return plan, nil
		}
		if opts.Target != "" && artifactOverlap(opts.Target, abs) {
			plan.Status, plan.Reason = "blocked", "artifacts_overlap_server_data"
			return plan, nil
		}
		plan.Path, plan.Status = abs, "existing"
		return plan, nil
	case "download":
		abs, err := validateArtifactDestination(opts.ArtifactsPath)
		if err != nil {
			plan.Status, plan.Reason = "blocked", err.Error()
			return plan, nil
		}
		if opts.Target != "" && artifactOverlap(opts.Target, abs) {
			plan.Status, plan.Reason = "blocked", "artifacts_overlap_server_data"
			return plan, nil
		}
		info, err := fetchRecommendedArtifact(ctx, opts)
		if err != nil {
			plan.Status, plan.Reason = "blocked", err.Error()
			return plan, nil
		}
		plan.Path, plan.Status, plan.Version, plan.URL, plan.Archive = abs, "ready", info.Build, info.URL, info.Archive
		return plan, nil
	case "auto":
		path := opts.ArtifactsPath
		if strings.TrimSpace(path) == "" {
			absTarget, err := filepath.Abs(opts.Target)
			if err != nil || absTarget == "" {
				plan.Status, plan.Reason = "blocked", "artifacts_path_invalid"
				return plan, nil
			}
			path = filepath.Join(filepath.Dir(absTarget), "mz-artifacts")
		}
		abs, err := validateArtifactDestination(path)
		if err != nil {
			plan.Status, plan.Reason = "blocked", err.Error()
			return plan, nil
		}
		if opts.Target != "" && artifactOverlap(opts.Target, abs) {
			plan.Status, plan.Reason = "blocked", "artifacts_overlap_server_data"
			return plan, nil
		}
		info, err := fetchRecommendedArtifact(ctx, opts)
		if err != nil {
			plan.Status, plan.Reason = "blocked", err.Error()
			return plan, nil
		}
		plan.Path, plan.Status, plan.Version, plan.URL, plan.Archive = abs, "ready", info.Build, info.URL, info.Archive
		return plan, nil
	default:
		plan.Status, plan.Reason = "blocked", "invalid_artifacts_mode"
		return plan, nil
	}
}

type recommendedArtifact struct {
	Build   string
	URL     string
	Archive string
}

func fetchRecommendedArtifact(ctx context.Context, opts FullOptions) (recommendedArtifact, error) {
	goos := artifactOS(opts)
	if goos != "windows" && goos != "linux" {
		return recommendedArtifact{}, fmt.Errorf("artifact_unsupported_host")
	}
	platform := "win32"
	if goos == "linux" {
		platform = "linux"
	}
	base := strings.TrimRight(opts.Sources.ChangelogBase, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/changelog/versions/"+platform+"/server", nil)
	if err != nil {
		return recommendedArtifact{}, fmt.Errorf("artifact_changelog_invalid")
	}
	req.Header.Set("User-Agent", "mz-starter-installer/2")
	client := opts.ArtifactClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return recommendedArtifact{}, fmt.Errorf("artifact_changelog_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return recommendedArtifact{}, fmt.Errorf("artifact_changelog_http_error")
	}
	var v changelogVersions
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&v); err != nil || v.Recommended == "" || v.RecommendedDownload == "" {
		return recommendedArtifact{}, fmt.Errorf("artifact_changelog_invalid")
	}
	if err = validateArtifactURL(v.RecommendedDownload, v.Recommended, opts.Sources.RuntimeBase, goos); err != nil {
		return recommendedArtifact{}, err
	}
	archive := "zip"
	if goos == "linux" {
		archive = "tar.xz"
	}
	return recommendedArtifact{Build: v.Recommended, URL: v.RecommendedDownload, Archive: archive}, nil
}

func validateArtifactURL(raw, build, runtimeBase, goos string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return fmt.Errorf("artifact_url_invalid")
	}
	suffix := "/server.zip"
	prefix := "/artifacts/fivem/build_server_windows/master/"
	if goos == "linux" {
		suffix = "/fx.tar.xz"
		prefix = "/artifacts/fivem/build_proot_linux/master/"
	} else if goos != "windows" {
		return fmt.Errorf("artifact_unsupported_host")
	}
	defaultBase := "https://runtime.fivem.net"
	base := strings.TrimRight(runtimeBase, "/")
	if base != defaultBase {
		if !strings.HasPrefix(raw, base+"/") || !strings.HasSuffix(u.Path, suffix) {
			return fmt.Errorf("artifact_url_invalid")
		}
		return nil
	}
	if u.Scheme != "https" || u.Host != "runtime.fivem.net" || u.Port() != "" {
		return fmt.Errorf("artifact_url_invalid")
	}
	if !strings.HasPrefix(u.Path, prefix) {
		return fmt.Errorf("artifact_url_invalid")
	}
	rest := strings.TrimPrefix(u.Path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || "/"+parts[1] != suffix || !strings.HasPrefix(parts[0], build+"-") {
		return fmt.Errorf("artifact_url_invalid")
	}
	return nil
}

func validateExistingArtifacts(path, goos string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("artifacts_path_required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("artifacts_path_invalid")
	}
	st, err := os.Lstat(abs)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(abs) {
		return "", fmt.Errorf("artifacts_path_invalid")
	}
	exe := "run.sh"
	if goos == "windows" {
		exe = "FXServer.exe"
	}
	if info, err := os.Stat(filepath.Join(abs, exe)); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("artifacts_executable_missing")
	}
	return abs, nil
}

func validateArtifactDestination(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("artifacts_path_required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("artifacts_path_invalid")
	}
	parent := filepath.Dir(abs)
	if p, err := os.Lstat(parent); err != nil || !p.IsDir() {
		return "", fmt.Errorf("artifacts_parent_missing")
	}
	if st, err := os.Lstat(abs); err == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("artifacts_destination_not_empty")
		}
		entries, err := os.ReadDir(abs)
		if err != nil || len(entries) != 0 {
			return "", fmt.Errorf("artifacts_destination_not_empty")
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("artifacts_destination_unavailable")
	}
	return abs, nil
}

func installArtifacts(ctx context.Context, opts FullOptions, target string, result *FullProgress, emit func()) (ArtifactPlan, error) {
	resolved, err := resolveArtifacts(ctx, opts)
	if err != nil {
		return resolved, err
	}
	if resolved.Status == "none" || resolved.Status == "not_provided" {
		return resolved, nil
	}
	if resolved.Status == "blocked" {
		return resolved, fmt.Errorf("%s", resolved.Reason)
	}
	if resolved.Mode == "existing" {
		return resolved, nil
	}
	parent := filepath.Dir(resolved.Path)
	work, err := makePrivateTemp(parent)
	if err != nil {
		return resolved, fmt.Errorf("staging_unavailable")
	}
	defer os.RemoveAll(work)
	archiveName := "server.zip"
	if resolved.Archive == "tar.xz" {
		archiveName = "fx.tar.xz"
	}
	archive := filepath.Join(work, archiveName)
	result.State = "artifacts"
	emit()
	_, err = downloadVerified(ctx, downloadSpec{URL: resolved.URL, MaxSize: 2 << 30, Client: opts.ArtifactClient, Progress: func(received, size int64) {
		result.Received, result.Size = received, size
		emit()
	}}, archive)
	if err != nil {
		return resolved, err
	}
	digest, _, err := fileSHA256(archive)
	if err != nil {
		return resolved, err
	}
	extract := filepath.Join(work, "artifact")
	if resolved.Archive == "tar.xz" {
		err = extractArtifactTarXZ(ctx, archive, extract)
	} else {
		err = extractArtifactArchive(ctx, archive, extract)
	}
	if err != nil {
		return resolved, err
	}
	if err = publishInstallStage(extract, resolved.Path); err != nil {
		return resolved, err
	}
	resolved.Status = "installed"
	resolved.Digest = digest
	return resolved, nil
}

func extractArtifactArchive(ctx context.Context, archive, dest string) error {
	z, err := openZip(archive)
	if err != nil {
		return fmt.Errorf("zip_corrupt")
	}
	defer z.Close()
	if err = os.MkdirAll(dest, 0700); err != nil {
		return err
	}
	var expanded uint64
	for _, f := range z.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !safeArchiveName(f.Name) {
			return fmt.Errorf("unsafe_archive_path")
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return fmt.Errorf("unsafe_archive_type")
		}
		size := f.UncompressedSize64
		if size > 512<<20 || size > (2<<30)-expanded {
			return fmt.Errorf("expanded_size_limit")
		}
		out := filepath.Join(dest, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(out, 0700); err != nil {
				return err
			}
			continue
		}
		expanded += size
		if err = os.MkdirAll(filepath.Dir(out), 0700); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(file, contextReader{ctx, in})
		closeErr := file.Close()
		_ = in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if info, err := os.Stat(filepath.Join(dest, "FXServer.exe")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("artifacts_executable_missing")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
