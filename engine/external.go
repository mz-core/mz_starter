package engine

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Pinned external contracts. Archives are immutable GitHub source snapshots at an
// audited commit; the digest of each snapshot is fixed here so a changed download
// fails closed. MZ resources themselves are never installed from source archives.
const (
	pmaVoiceCommit = "3379e45dcf7aaa8b7651317190c108095a121553"
	bob74Commit    = "037dd6c71f6894be771e9343008223d7a3305d79"
	screenCommit   = "5e89d4afb3e28b490344e84b5e2b5ac8cc5d9c75"
	cfxDataCommit  = "c6afa3909c763e3327ed76825e78453286c99f05"

	pmaVoiceDigest = "e198aaa8e0d5ac80c6a0c49ff1a7143fc975d2ae386f08bc5ff15b4198688512"
	bob74Digest    = "dd588fd5c549dbe65bec56f5de177c98b9660c60a59bb213f906723255df990d"
	screenDigest   = "9ab1185293f0365a365ddec628bd2186c34367838248f9d18bbf67776fbe33ec"
	cfxDataDigest  = "c74f781c9f81ee19e20e4df68846eb11d2b4117dafce83dce97464118bf932d1"

	pmaOriginalPhoneSHA = "c0e3bf13cd4cd85262a5242a89729d1ffd03dddf1bf70fa56c280ebab327182d"
	pmaPatchedPhoneSHA  = "7d81a1793147cb890a435179526254772d0b0a23047e6d161b56aef487db4c8a"
)

type ExternalKind string

const (
	ExternalRelease ExternalKind = "external_release"
	ExternalPinned  ExternalKind = "pinned_archive"
)

type ExternalContract struct {
	ID               string
	Kind             ExternalKind
	Repository       string
	Asset            string
	DestinationGroup string
	RequiredFiles    []string
	Version          string
	URL              string
	ArchivePath      string
	ArchiveDigest    string
	ArchiveRoot      string
	ArchiveSubdir    string
	Patch            string
}

type Sources struct {
	CodeloadBase  string
	ChangelogBase string
	RuntimeBase   string
	External      map[string]ExternalContract
}

func defaultSources() Sources {
	contracts := map[string]ExternalContract{}
	add := func(c ExternalContract) { contracts[c.ID] = c }
	add(ExternalContract{ID: "oxmysql", Kind: ExternalPinned, Repository: "https://github.com/overextended/oxmysql", Asset: "oxmysql.zip", DestinationGroup: "[ox]", RequiredFiles: []string{"fxmanifest.lua", "dist/build.js"}, Version: "2.14.3", URL: "https://github.com/overextended/oxmysql/releases/download/v2.14.3/oxmysql.zip", ArchiveDigest: "f6db8f612a36a8a75fccde300e8c4135696b5bbca5770cd7f26458949e1f7872", ArchiveRoot: "oxmysql"})
	add(ExternalContract{ID: "ox_lib", Kind: ExternalPinned, Repository: "https://github.com/overextended/ox_lib", Asset: "ox_lib.zip", DestinationGroup: "[ox]", RequiredFiles: []string{"fxmanifest.lua", "web/build/index.html"}, Version: "3.42.2", URL: "https://github.com/overextended/ox_lib/releases/download/v3.42.2/ox_lib.zip", ArchiveDigest: "a21d3ffcb1d272d4d97309e5fe2b5162d0b474eba052c7fc8267ffbe549018f7", ArchiveRoot: "ox_lib"})
	add(ExternalContract{ID: "pma-voice", Kind: ExternalPinned, DestinationGroup: "[som]", Version: "7.0.0", RequiredFiles: []string{"fxmanifest.lua", "server/module/phone.lua"}, ArchivePath: "/AvarianKnight/pma-voice/zip/" + pmaVoiceCommit, ArchiveDigest: pmaVoiceDigest, ArchiveRoot: "pma-voice-" + pmaVoiceCommit, Patch: "pma-voice"})
	add(ExternalContract{ID: "bob74_ipl", Kind: ExternalPinned, DestinationGroup: "[maps]", Version: "2.7.1", RequiredFiles: []string{"fxmanifest.lua", "LICENSE", "client.lua", "gtav/base.lua", "dlc_cayoperico/base.lua"}, ArchivePath: "/Bob74/bob74_ipl/zip/" + bob74Commit, ArchiveDigest: bob74Digest, ArchiveRoot: "bob74_ipl-" + bob74Commit})
	add(ExternalContract{ID: "screenshot-basic", Kind: ExternalPinned, DestinationGroup: "[standalone]", RequiredFiles: []string{"fxmanifest.lua", "client.config.js", "server.config.js", "ui.config.js"}, ArchivePath: "/citizenfx/screenshot-basic/zip/" + screenCommit, ArchiveDigest: screenDigest, ArchiveRoot: "screenshot-basic-" + screenCommit})
	cfx := func(id, group, subdir string) ExternalContract {
		return ExternalContract{ID: id, Kind: ExternalPinned, DestinationGroup: group, RequiredFiles: []string{"fxmanifest.lua"}, ArchivePath: "/citizenfx/cfx-server-data/zip/" + cfxDataCommit, ArchiveDigest: cfxDataDigest, ArchiveRoot: "cfx-server-data-" + cfxDataCommit, ArchiveSubdir: subdir}
	}
	add(cfx("mapmanager", "[managers]", "resources/[managers]/mapmanager"))
	add(cfx("spawnmanager", "[managers]", "resources/[managers]/spawnmanager"))
	add(cfx("baseevents", "[system]", "resources/[system]/baseevents"))
	return Sources{CodeloadBase: "https://codeload.github.com", ChangelogBase: "https://changelogs-live.fivem.net", RuntimeBase: "https://runtime.fivem.net", External: contracts}
}

