package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	localMaxFiles = 50000
	localMaxBytes = int64(3) << 30
	localMaxFile  = int64(1) << 30
)

type LocalFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type LocalPackage struct {
	Kind    string      `json:"kind"`
	ID      string      `json:"id"`
	Version string      `json:"version"`
	Root    string      `json:"root"`
	Digest  string      `json:"digest"`
	Total   int64       `json:"total"`
	Files   []LocalFile `json:"files"`
}

func localSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".github", "node_modules", "graphify-out", "tests", "test-results", "logs", "cache", "crashes", "tmp":
		return true
	}
	return false
}

func localSkipFile(name string) bool {
	lower := strings.ToLower(name)
	if lower == ".env" || lower == ".mz-starter.json" || strings.HasPrefix(lower, ".env.") || lower == ".ds_store" || lower == "thumbs.db" {
		return true
	}
	for _, suffix := range []string{".log", ".tmp", ".bak", ".pdb"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func localSkipPath(rel string) bool {
	lower := strings.ToLower(filepath.ToSlash(rel))
	return strings.HasPrefix(lower, "web/src/") || strings.Contains(lower, "/web/src/")
}

func localTreeDigest(files []LocalFile) string {
	sorted := append([]LocalFile{}, files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	h := sha256.New()
	for _, f := range sorted {
		fmt.Fprintf(h, "%s\x00%d\x00%s\n", f.Path, f.Size, f.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// resolveLocalPackage locates an operator-provided package for one resource.
// A folder or archive is never trusted automatically: authorization is required
// whenever the catalog has no matching digest.
func resolveLocalPackage(root string, r Resource, authorized bool) (LocalPackage, error) {
	if strings.TrimSpace(root) == "" {
		return LocalPackage{}, fmt.Errorf("local_root_missing")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return LocalPackage{}, err
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return LocalPackage{}, fmt.Errorf("local_root_unavailable")
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(abs) {
		return LocalPackage{}, fmt.Errorf("local_root_invalid")
	}
	for _, candidate := range localZipCandidates(abs, r) {
		if info, e := os.Lstat(candidate); e == nil {
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return LocalPackage{}, fmt.Errorf("local_package_invalid")
			}
			return inspectLocalZip(candidate, r, authorized)
		}
	}
	source := filepath.Join(abs, r.ID)
	if info, e := os.Lstat(source); e == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(source) {
			return LocalPackage{}, fmt.Errorf("local_package_invalid")
		}
		return scanLocalFolder(source, r, authorized)
	}
	return LocalPackage{}, fmt.Errorf("local_package_missing")
}

func localZipCandidates(root string, r Resource) []string {
	var out []string
	if r.Version != "" {
		out = append(out, filepath.Join(root, r.ID+"-v"+r.Version+".zip"))
		out = append(out, filepath.Join(root, r.ID+"-"+r.Version+".zip"))
	}
	out = append(out, filepath.Join(root, r.ID+".zip"))
	return out
}

func inspectLocalZip(filename string, r Resource, authorized bool) (LocalPackage, error) {
	digest, _, err := fileSHA256(filename)
	if err != nil {
		return LocalPackage{}, err
	}
	expected, hasExpected := catalogDigest(r.SHA256)
	if hasExpected {
		if !strings.EqualFold(digest, expected) {
			return LocalPackage{}, fmt.Errorf("local_digest_mismatch")
		}
	} else if !authorized {
		return LocalPackage{}, fmt.Errorf("local_authorization_required")
	}
	z, err := openZip(filename)
	if err != nil {
		return LocalPackage{}, fmt.Errorf("zip_corrupt")
	}
	defer z.Close()
	version, err := readZipManifestVersion(&z.Reader, r.ID)
	if err != nil {
		return LocalPackage{}, err
	}
	if r.Version != "" && version != r.Version {
		return LocalPackage{}, fmt.Errorf("local_version_mismatch")
	}
	for _, required := range r.RequiredFiles {
		found := false
		for _, f := range z.File {
			if f.Name == r.ID+"/"+strings.TrimPrefix(filepath.ToSlash(required), "/") {
				found = true
				break
			}
		}
		if !found {
			return LocalPackage{}, fmt.Errorf("required_file_missing:%s", required)
		}
	}
	info, err := os.Stat(filename)
	if err != nil {
		return LocalPackage{}, err
	}
	return LocalPackage{Kind: "zip", ID: r.ID, Version: version, Root: filename, Digest: digest, Total: info.Size()}, nil
}

func scanLocalFolder(source string, r Resource, authorized bool) (LocalPackage, error) {
	manifestPath := filepath.Join(source, "fxmanifest.lua")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return LocalPackage{}, fmt.Errorf("local_manifest_missing")
	}
	version, _ := ParseManifest(manifest)
	if r.Version != "" && version != r.Version {
		return LocalPackage{}, fmt.Errorf("local_version_mismatch")
	}
	if !authorized {
		return LocalPackage{}, fmt.Errorf("local_authorization_required")
	}
	webSrc := false
	webDist := false
	var files []LocalFile
	var total int64
	err = filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if localSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			if rel == "web/src" {
				webSrc = true
			}
			if rel == "web/dist" {
				webDist = true
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || isReparsePoint(path) {
			return fmt.Errorf("local_unsafe_entry")
		}
		if localSkipFile(d.Name()) || localSkipPath(rel) {
			return nil
		}
		if info.Size() > localMaxFile {
			return fmt.Errorf("local_file_too_large:%s", rel)
		}
		if len(files) >= localMaxFiles || total+info.Size() > localMaxBytes {
			return fmt.Errorf("local_tree_too_large")
		}
		sum, _, err := fileSHA256(path)
		if err != nil {
			return err
		}
		files = append(files, LocalFile{Path: rel, Size: info.Size(), SHA256: sum})
		total += info.Size()
		return nil
	})
	if err != nil {
		return LocalPackage{}, err
	}
	if webSrc && !webDist {
		return LocalPackage{}, fmt.Errorf("local_web_dist_missing")
	}
	for _, required := range r.RequiredFiles {
		found := false
		for _, f := range files {
			if f.Path == strings.TrimPrefix(filepath.ToSlash(required), "/") {
				found = true
				break
			}
		}
		if !found {
			return LocalPackage{}, fmt.Errorf("required_file_missing:%s", required)
		}
	}
	return LocalPackage{Kind: "folder", ID: r.ID, Version: version, Root: source, Digest: localTreeDigest(files), Total: total, Files: files}, nil
}

// copyLocalPackage publishes a previously scanned package into staging and verifies
// every copied byte against the scanned digest.
func copyLocalPackage(ctx context.Context, pkg LocalPackage, dest string, protected bool) (int, error) {
	if pkg.Kind == "zip" {
		digest, _, err := fileSHA256(pkg.Root)
		if err != nil {
			return 0, err
		}
		if !strings.EqualFold(digest, pkg.Digest) {
			return 0, fmt.Errorf("local_source_changed")
		}
		z, err := openZip(pkg.Root)
		if err != nil {
			return 0, err
		}
		defer z.Close()
		var installed []string
		return extractZipTree(ctx, &z.Reader, dest, pkg.ID, "", nil, protected, &installed)
	}
	count := 0
	for _, f := range pkg.Files {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}
		src := filepath.Join(pkg.Root, filepath.FromSlash(f.Path))
		out := filepath.Join(dest, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
			return count, err
		}
		in, err := os.Open(src)
		if err != nil {
			return count, err
		}
		file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = in.Close()
			return count, err
		}
		h := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(file, h), contextReader{ctx, in})
		closeErr := file.Close()
		_ = in.Close()
		if copyErr != nil {
			return count, copyErr
		}
		if closeErr != nil {
			return count, closeErr
		}
		if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), f.SHA256) {
			return count, fmt.Errorf("local_source_changed:%s", f.Path)
		}
		count++
	}
	return count, nil
}

func fileSHA256(name string) (string, int64, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
