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

  if [ ! -d "resources" ]; then
    echo "[preflight] A pasta resources nao foi encontrada."
    echo "[preflight] Este instalador nao cria a base stock do FiveM."
    echo "[preflight] Prepare o server-data com cfx-server-data (ou equivalente) antes de continuar."
    exit 1
  fi

  if [ -z "$(find "resources" -type d -name "mapmanager" -print -quit)" ]; then
    echo "[preflight] O recurso stock obrigatorio 'mapmanager' nao foi encontrado em resources."
    echo "[preflight] Prepare o server-data com cfx-server-data (ou equivalente) antes de continuar."
    exit 1
  fi

  if [ -z "$(find "resources" -type d -name "spawnmanager" -print -quit)" ]; then
    echo "[preflight] O recurso stock obrigatorio 'spawnmanager' nao foi encontrado em resources."
    echo "[preflight] Prepare o server-data com cfx-server-data (ou equivalente) antes de continuar."
    exit 1
  fi

  echo "[preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto..."
}

preflight

mkdir -p "resources/[ox]" "resources/[mz]" "tmp"

OXMYSQL_ZIP="tmp/oxmysql.zip"
OXLIB_ZIP="tmp/ox_lib.zip"

cleanup_zip() {
  local file="$1"
  [ -f "$file" ] && rm -f "$file"
}

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

if [ -d "resources/[mz]/mz_core/.git" ] || [ -d "resources/[mz]/mz_core" ]; then
  echo "Atualizando mz_core..."
  if [ -d "resources/[mz]/mz_core/.git" ]; then
    git -C "resources/[mz]/mz_core" pull --ff-only
  else
    echo "mz_core ja existe sem .git; mantendo como esta."
  fi
else
  echo "Clonando mz_core..."
  git clone https://github.com/Mazus-Ofc/mz_core.git "resources/[mz]/mz_core"
fi

echo "Dependencias do projeto instaladas."