func externalContract(s Sources, id string) (ExternalContract, bool) {
	c, ok := s.External[id]
	return c, ok
}

// downloadSpec is a bounded HTTPS download whose bytes must match an expected digest.
type downloadSpec struct {
	URL      string
	Digest   string
	MaxSize  int64
	Client   *http.Client
	Progress func(received, size int64)
}

func downloadVerified(ctx context.Context, spec downloadSpec, dest string) (int64, error) {
	if spec.MaxSize <= 0 {
		spec.MaxSize = 512 << 20
	}
	u, err := url.Parse(spec.URL)
	if err != nil || u.User != nil {
		return 0, fmt.Errorf("invalid_download_url")
	}
	insecureLoopback := spec.Client != nil && u.Scheme == "http" && isLoopbackHost(u.Hostname())
	if u.Scheme != "https" && !insecureLoopback {
		return 0, fmt.Errorf("invalid_download_url")
	}
	client := spec.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil {
				return http.ErrUseLastResponse
			}
			return nil
		}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "mz-starter-installer/2")
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, fmt.Errorf("download_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download_http_error")
	}
	if resp.ContentLength > spec.MaxSize {
		return 0, fmt.Errorf("download_too_large")
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}
	h := sha256.New()
	buf := make([]byte, 64<<10)
	var total int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > spec.MaxSize {
				_ = f.Close()
				return 0, fmt.Errorf("download_too_large")
			}
			if _, w := f.Write(buf[:n]); w != nil {
				_ = f.Close()
				return 0, fmt.Errorf("temporary_storage_error")
			}
			_, _ = h.Write(buf[:n])
			if spec.Progress != nil {
				spec.Progress(total, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = f.Close()
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			return total, fmt.Errorf("download_interrupted")
		}
	}
	if err = f.Close(); err != nil {
		return total, err
	}
	if spec.Digest != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), strings.TrimPrefix(spec.Digest, "sha256:")) {
		return total, fmt.Errorf("sha256_mismatch")
	}
	return total, nil
}

