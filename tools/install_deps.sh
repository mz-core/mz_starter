#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

PMA_REPOSITORY="https://github.com/AvarianKnight/pma-voice.git"
PMA_VERSION="7.0.0"
PMA_TAG="v7.0.0"
PMA_COMMIT="3379e45dcf7aaa8b7651317190c108095a121553"
PMA_PATCH="$ROOT_DIR/patches/pma-voice-server-authoritative-calls.patch"

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

verify_pma_voice() {
  local target="$1"
  local current_commit=""
  local status=""

  current_commit="$(git -C "$target" rev-parse HEAD)"
  if [ "$current_commit" != "$PMA_COMMIT" ]; then
    echo "pma-voice commit inesperado: $current_commit (esperado: $PMA_COMMIT)."
    return 1
  fi

  status="$(git -C "$target" status --porcelain --untracked-files=all)"
  if [ "$status" != " M server/module/phone.lua" ]; then
    echo "pma-voice possui alteracoes diferentes do patch MZ esperado:"
    echo "$status"
    return 1
  fi

  if ! git -C "$target" apply --reverse --check "$PMA_PATCH" >/dev/null 2>&1; then
    echo "O patch server-authoritative do pma-voice nao esta aplicado como esperado."
    return 1
  fi

  if grep -Fq "RegisterNetEvent('pma-voice:setPlayerCall'" "$target/server/module/phone.lua"; then
    echo "O handler client-authoritative de call channel ainda esta presente no pma-voice."
    return 1
  fi

  if ! grep -Fq "exports('setPlayerCall', setPlayerCall)" "$target/server/module/phone.lua"; then
    echo "O export server setPlayerCall esperado nao foi encontrado no pma-voice."
    return 1
  fi
}

sync_pma_voice() {
  local target="resources/[som]/pma-voice"
  local current_commit=""
  local status=""

  if [ -d "$target" ] && [ ! -d "$target/.git" ]; then
    echo "pma-voice existe sem .git; nao e possivel validar tag, commit e patch."
    echo "Remova ou mova $target e rode o instalador novamente."
    return 1
  fi

  if [ ! -d "$target/.git" ]; then
    echo "Clonando pma-voice $PMA_TAG..."
    git clone --branch "$PMA_TAG" --depth 1 "$PMA_REPOSITORY" "$target"
  else
    current_commit="$(git -C "$target" rev-parse HEAD)"
    status="$(git -C "$target" status --porcelain --untracked-files=all)"

    if [ -n "$status" ]; then
      if [ "$current_commit" = "$PMA_COMMIT" ] && verify_pma_voice "$target"; then
        echo "pma-voice $PMA_TAG ja esta pinado e hardened."
        return 0
      fi
      echo "pma-voice possui alteracoes locais inesperadas; pin abortado sem sobrescrever arquivos."
      return 1
    fi

    echo "Fixando pma-voice em $PMA_TAG ($PMA_COMMIT)..."
    git -C "$target" fetch --depth 1 origin "refs/tags/$PMA_TAG"
    if [ "$(git -C "$target" rev-parse FETCH_HEAD)" != "$PMA_COMMIT" ]; then
      echo "A tag $PMA_TAG nao corresponde ao commit aprovado."
      return 1
    fi
    git -C "$target" checkout --detach "$PMA_COMMIT"
  fi

  if [ "$(git -C "$target" rev-parse HEAD)" != "$PMA_COMMIT" ]; then
    echo "Clone do pma-voice nao corresponde ao commit aprovado."
    return 1
  fi

  git -C "$target" apply --check "$PMA_PATCH"
  git -C "$target" apply "$PMA_PATCH"
  verify_pma_voice "$target"
  echo "pma-voice $PMA_VERSION instalado com call membership server-authoritative."
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
  "resources/[standalone]" \
  "resources/[som]" \
  "resources/[managers]" \
  "resources/[system]" \
  "resources/[system]/[builders]" \
  "resources/[gameplay]" \
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

# Os zips do overextended ja incluem a pasta raiz do recurso.
# Extraindo em resources/[ox] evitamos criar resources/[ox]/ox_lib/ox_lib e similares.
echo "Extraindo oxmysql..."
unzip -oq "$OXMYSQL_ZIP" -d "resources/[ox]"

echo "Extraindo ox_lib..."
unzip -oq "$OXLIB_ZIP" -d "resources/[ox]"

cleanup_zip "$OXMYSQL_ZIP"
cleanup_zip "$OXLIB_ZIP"

prepare_cfx_server_data "$CFX_SERVER_DATA_TMP"
sync_cfx_resource "mapmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/mapmanager" "resources/[managers]/mapmanager"
sync_cfx_resource "spawnmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/spawnmanager" "resources/[managers]/spawnmanager"
sync_cfx_resource "sessionmanager" "$CFX_SERVER_DATA_TMP/resources/[system]/sessionmanager" "resources/[system]/sessionmanager"
sync_cfx_resource "baseevents" "$CFX_SERVER_DATA_TMP/resources/[system]/baseevents" "resources/[system]/baseevents"
sync_cfx_resource "yarn" "$CFX_SERVER_DATA_TMP/resources/[system]/[builders]/yarn" "resources/[system]/[builders]/yarn"
sync_cfx_resource "webpack" "$CFX_SERVER_DATA_TMP/resources/[system]/[builders]/webpack" "resources/[system]/[builders]/webpack"
sync_cfx_resource "chat" "$CFX_SERVER_DATA_TMP/resources/[gameplay]/chat" "resources/[gameplay]/chat"

sync_git_repo "screenshot-basic" "https://github.com/citizenfx/screenshot-basic.git" "resources/[standalone]/screenshot-basic"
sync_pma_voice
sync_git_repo "mz_notify" "https://github.com/mz-core/mz_notify.git" "resources/[mz]/mz_notify"
sync_git_repo "mz_sync" "https://github.com/mz-core/mz_sync.git" "resources/[mz]/mz_sync"
sync_git_repo "mz_core" "https://github.com/mz-core/mz_core.git" "resources/[mz]/mz_core"
sync_git_repo "mz_settings" "https://github.com/mz-core/mz_settings.git" "resources/[mz]/mz_settings"
sync_git_repo "mz_interact" "https://github.com/mz-core/mz_interact.git" "resources/[mz]/mz_interact"
sync_git_repo "mz_vehicles" "https://github.com/mz-core/mz_vehicles.git" "resources/[mz]/mz_vehicles"
sync_git_repo "mz_inventory" "https://github.com/mz-core/mz_inventory.git" "resources/[mz]/mz_inventory"
sync_git_repo "mz_target" "https://github.com/mz-core/mz_target.git" "resources/[mz]/mz_target"
sync_git_repo "mz_radio" "https://github.com/mz-core/mz_radio.git" "resources/[mz]/mz_radio"
sync_git_repo "mz_creator" "https://github.com/mz-core/mz_creator.git" "resources/[mz]/mz_creator"
sync_git_repo "mz_clothing" "https://github.com/mz-core/mz_clothing.git" "resources/[mz]/mz_clothing"
sync_git_repo "mz_garagem" "https://github.com/mz-core/mz_garagem.git" "resources/[mz]/mz_garagem"
sync_git_repo "mz_hud" "https://github.com/mz-core/mz_hud.git" "resources/[mz]/mz_hud"
sync_git_repo "mz_banguard" "https://github.com/mz-core/mz_banguard.git" "resources/[mz]/mz_banguard"
sync_git_repo "mz_admin" "https://github.com/mz-core/mz_admin.git" "resources/[mz]/mz_admin"

echo "Dependencias do projeto instaladas."
