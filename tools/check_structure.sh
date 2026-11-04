#!/bin/bash
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

FAILED=0
BOB74_COMMIT="037dd6c71f6894be771e9343008223d7a3305d79"

check_file() {
  local path="$1"
  if [ -f "$path" ]; then
    echo "[OK] $path"
  else
    echo "[MISSING] $path"
    FAILED=1
  fi
}

check_dir() {
  local path="$1"
  if [ -d "$path" ]; then
    echo "[OK] $path"
  else
    echo "[MISSING] $path"
    FAILED=1
  fi
}

check_artifact_resource() {
  local name="$1"
  local legacy="$2"
  if [ -d "server/citizen/system_resources/$name" ]; then
    echo "[OK] server/citizen/system_resources/$name"
  elif [ -d "$legacy" ]; then
    echo "[OK LEGACY] $legacy"
  else
    echo "[MISSING] FXServer system_resources/$name"
    FAILED=1
  fi
}

echo "Verificando arquivos base do mz_starter..."
for path in \
  server.cfg \
  cfg/base.cfg \
  cfg/database.cfg \
  cfg/endpoints.cfg \
  cfg/onesync.cfg \
  cfg/permissions.cfg \
  cfg/resources.cfg \
  cfg/tags.cfg; do
  check_file "$path"
done

echo
echo "Verificando recursos CFX base..."
check_dir "resources/[managers]/mapmanager"
check_dir "resources/[managers]/spawnmanager"
check_dir "resources/[system]/baseevents"

echo
echo "Verificando recursos fornecidos pelo FXServer artifact..."
check_artifact_resource chat "resources/[gameplay]/chat"
check_artifact_resource yarn "resources/[system]/[builders]/yarn"
check_artifact_resource webpack "resources/[system]/[builders]/webpack"

echo
echo "Verificando dependencias..."
check_dir "resources/[ox]/oxmysql"
check_dir "resources/[ox]/ox_lib"
check_dir "resources/[som]/pma-voice"
check_dir "resources/[standalone]/screenshot-basic"

echo
echo "Verificando grupo de mapas..."
check_dir "resources/[maps]"

echo
echo Verificando bob74_ipl...
bob74_paths=()
while IFS= read -r path; do
  bob74_paths+=("$path")
done < <(find resources \( -type d -o -type l \) -name bob74_ipl -print 2>/dev/null)
if [ "${#bob74_paths[@]}" -ne 1 ]; then
  echo "[MISSING/CONFLICT] esperado exatamente um bob74_ipl; encontrados ${#bob74_paths[@]}."
  FAILED=1
else
  bob74_path="${bob74_paths[0]}"
  if [ "$bob74_path" != "resources/[maps]/bob74_ipl" ]; then
    echo "[INVALID] bob74_ipl deve estar em resources/[maps]/bob74_ipl, encontrado em $bob74_path."
    FAILED=1
  elif [ ! -f "$bob74_path/fxmanifest.lua" ] \
      || [ ! -f "$bob74_path/LICENSE" ] \
      || ! grep -Fq "version '2.7.1'" "$bob74_path/fxmanifest.lua" \
      || [ ! -f "$bob74_path/client.lua" ] \
      || [ ! -f "$bob74_path/gtav/base.lua" ] \
      || [ ! -f "$bob74_path/dlc_cayoperico/base.lua" ]; then
    echo "[INVALID] bob74_ipl em $bob74_path; esperado manifest 2.7.1 e arquivos oficiais."
    FAILED=1
  elif [ -d "$bob74_path/.git" ]; then
    bob74_commit="$(git -C "$bob74_path" rev-parse HEAD 2>/dev/null)"
    bob74_status="$(git -C "$bob74_path" status --porcelain --untracked-files=all 2>/dev/null)"
    if [ "$bob74_commit" != "$BOB74_COMMIT" ] || [ -n "$bob74_status" ]; then
      echo "[INVALID] bob74_ipl diverge do commit pinado ou possui alteracoes locais."
      FAILED=1
    else
      echo "[OK] $bob74_path (Bob74 IPL 2.7.1, commit verificado)"
    fi
  else
    echo "[OK] $bob74_path (Bob74 IPL 2.7.1)"
  fi
fi

echo
echo Verificando mz_cayo...
cayo_paths=()
while IFS= read -r path; do
  cayo_paths+=("$path")
done < <(find resources \( -type d -o -type l \) -name mz_cayo -print 2>/dev/null)
if [ "${#cayo_paths[@]}" -ne 1 ]; then
  echo "[MISSING/CONFLICT] esperado exatamente um mz_cayo; encontrados ${#cayo_paths[@]}."
  FAILED=1
elif [ "${cayo_paths[0]}" != "resources/[maps]/mz_cayo" ]; then
  echo "[INVALID] mz_cayo deve estar em resources/[maps]/mz_cayo, encontrado em ${cayo_paths[0]}."
  FAILED=1
else
  for path in fxmanifest.lua config.lua client/ipl_data.lua client/main.lua; do
    check_file "resources/[maps]/mz_cayo/$path"
  done
fi

echo
echo "Verificando resources MZ declarados no boot..."
while read -r directive resource _; do
  if [ "$directive" = "ensure" ] && [[ "$resource" == mz_* && "$resource" != "mz_cayo" ]]; then
    check_file "resources/[mz]/$resource/fxmanifest.lua"
  fi
done < cfg/resources.cfg

echo
if [ "$FAILED" -eq 0 ]; then
  echo "[check] Estrutura OK."
  exit 0
fi

echo "[check] Estrutura incompleta. Rode tools/install_deps.sh ou confira os resources ausentes."
exit 1