// downloadGitHubAsset resolves an asset by ID through the GitHub API. Redirects are
// limited to GitHub release asset hosts and the token is dropped on redirect.
func downloadGitHubAsset(ctx context.Context, g GitHub, apiBase, repository string, asset Asset, dest string, progress func(received, size int64)) (int64, error) {
	u, err := url.Parse(repository)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil {
		return 0, fmt.Errorf("invalid_repository")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid_repository")
	}
	base := apiBase
	if base == "" {
		base = "https://api.github.com"
	}
	if g.Token != "" && base != "https://api.github.com" {
		return 0, fmt.Errorf("invalid_api_origin")
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
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
		return 0, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "mz-starter-installer/2")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, fmt.Errorf("download_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download_http_error")
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}
	h := sha256.New()
	buf := make([]byte, 64<<10)
	var total int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if asset.Size > 0 && total > asset.Size {
				_ = f.Close()
				return 0, fmt.Errorf("download_size_mismatch")
			}
			if _, w := f.Write(buf[:n]); w != nil {
				_ = f.Close()
				return 0, fmt.Errorf("temporary_storage_error")
			}
			_, _ = h.Write(buf[:n])
			if progress != nil {
				progress(total, asset.Size)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = f.Close()
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			return total, fmt.Errorf("download_interrupted")
		}
	}
	if err = f.Close(); err != nil {
		return total, err
	}
	if asset.Size > 0 && total != asset.Size {
		return total, fmt.Errorf("download_size_mismatch")
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), strings.TrimPrefix(asset.Digest, "sha256:")) {
		return total, fmt.Errorf("sha256_mismatch")
	}
	return total, nil
}

func archiveURL(s Sources, contract ExternalContract) string {
	return strings.TrimRight(s.CodeloadBase, "/") + contract.ArchivePath
}

// applyPMAVoicePatch turns the pinned upstream file into the audited MZ version.
func applyPMAVoicePatch(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	current := hex.EncodeToString(sum[:])
	text := string(b)
	if current == pmaPatchedPhoneSHA {
		return verifyPMAVoiceText(text)
	}
	if current != pmaOriginalPhoneSHA {
		return fmt.Errorf("pma_voice_pin_mismatch")
	}
	block := "\nRegisterNetEvent('pma-voice:setPlayerCall', function(callChannel)\n    setPlayerCall(source, callChannel)\nend)\n"
	if !strings.HasSuffix(text, block) {
		return fmt.Errorf("pma_voice_patch_preimage_mismatch")
	}
	patched := strings.TrimSuffix(text, block)
	patchedSum := sha256.Sum256([]byte(patched))
	if hex.EncodeToString(patchedSum[:]) != pmaPatchedPhoneSHA {
		return fmt.Errorf("pma_voice_patch_result_mismatch")
	}
	if err = os.WriteFile(path, []byte(patched), 0600); err != nil {
		return err
	}
	return verifyPMAVoiceText(patched)
}

func verifyPMAVoiceText(text string) error {
	if strings.Contains(text, "RegisterNetEvent('pma-voice:setPlayerCall'") {
		return fmt.Errorf("pma_voice_client_handler_present")
	}
	if !strings.Contains(text, "exports('setPlayerCall', setPlayerCall)") {
		return fmt.Errorf("pma_voice_export_missing")
	}
	return nil
}

// validatePinnedArchive enforces the archive's pinned digest and root layout before
// any file is extracted.
func validatePinnedArchive(ctx context.Context, filename string, contract ExternalContract) (*zip.ReadCloser, error) {
	z, err := openZip(filename)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSuffix(contract.ArchiveRoot, "/")
	prefix := root + "/"
	if contract.ArchiveSubdir != "" {
		prefix += strings.Trim(contract.ArchiveSubdir, "/") + "/"
	}
	found := false
	for _, f := range z.File {
		if ctx.Err() != nil {
			z.Close()
			return nil, ctx.Err()
		}
		name := strings.TrimPrefix(f.Name, prefix)
		if name == f.Name {
			continue
		}
		found = true
		if !safeArchiveName(name) {
			z.Close()
			return nil, fmt.Errorf("unsafe_archive_path")
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			z.Close()
			return nil, fmt.Errorf("unsafe_archive_type")
		}
	}
	if !found {
		z.Close()
		return nil, fmt.Errorf("archive_root_mismatch")
	}
	return z, nil
}
