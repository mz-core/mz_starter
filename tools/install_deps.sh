#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

preflight() {
  if [ ! -f "server.cfg" ]; then
    echo "[preflight] server.cfg nao foi encontrado na raiz do projeto."
    echo "[preflight] Use o mz_starter como raiz do server-data antes de rodar este instalador."
    exit 1
  fi

  if [ ! -f "cfg/resources.cfg" ]; then
    echo "[preflight] cfg/resources.cfg nao foi encontrado."
    echo "[preflight] A estrutura minima esperada do mz_starter esta incompleta."
    exit 1
  fi

  if ! command -v git >/dev/null 2>&1; then
    echo "[preflight] Git nao foi encontrado no PATH."
    echo "[preflight] Instale o Git antes de rodar este instalador."
    exit 1
  fi

  if ! command -v curl >/dev/null 2>&1; then
    echo "[preflight] curl nao foi encontrado no PATH."
    echo "[preflight] Instale o curl antes de rodar este instalador."
    exit 1
  fi

  if ! command -v unzip >/dev/null 2>&1; then
    echo "[preflight] unzip nao foi encontrado no PATH."
    echo "[preflight] Instale o unzip antes de rodar este instalador."
    exit 1
  fi

  echo "[preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto..."
}

cleanup_zip() {
  local file="$1"
  [ -f "$file" ] && rm -f "$file"
  return 0
}

sync_git_repo() {
  local label="$1"
  local repo_url="$2"
  local target="$3"

  if [ -d "$target/.git" ]; then
    echo "Atualizando $label..."
    git -C "$target" pull --ff-only
  elif [ -d "$target" ]; then
    echo "$label ja existe sem .git; mantendo como esta."
  else
    echo "Clonando $label..."
    git clone --depth 1 "$repo_url" "$target"
  fi
}

prepare_cfx_server_data() {
  local repo_path="$1"

  if [ -d "$repo_path/.git" ]; then
    echo "Atualizando cfx-server-data oficial..."
    git -C "$repo_path" pull --ff-only
  else
    rm -rf "$repo_path"
    echo "Clonando cfx-server-data oficial..."
    git clone --depth 1 https://github.com/citizenfx/cfx-server-data.git "$repo_path"
  fi
}

find_resource_path() {
  local name="$1"
  find "resources" -type d -name "$name" -print -quit
}

sync_cfx_resource() {
  local name="$1"
  local source="$2"
  local target="$3"
  local existing=""

  existing="$(find_resource_path "$name" || true)"
  if [ -n "$existing" ] && [ "$existing" != "$target" ]; then
    echo "$name ja existe em $existing; mantendo recurso existente para evitar duplicidade."
    return 0
  fi

  if [ -n "$existing" ]; then
    echo "Atualizando $name do cfx-server-data oficial..."
  else
    echo "Instalando $name do cfx-server-data oficial..."
  fi

  rm -rf "$target"
  mkdir -p "$(dirname "$target")"
  cp -R "$source" "$target"
}

preflight

mkdir -p \
  "resources" \
  "resources/[ox]" \
  "resources/[mz]" \
  "resources/[som]" \
  "resources/[managers]" \
  "resources/[system]" \
  "tmp"

OXMYSQL_ZIP="tmp/oxmysql.zip"
OXLIB_ZIP="tmp/ox_lib.zip"
CFX_SERVER_DATA_TMP="tmp/cfx-server-data"

cleanup_zip "$OXMYSQL_ZIP"
cleanup_zip "$OXLIB_ZIP"

rm -rf "resources/[ox]/oxmysql" "resources/[ox]/ox_lib"

echo "Baixando oxmysql release..."
curl -L "https://github.com/overextended/oxmysql/releases/latest/download/oxmysql.zip" -o "$OXMYSQL_ZIP"

echo "Baixando ox_lib release..."
curl -L "https://github.com/overextended/ox_lib/releases/latest/download/ox_lib.zip" -o "$OXLIB_ZIP"

echo "Extraindo oxmysql..."
unzip -oq "$OXMYSQL_ZIP" -d "resources/[ox]/oxmysql"

echo "Extraindo ox_lib..."
unzip -oq "$OXLIB_ZIP" -d "resources/[ox]/ox_lib"

cleanup_zip "$OXMYSQL_ZIP"
cleanup_zip "$OXLIB_ZIP"

prepare_cfx_server_data "$CFX_SERVER_DATA_TMP"
sync_cfx_resource "mapmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/mapmanager" "resources/[managers]/mapmanager"
sync_cfx_resource "spawnmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/spawnmanager" "resources/[managers]/spawnmanager"
sync_cfx_resource "sessionmanager" "$CFX_SERVER_DATA_TMP/resources/[system]/sessionmanager" "resources/[system]/sessionmanager"

sync_git_repo "mz_core" "https://github.com/mz-core/mz_core.git" "resources/[mz]/mz_core"
sync_git_repo "mz_notify" "https://github.com/mz-core/mz_notify.git" "resources/[mz]/mz_notify"
sync_git_repo "pma-voice" "https://github.com/AvarianKnight/pma-voice.git" "resources/[som]/pma-voice"

echo "Dependencias do projeto instaladas."
