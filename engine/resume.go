package engine

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

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
			return "", fmt.Errorf("existing_installation_incomplete")
		}
	}
	for _, dir := range []string{"cfg", "resources"} {
		st, err := os.Lstat(filepath.Join(abs, dir))
		if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(filepath.Join(abs, dir)) {
			return "", fmt.Errorf("existing_installation_incomplete")
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
	for _, comp := range plan.Components {
		if comp.Status == "ready" && !comp.Installed {
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
		if comp.Status != "ready" || comp.Installed {
			continue
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
		if _, e := os.Lstat(dst); e == nil {
			result.Failures = append(result.Failures, InstallFailure{ID: comp.ID, Reason: "destination_exists"})
			result.Missing = append(result.Missing, comp.ID)
			continue
		} else if !os.IsNotExist(e) {
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
		if err = publishNoReplace(src, dst); err != nil {
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
