package engine

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	fullMaxEntries  = 30000
	fullMaxExpanded = 2 << 30
	fullMaxEntry    = 1 << 30
)

func openZip(filename string) (*zip.ReadCloser, error) { return zip.OpenReader(filename) }

func safeArchiveName(name string) bool {
	name = strings.TrimSuffix(name, "/")
	if name == "" {
		return true
	}
	if strings.ContainsAny(name, `\:`) || strings.HasPrefix(name, "/") {
		return false
	}
	return fs.ValidPath(name)
}

// forbiddenArchivePath reports distribution files that must never leave staging.
func forbiddenArchivePath(name string) bool {
	lower := strings.ToLower("/" + strings.TrimPrefix(filepath.ToSlash(name), "/"))
	for _, part := range []string{"/.git/", "/.github/", "/node_modules/", "/graphify-out/", "/tests/", "/test-results/", "/.env", "/logs/", "/cache/", "/crashes/"} {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return strings.HasPrefix(lower, "/.env")
}

// extractZipTree extracts the selected archive subtree into dest. It never writes
// outside dest, never follows links and never overwrites an existing file.
func extractZipTree(ctx context.Context, z *zip.Reader, dest, prefix, subdir string, required []string, protected bool, installed *[]string) (int, error) {
	if len(z.File) > fullMaxEntries {
		return 0, fmt.Errorf("too_many_entries")
	}
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	if subdir != "" {
		prefix += strings.Trim(subdir, "/") + "/"
	}
	if err := os.MkdirAll(dest, 0700); err != nil {
		return 0, err
	}
	var expanded uint64
	files := map[string]bool{}
	count := 0
	for _, f := range z.File {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		name := strings.TrimPrefix(f.Name, prefix)
		if name == "" {
			continue
		}
		if forbiddenArchivePath(name) {
			continue
		}
		if !safeArchiveName(name) {
			if protected {
				return count, fmt.Errorf("unsafe_archive_path")
			}
			return count, fmt.Errorf("unsafe_archive_path:%s", name)
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return count, fmt.Errorf("unsafe_archive_type")
		}
		size := f.UncompressedSize64
		if size > fullMaxEntry || size > fullMaxExpanded-expanded {
			return count, fmt.Errorf("expanded_size_limit")
		}
		if size > 1<<20 && (f.CompressedSize64 == 0 || size/f.CompressedSize64 > 200) {
			return count, fmt.Errorf("compression_ratio_limit")
		}
		out := filepath.Join(dest, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0700); err != nil {
				return count, err
			}
			continue
		}
		expanded += size
		if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
			return count, err
		}
		in, err := f.Open()
		if err != nil {
			return count, err
		}
		file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = in.Close()
			return count, err
		}
		_, copyErr := io.Copy(file, contextReader{ctx, in})
		closeErr := file.Close()
		_ = in.Close()
		if copyErr != nil {
			return count, copyErr
		}
		if closeErr != nil {
			return count, closeErr
		}
		count++
		rel := filepath.ToSlash(name)
		files[rel] = true
		if !protected {
			*installed = append(*installed, rel)
		}
	}
	for _, req := range required {
		if !files[strings.TrimPrefix(filepath.ToSlash(req), "/")] {
			return count, fmt.Errorf("required_file_missing:%s", req)
		}
	}
	return count, nil
}

func readZipManifestVersion(z *zip.Reader, prefix string) (string, error) {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	for _, f := range z.File {
		if f.Name != prefix+"fxmanifest.lua" {
			continue
		}
		in, err := f.Open()
		if err != nil {
			return "", err
		}
		b, err := io.ReadAll(io.LimitReader(in, 2<<20))
		_ = in.Close()
		if err != nil {
			return "", err
		}
		v, _ := ParseManifest(b)
		return v, nil
	}
	return "", fmt.Errorf("manifest_missing")
}
