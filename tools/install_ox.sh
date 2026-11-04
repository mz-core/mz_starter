#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
mkdir -p 'tmp' 'resources/[ox]'
SESSION="$(mktemp -d "$ROOT_DIR/tmp/ox-install-XXXXXXXX")"

declare -A OLD=()
declare -A NEW=()

rollback() {
  local status=$?
  trap - ERR
  set +e
  for name in ox_lib oxmysql; do
    target="resources/[ox]/$name"
    if [[ "${NEW[$name]:-0}" == 1 && -e "$target" ]]; then
      mv "$target" "$SESSION/failed-$name"
    fi
    if [[ "${OLD[$name]:-0}" == 1 && -e "$SESSION/backup-$name" ]]; then
      mv "$SESSION/backup-$name" "$target"
    fi
  done
  echo "Instalacao ox falhou; anteriores restaurados. Diagnostico: $SESSION" >&2
  exit "$status"
}
trap rollback ERR

download_and_validate() {
  local name="$1" url="$2" build="$3"
  local archive="$SESSION/$name.zip" extract="$SESSION/extract-$name"
  echo "Baixando e validando $name..."
  curl --fail --location --retry 3 --silent --show-error --output "$archive" "$url"
  unzip -tqq "$archive"
  local entries
  entries="$(unzip -Z1 "$archive")"
  grep -Fxq "$name/fxmanifest.lua" <<< "$entries"
  grep -Fxq "$name/$build" <<< "$entries"
  mkdir -p "$extract"
  unzip -oq "$archive" -d "$extract"
  test -f "$extract/$name/fxmanifest.lua"
  test -f "$extract/$name/$build"
}

# No existing resource changes until BOTH archives are complete and extracted.
download_and_validate oxmysql 'https://github.com/overextended/oxmysql/releases/latest/download/oxmysql.zip' 'dist/build.js'
download_and_validate ox_lib 'https://github.com/overextended/ox_lib/releases/latest/download/ox_lib.zip' 'web/build/index.html'

for name in oxmysql ox_lib; do
  target="resources/[ox]/$name"
  if [[ -e "$target" ]]; then
    mv "$target" "$SESSION/backup-$name"
    OLD[$name]=1
  fi
  mv "$SESSION/extract-$name/$name" "$target"
  NEW[$name]=1
done

trap - ERR
echo 'oxmysql e ox_lib instalados e validados.'
if [[ "${#OLD[@]}" -gt 0 ]]; then
  echo "Backups recuperaveis dos recursos anteriores: $SESSION"
fi
