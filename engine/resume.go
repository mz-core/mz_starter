package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func resourceFileHashes(root string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || isReparsePoint(path) {
			return fmt.Errorf("unsafe_resource_link")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsafe_resource_file")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".mz-starter.json" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		files[filepath.ToSlash(rel)] = hex.EncodeToString(h.Sum(nil))
		return nil
	})
	return files, err
}

func unchangedManagedResource(path, id string) (string, error) {
	data, err := os.ReadFile(filepath.Join(path, ".mz-starter.json"))
	if err != nil || len(data) > 4<<20 {
		return "", fmt.Errorf("update_baseline_missing")
	}
	var record managedRecord
	if json.Unmarshal(data, &record) != nil || record.SchemaVersion != 2 || record.ID != id || len(record.Files) == 0 {
		return "", fmt.Errorf("update_baseline_missing")
	}
	files, err := resourceFileHashes(path)
	if err != nil {
		return "", err
	}
	for name := range record.Files {
		if _, exists := files[name]; !exists {
			return "", fmt.Errorf("update_local_changes")
		}
	}
	return record.Version, nil
}

// The marker describes the official package, so preserved files remain detectable later.
func preserveManagedChanges(old, staged, id string) error {
	data, err := os.ReadFile(filepath.Join(old, ".mz-starter.json"))
	if err != nil || len(data) > 4<<20 {
		return fmt.Errorf("update_baseline_missing")
	}
	var record managedRecord
	if json.Unmarshal(data, &record) != nil || record.SchemaVersion != 2 || record.ID != id || len(record.Files) == 0 {
		return fmt.Errorf("update_baseline_missing")
	}
	before, err := resourceFileHashes(old)
	if err != nil {
		return err
	}
	for name := range record.Files {
		if _, ok := before[name]; !ok {
			return fmt.Errorf("update_local_changes")
		}
	}
	for name, hash := range before {
		if record.Files[name] == hash {
			continue
		}
		if strings.EqualFold(name, "fxmanifest.lua") {
			return fmt.Errorf("update_local_changes")
		}
		rel := filepath.FromSlash(name)
		target := filepath.Join(staged, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		in, err := os.Open(filepath.Join(old, rel))
		if err != nil {
			return err
		}
		info, err := in.Stat()
		if err != nil {
			in.Close()
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
		if closeInErr != nil {
			return closeInErr
		}
	}
	after, err := resourceFileHashes(old)
	if err != nil {
		return err
	}
	marker, err := os.ReadFile(filepath.Join(old, ".mz-starter.json"))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) || string(data) != string(marker) {
		return fmt.Errorf("update_local_changes")
	}
	return nil
}

// validateResumeTarget only accepts an existing server-data created by the Starter.
// Arbitrary FiveM installations are refused so their files and configuration stay untouched.
func validateResumeTarget(target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("invalid_directory")
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(abs) {
		return "", fmt.Errorf("existing_installation_missing")
	}
	for _, required := range []string{"server.cfg", "cfg/resources.cfg"} {
		st, err := os.Stat(filepath.Join(abs, filepath.FromSlash(required)))
		if err != nil || !st.Mode().IsRegular() {
			return "", fmt.Errorf("existing_installation_incomplete:%s", required)
		}
	}
	for _, dir := range []string{"cfg", "resources"} {
		st, err := os.Lstat(filepath.Join(abs, dir))
		if dir == "resources" && os.IsNotExist(err) {
			continue
		}
		if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(filepath.Join(abs, dir)) {
			return "", fmt.Errorf("existing_installation_incomplete:%s", dir)
		}
	}
	boot, err := os.ReadFile(filepath.Join(abs, "cfg", "resources.cfg"))
	if err != nil || !strings.Contains(string(boot), "MZ STARTER") {
		return "", fmt.Errorf("existing_installation_not_created_by_starter")
	}
	return abs, nil
}

// installResume completes an existing Starter installation. Existing resource
// directories, server.cfg and cfg/ are never rewritten.
func installResume(ctx context.Context, c Catalog, templates fs.FS, target string, opts FullOptions) (result FullProgress, err error) {
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
	absolute, err := validateResumeTarget(target)
	if err != nil {
		return result, err
	}
	if err = ValidateBootstrap(c, templates); err != nil {
		return result, err
	}
	if err = os.Mkdir(filepath.Join(absolute, "resources"), 0700); err != nil && !os.IsExist(err) {
		return result, fmt.Errorf("existing_resources_create_failed: %w", err)
	}
	if st, e := os.Lstat(filepath.Join(absolute, "resources")); e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(filepath.Join(absolute, "resources")) {
		return result, fmt.Errorf("existing_installation_incomplete:resources")
	}
	parent := filepath.Dir(absolute)
	result.Destination = absolute
	result.State = "resolving"
	emit()
	plan, err := ResolveFull(ctx, c, opts)
	if err != nil {
		return result, err
	}
	result.Plan = &plan
	result.Artifacts = plan.Artifacts
	result.Missing = blockedIDs(plan)
	selected := map[string]bool{}
	for _, id := range opts.UpdateIDs {
		selected[id] = true
	}
	for _, comp := range plan.Components {
		if comp.Status == "ready" && (!comp.Installed || (opts.UpdateExisting && comp.Type == "fivem_resource" && (len(selected) == 0 || selected[comp.ID]))) {
			result.Total++
		}
	}
	work, err := makePrivateTemp(parent)
	if err != nil {
		return result, fmt.Errorf("staging_unavailable: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	downloads := filepath.Join(work, "downloads")
	if err = os.MkdirAll(downloads, 0700); err != nil {
		return result, err
	}
	archiveCache := map[string]string{}
	index := 0
	for i := range plan.Components {
		comp := &plan.Components[i]
		if comp.Status != "ready" || (comp.Installed && (!opts.UpdateExisting || comp.Type != "fivem_resource")) {
			continue
		}
		if comp.Installed && len(selected) > 0 && !selected[comp.ID] {
			continue
		}
		if comp.Installed {
			dst := filepath.Join(absolute, filepath.FromSlash(comp.Destination))
			parts := strings.Split(filepath.ToSlash(comp.Destination), "/")
			if _, e := noSymlinkPath(absolute, parts...); e != nil {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "unsafe_destination"})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			version, e := unchangedManagedResource(dst, comp.ID)
			if e != nil {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: e.Error()})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			if version == comp.Version {
				continue
			}
			if comp.ID == "mz_core" {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "core_migration_review_required"})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			if !semverPattern.MatchString(version) || !semverPattern.MatchString(comp.Version) || !semverGreater(comp.Version, version) {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "update_version_unverified"})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		index++
		result.State, result.Resource, result.Index = "downloading", comp.ID, index
		emit()
		compStage := filepath.Join(work, "stage-"+comp.ID)
		var count int
		switch comp.kind {
		case "github", "direct_release":
			count, err = installReleaseComponent(ctx, comp, compStage, downloads, opts, &result, emit)
		case "external_release":
			count, err = installExternalRelease(ctx, comp, compStage, downloads, opts, &result, emit)
		case "pinned":
			count, err = installPinned(ctx, comp, compStage, downloads, archiveCache, opts, &result, emit)
		case "local_zip", "local_folder":
			count, err = copyLocalPackage(ctx, comp.local, filepath.Join(compStage, filepath.FromSlash(comp.Destination)), comp.Protected)
			if err == nil && comp.contract.Patch == "pma-voice" {
				err = verifyPMAVoiceFile(filepath.Join(compStage, filepath.FromSlash(comp.Destination), "server", "module", "phone.lua"))
			}
		default:
			err = fmt.Errorf("component_source_unknown")
		}
		if err != nil {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: err.Error()})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		if err = verifyInstalledResource(filepath.Join(compStage, filepath.FromSlash(comp.Destination)), comp); err != nil {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: err.Error()})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		if err = writeManagedMarker(compStage, comp); err != nil {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: err.Error()})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		dst := filepath.Join(absolute, filepath.FromSlash(comp.Destination))
		if _, e := os.Lstat(dst); e == nil && !comp.Installed {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "destination_exists"})
			result.Missing = append(result.Missing, comp.ID)
			continue
		} else if e != nil && !os.IsNotExist(e) {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "destination_unavailable"})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return result, err
		}
		parts := strings.Split(filepath.ToSlash(comp.Destination), "/")
		if _, e := noSymlinkPath(absolute, parts...); e != nil {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "unsafe_destination"})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		src := filepath.Join(compStage, filepath.FromSlash(comp.Destination))
		var backup string
		if comp.Installed {
			if _, e := unchangedManagedResource(dst, comp.ID); e != nil {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: e.Error()})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			if e := preserveManagedChanges(dst, src, comp.ID); e != nil {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: e.Error()})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			if e := verifyInstalledResource(src, comp); e != nil {
				result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: e.Error()})
				result.Missing = append(result.Missing, comp.ID)
				continue
			}
			backupDir, e := os.MkdirTemp(parent, ".mz-starter-backup-")
			if e != nil {
				return result, fmt.Errorf("backup_failed: %w", e)
			}
			backup = filepath.Join(backupDir, comp.ID)
			if e = os.Rename(dst, backup); e != nil {
				return result, fmt.Errorf("backup_failed: %w", e)
			}
			result.Backups = append(result.Backups, backup)
		}
		if err = publishNoReplace(src, dst); err != nil {
			if backup != "" {
				if restoreErr := os.Rename(backup, dst); restoreErr != nil {
					return result, fmt.Errorf("restore_failed_backup_at_%s: %w", backup, restoreErr)
				}
			}
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "publication_failed"})
			result.Missing = append(result.Missing, comp.ID)
			continue
		}
		comp.Installed = true
		result.Installed++
		result.InstalledResources = append(result.InstalledResources, comp.ID)
		result.Files += count
		result.State, result.Resource = "installed", comp.ID
		emit()
	}
	result.Resource = ""
	launchWarning := ""
	artifactResult, artifactErr := installArtifacts(ctx, opts, absolute, &result, emit)
	result.Artifacts = artifactResult
	if artifactErr == nil && (artifactResult.Status == "installed" || artifactResult.Status == "existing") {
		launch, created, launchErr := writeLaunchScripts(absolute, artifactResult.Path, true)
		if launchErr != nil {
			launchWarning = "launch_script_failed:" + launchErr.Error()
		} else {
			result.LaunchScript = launch
			result.TxAdminURL = "http://127.0.0.1:40120"
			if created {
				result.Files += 2
			}
		}
	}
	result.Missing = uniqueStrings(result.Missing)
	result.FilesReady = len(result.Missing) == 0
	if !result.FilesReady {
		result.State = "incomplete"
		result.Detail = "missing_required_components"
		if artifactErr != nil {
			result.Detail += "; artifacts_failed:" + artifactErr.Error()
		}
		if launchWarning != "" {
			result.Detail += "; " + launchWarning
		}
		result.ServerReady = false
		emit()
		return result, nil
	}
	if artifactErr != nil {
		result.State = "completed_files_only"
		result.Detail = "artifacts_failed:" + artifactErr.Error()
		result.ServerReady = false
		emit()
		return result, nil
	}
	if artifactResult.Status == "ready" || artifactResult.Status == "installed" || artifactResult.Status == "existing" {
		result.State = "completed"
		result.Detail = "configuration_required"
	} else {
		result.State = "completed_files_only"
		result.Detail = "server_artifacts_not_provided"
	}
	if launchWarning != "" {
		result.Detail += "; " + launchWarning
	}
	result.ServerReady = false
	emit()
	return result, nil
}
