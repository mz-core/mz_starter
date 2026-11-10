package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// writeLaunchScripts creates minimal manual launchers inside the new server-data.
// The scripts never contain credentials and never start FXServer during installation.
func writeLaunchScripts(serverData, artifactsPath string) (string, error) {
	absData, err := filepath.Abs(serverData)
	if err != nil {
		return "", fmt.Errorf("launch_path_invalid")
	}
	absArtifacts, err := filepath.Abs(artifactsPath)
	if err != nil {
		return "", fmt.Errorf("launch_path_invalid")
	}
	if info, err := os.Stat(absData); err != nil || !info.IsDir() {
		return "", fmt.Errorf("launch_server_data_missing")
	}
	if info, err := os.Stat(absArtifacts); err != nil || !info.IsDir() {
		return "", fmt.Errorf("launch_artifacts_missing")
	}
	rel, err := filepath.Rel(absData, absArtifacts)
	relative := err == nil && rel != "" && rel != "." && !filepath.IsAbs(rel)
	if !relative {
		rel = absArtifacts
	}
	winRef := filepath.ToSlash(rel)
	shRef := filepath.ToSlash(rel)
	if relative {
		winRef = "%~dp0" + filepath.FromSlash(filepath.ToSlash(rel))
		shRef = "${SERVER_DATA}/" + shRef
	} else {
		winRef = absArtifacts
		shRef = absArtifacts
	}

	cmd := fmt.Sprintf(`@echo off
setlocal
set "SERVER_DATA=%%~dp0"
set "ARTIFACTS=%s"
for %%%%I in ("%%ARTIFACTS%%") do set "ARTIFACTS=%%%%~fI"
if /I "%%~1"=="--check" (
  if not exist "%%ARTIFACTS%%\FXServer.exe" (
    echo MISSING "%%ARTIFACTS%%\FXServer.exe"
    exit /b 1
  )
  echo SERVER_DATA=%%SERVER_DATA%%
  echo FXSERVER_DIR=%%ARTIFACTS%%
  exit /b 0
)
if not exist "%%ARTIFACTS%%\FXServer.exe" (
  echo FXServer artifact not found: "%%ARTIFACTS%%\FXServer.exe"
  echo Run the MZ Starter installation with artifacts first.
  exit /b 1
)
echo Server-data: "%%SERVER_DATA%%"
echo FXServer:   "%%ARTIFACTS%%"
echo.
echo Starting txAdmin. Open the URL printed below.
echo Choose Existing server and select:
echo   %%SERVER_DATA%%
echo Do not deploy a new server.
echo For a remote VPS, use an SSH tunnel to the txAdmin port (default 40120).
echo.
cd /d "%%SERVER_DATA%%"
"%%ARTIFACTS%%\FXServer.exe"
if errorlevel 1 pause
`, winRef)

	sh := fmt.Sprintf(`#!/bin/sh
set -eu
SERVER_DATA="$(cd "$(dirname "$0")" && pwd)"
ARTIFACTS="%s"
if [ -d "${ARTIFACTS}" ]; then
  ARTIFACTS="$(cd "${ARTIFACTS}" && pwd)"
fi
if [ "${1:-}" = "--check" ]; then
  if [ ! -x "${ARTIFACTS}/run.sh" ]; then
    echo "MISSING ${ARTIFACTS}/run.sh" >&2
    exit 1
  fi
  echo "SERVER_DATA=${SERVER_DATA}"
  echo "FXSERVER_DIR=${ARTIFACTS}"
  exit 0
fi
if [ ! -x "${ARTIFACTS}/run.sh" ]; then
  echo "FXServer artifact not found or not executable: ${ARTIFACTS}/run.sh" >&2
  echo "Run the MZ Starter installation with artifacts first, or chmod u+x the run.sh." >&2
  exit 1
fi
echo "Server-data: ${SERVER_DATA}"
echo "FXServer:   ${ARTIFACTS}"
echo
echo "Starting txAdmin. Open the URL printed below."
echo "Choose Existing server and select:"
echo "  ${SERVER_DATA}"
echo "Do not deploy a new server."
echo "For a remote VPS, use an SSH tunnel to the txAdmin port (default 40120)."
echo
cd "${SERVER_DATA}"
exec "${ARTIFACTS}/run.sh"
`, shRef)

	cmdPath := filepath.Join(absData, "start-txadmin.cmd")
	shPath := filepath.Join(absData, "start-txadmin.sh")
	if err = writeExclusive(cmdPath, cmd, 0700); err != nil {
		return "", fmt.Errorf("launch_script_conflict:start-txadmin.cmd")
	}
	if err = writeExclusive(shPath, sh, 0700); err != nil {
		_ = os.Remove(cmdPath)
		return "", fmt.Errorf("launch_script_conflict:start-txadmin.sh")
	}
	launch := shPath
	if runtime.GOOS == "windows" {
		launch = cmdPath
	}
	return launch, nil
}

func writeExclusive(name, content string, mode os.FileMode) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
