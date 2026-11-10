package engine

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

const (
	artifactMaxEntries  = 20000
	artifactMaxExpanded = uint64(2) << 30
	artifactMaxFile     = uint64(512) << 20
)

// extractArtifactTarXZ extracts the official Linux proot artifact. Absolute
// symlinks are rewritten relative to their chroot root (alpine/) so no link
// escapes the artifact directory on the host.
func extractArtifactTarXZ(ctx context.Context, archive, dest string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := xz.NewReader(file)
	if err != nil {
		return fmt.Errorf("artifact_archive_invalid")
	}
	tr := tar.NewReader(reader)
	if err = os.MkdirAll(dest, 0700); err != nil {
		return err
	}
	entries := 0
	var expanded uint64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("artifact_archive_invalid")
		}
		entries++
		if entries > artifactMaxEntries {
			return fmt.Errorf("artifact_too_many_entries")
		}
		name := strings.TrimSuffix(filepath.ToSlash(hdr.Name), "/")
		if name == "" || !safeArchiveName(name) {
			return fmt.Errorf("artifact_unsafe_path")
		}
		out := filepath.Join(dest, filepath.FromSlash(name))
		if !withinRoot(dest, out) {
			return fmt.Errorf("artifact_unsafe_path")
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(out, 0700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if hdr.Size < 0 || uint64(hdr.Size) > artifactMaxFile || expanded+uint64(hdr.Size) > artifactMaxExpanded {
				return fmt.Errorf("artifact_expanded_size_limit")
			}
			if err = os.MkdirAll(filepath.Dir(out), 0700); err != nil {
				return err
			}
			mode := os.FileMode(0600)
			if hdr.Mode&0111 != 0 {
				mode = 0700
			}
			f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, contextReader{ctx, tr})
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			expanded += uint64(hdr.Size)
		case tar.TypeSymlink:
			target, err := artifactSymlinkTarget(dest, name, hdr.Linkname)
			if err != nil {
				return err
			}
			if err = os.MkdirAll(filepath.Dir(out), 0700); err != nil {
				return err
			}
			if err = os.Symlink(target, out); err != nil {
				return err
			}
		case tar.TypeLink:
			targetName := strings.TrimSuffix(filepath.ToSlash(hdr.Linkname), "/")
			if !safeArchiveName(targetName) {
				return fmt.Errorf("artifact_unsafe_link")
			}
			targetPath := filepath.Join(dest, filepath.FromSlash(targetName))
			if !withinRoot(dest, targetPath) {
				return fmt.Errorf("artifact_unsafe_link")
			}
			if err = os.MkdirAll(filepath.Dir(out), 0700); err != nil {
				return err
			}
			if err = os.Link(targetPath, out); err != nil {
				return err
			}
		case tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			continue
		default:
			return fmt.Errorf("artifact_unsafe_type")
		}
	}
	run := filepath.Join(dest, "run.sh")
	server := filepath.Join(dest, "alpine", "opt", "cfx-server", "FXServer")
	for _, required := range []string{run, server} {
		info, err := os.Stat(required)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("artifact_executable_missing")
		}
	}
	if err = os.Chmod(run, 0700); err != nil {
		return err
	}
	return os.Chmod(server, 0700)
}

// artifactSymlinkTarget converts an archive link into a relative link that stays
// inside the artifact tree. The Linux proot uses absolute paths from its alpine/
// root; rewriting them keeps host paths out of the extracted artifact.
func artifactSymlinkTarget(dest, name, link string) (string, error) {
	out := filepath.Join(dest, filepath.FromSlash(name))
	root := dest
	if name == "alpine" || strings.HasPrefix(name, "alpine/") {
		root = filepath.Join(dest, "alpine")
	}
	if !withinRoot(dest, out) || !withinRoot(root, out) {
		return "", fmt.Errorf("artifact_unsafe_symlink")
	}
	var targetPath string
	if strings.HasPrefix(link, "/") {
		targetPath = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(link, "/")))
	} else {
		targetPath = filepath.Join(filepath.Dir(out), filepath.FromSlash(link))
	}
	targetPath = filepath.Clean(targetPath)
	if !withinRoot(root, targetPath) || !withinRoot(dest, targetPath) {
		return "", fmt.Errorf("artifact_unsafe_symlink")
	}
	rel, err := filepath.Rel(filepath.Dir(out), targetPath)
	if err != nil {
		return "", fmt.Errorf("artifact_unsafe_symlink")
	}
	return rel, nil
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
