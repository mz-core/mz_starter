#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

OX_ONLY=0
RUNTIME_DEPS_ONLY=0
case "${1:-}" in
  --help)
    echo "Uso: install_deps.sh [--ox-only|--runtime-deps-only]"
    echo "--ox-only instala apenas oxmysql e ox_lib; nao sincroniza Cfx ou repositorios MZ."
    echo "--runtime-deps-only instala ox, Cfx, screenshot-basic, pma-voice e bob74_ipl; nao sincroniza repositorios MZ."
    exit 0
    ;;
  --ox-only) OX_ONLY=1 ;;
  --runtime-deps-only) RUNTIME_DEPS_ONLY=1 ;;
  '') ;;
  *) echo "Argumento desconhecido: $1" >&2; exit 1 ;;
esac
if [ "$#" -gt 1 ]; then
  echo "Argumentos extras nao sao aceitos." >&2
  exit 1
fi

PMA_REPOSITORY="https://github.com/AvarianKnight/pma-voice.git"
PMA_VERSION="7.0.0"
PMA_TAG="v7.0.0"
PMA_COMMIT="3379e45dcf7aaa8b7651317190c108095a121553"
PMA_PATCH="$ROOT_DIR/patches/pma-voice-server-authoritative-calls.patch"
BOB74_REPOSITORY="https://github.com/Bob74/bob74_ipl.git"
BOB74_VERSION="2.7.1"
BOB74_TAG="2.7.1"
BOB74_COMMIT="037dd6c71f6894be771e9343008223d7a3305d79"

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

  if [ "$OX_ONLY" -eq 0 ] && ! command -v git >/dev/null 2>&1; then
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

  if [ -n "${MZ_RESOURCE_SOURCE_ROOT:-}" ] && ! command -v tar >/dev/null 2>&1; then
    echo "[preflight] tar nao foi encontrado no PATH." >&2
    echo "[preflight] Instale o tar para copiar a distribuicao local MZ." >&2
    exit 1
  fi

  if [ "$OX_ONLY" -eq 0 ] && [ "$RUNTIME_DEPS_ONLY" -eq 0 ]; then
    preflight_distribution_resources
  fi

  echo "[preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto..."
}

preflight_distribution_resources() {
  local resource=""
  local missing=0
  local distribution_resources=(
    mz_status
    mz_medical
    mz_world_editor
    mz_police
    mz_prison
    mz_furniture
    mz_computer
    mz_transport
    mz_multichar
    mz_warns
    mz_feedback
  )

  for resource in "${distribution_resources[@]}"; do
    if [ -f "resources/[mz]/$resource/fxmanifest.lua" ]; then
      continue
    fi
    if [ -z "${MZ_RESOURCE_SOURCE_ROOT:-}" ]; then
      echo "[preflight] $resource nao esta instalado e MZ_RESOURCE_SOURCE_ROOT nao foi definido." >&2
      missing=1
    elif [ ! -f "$MZ_RESOURCE_SOURCE_ROOT/$resource/fxmanifest.lua" ]; then
      echo "[preflight] $resource nao foi encontrado em $MZ_RESOURCE_SOURCE_ROOT/$resource." >&2
      missing=1
    fi
  done

  if [ ! -f "resources/[maps]/mz_cayo/fxmanifest.lua" ]; then
    if [ -z "${MZ_RESOURCE_SOURCE_ROOT:-}" ]; then
      echo "[preflight] mz_cayo nao esta instalado e MZ_RESOURCE_SOURCE_ROOT nao foi definido." >&2
      missing=1
    elif [ ! -f "$MZ_RESOURCE_SOURCE_ROOT/mz_cayo/fxmanifest.lua" ]; then
      echo "[preflight] mz_cayo nao foi encontrado em $MZ_RESOURCE_SOURCE_ROOT/mz_cayo." >&2
      missing=1
    fi
  fi

  if [ "$missing" -ne 0 ]; then
    echo "[preflight] A distribuicao MZ esta incompleta; nenhuma dependencia foi alterada." >&2
    echo "[preflight] Aponte MZ_RESOURCE_SOURCE_ROOT para a raiz que contem os resources mz_* atuais." >&2
    return 1
  fi
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

sync_mz_resource() {
  local label="$1"
  local repo_url="$2"
  local target="resources/[mz]/$label"
  local source="${MZ_RESOURCE_SOURCE_ROOT:-}/$label"

  if [ -d "$target" ]; then
    if [ -n "$repo_url" ] && [ -d "$target/.git" ]; then
      sync_git_repo "$label" "$repo_url" "$target"
    else
      echo "$label ja existe sem repositorio atualizavel; mantendo como esta."
    fi
  elif [ -n "${MZ_RESOURCE_SOURCE_ROOT:-}" ] && [ -f "$source/fxmanifest.lua" ]; then
    echo "Instalando $label da distribuicao local MZ..."
    mkdir -p "$target"
    (
      cd "$source"
      tar --exclude=.git --exclude=graphify-out --exclude=node_modules -cf - .
    ) | (
      cd "$ROOT_DIR/$target"
      tar -xf -
    )
  elif [ -n "$repo_url" ]; then
    sync_git_repo "$label" "$repo_url" "$target"
  else
    echo "$label nao esta instalado e nao possui repositorio publico configurado." >&2
    echo "Defina MZ_RESOURCE_SOURCE_ROOT para uma distribuicao MZ completa." >&2
    return 1
  fi

  if [ ! -f "$target/fxmanifest.lua" ]; then
    echo "$label foi resolvido sem fxmanifest.lua em $target." >&2
    return 1
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

verify_bob74_ipl() {
  local target="$1"
  local current_commit=""
  local status=""
  local required_file=""

  for required_file in fxmanifest.lua LICENSE client.lua gtav/base.lua dlc_cayoperico/base.lua; do
    if [ ! -f "$target/$required_file" ]; then
      echo "bob74_ipl sem arquivo obrigatório: $target/$required_file." >&2
      return 1
    fi
  done

  if ! grep -Fq "version '$BOB74_VERSION'" "$target/fxmanifest.lua"; then
    echo "bob74_ipl em $target não corresponde à versão $BOB74_VERSION." >&2
    return 1
  fi

  if [ -d "$target/.git" ]; then
    current_commit="$(git -C "$target" rev-parse HEAD)"
    if [ "$current_commit" != "$BOB74_COMMIT" ]; then
      echo "bob74_ipl commit inesperado: $current_commit (esperado: $BOB74_COMMIT)." >&2
      return 1
    fi
    status="$(git -C "$target" status --porcelain --untracked-files=all)"
    if [ -n "$status" ]; then
      echo "bob74_ipl possui alterações locais; nenhuma atualização automática foi feita:" >&2
      echo "$status" >&2
      return 1
    fi
  fi
}

sync_bob74_ipl() {
  local target="resources/[maps]/bob74_ipl"
  local legacy="resources/[standalone]/bob74_ipl"
  local existing=""
  local -a matches=()

  mapfile -t matches < <(find "resources" \( -type d -o -type l \) -name "bob74_ipl" -print)
  if [ "${#matches[@]}" -gt 1 ]; then
    echo "Foram encontradas várias instalações de bob74_ipl; nenhuma foi alterada:" >&2
    printf '%s\n' "${matches[@]}" >&2
    return 1
  fi

  if [ "${#matches[@]}" -eq 1 ]; then
    existing="${matches[0]}"
    if [ "$existing" != "$target" ]; then
      if [ "$existing" != "$legacy" ]; then
        echo "bob74_ipl está em caminho inesperado ($existing); mova-o manualmente para $target." >&2
        return 1
      fi
      if [ ! -d "$existing/.git" ]; then
        echo "bob74_ipl no caminho antigo nao tem .git; pin/limpeza nao podem ser verificados. Migre manualmente apos backup." >&2
        return 1
      fi
      verify_bob74_ipl "$existing" || return 1
      if [ -e "$target" ] || [ -L "$target" ]; then
        echo "Destino $target ja existe; nenhuma copia Bob foi movida ou sobrescrita." >&2
        return 1
      fi
      mkdir -p "resources/[maps]"
      echo "Movendo bob74_ipl validado de $legacy para $target..."
      mv -- "$existing" "$target" || return 1
    else
      echo "Reutilizando bob74_ipl existente em $existing..."
    fi
  else
    mkdir -p "resources/[maps]"
    echo "Clonando bob74_ipl $BOB74_TAG..."
    git clone --branch "$BOB74_TAG" --depth 1 "$BOB74_REPOSITORY" "$target"
  fi

  verify_bob74_ipl "$target"
}

sync_maps_resource() {
  local label="$1"
  local target="resources/[maps]/$label"
  local source="${MZ_RESOURCE_SOURCE_ROOT:-}/$label"
  local existing=""
  local -a matches=()

  mapfile -t matches < <(find "resources" \( -type d -o -type l \) -name "$label" -print)
  if [ "${#matches[@]}" -gt 1 ]; then
    echo "Foram encontradas varias instalacoes de $label; nenhuma foi alterada:" >&2
    printf '%s\n' "${matches[@]}" >&2
    return 1
  fi

  if [ "${#matches[@]}" -eq 1 ]; then
    existing="${matches[0]}"
    if [ "$existing" != "$target" ]; then
      echo "$label existe fora de $target; mova a copia manualmente apos validar o destino." >&2
      return 1
    fi
    for required_file in fxmanifest.lua config.lua client/ipl_data.lua client/main.lua; do
      if [ ! -f "$target/$required_file" ]; then
        echo "$label existente nao tem $required_file em $target." >&2
        return 1
      fi
    done
    echo "Reutilizando $label existente em $target..."
    return 0
  fi

  if [ ! -f "$source/fxmanifest.lua" ]; then
    echo "$label nao foi encontrado em $source; nenhuma copia foi instalada." >&2
    return 1
  fi

  if [ -e "$target" ] || [ -L "$target" ]; then
    echo "O destino $target existe sem ser uma pasta de resource reconhecida." >&2
    return 1
  fi

  local staging=""
  mkdir -p "resources/[maps]"
  staging="$(mktemp -d "resources/[maps]/.${label}.install.XXXXXX")"
  echo "Instalando $label da distribuicao local MZ..."
  if ! (
    cd "$source"
    tar --exclude=.git --exclude=graphify-out --exclude=node_modules -cf - .
  ) | (
    cd "$ROOT_DIR/$staging"
    tar -xf -
  ); then
    rm -rf -- "$staging"
    echo "Falha ao copiar $label; staging removido e destino preservado." >&2
    return 1
  fi

  for required_file in fxmanifest.lua config.lua client/ipl_data.lua client/main.lua; do
    if [ ! -f "$staging/$required_file" ]; then
      rm -rf -- "$staging"
      echo "$label foi copiado sem $required_file; staging removido." >&2
      return 1
    fi
  done

  if ! mv -- "$staging" "$target"; then
    rm -rf -- "$staging"
    echo "Falha ao publicar $label em $target; nenhuma copia existente foi sobrescrita." >&2
    return 1
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
  "resources/[standalone]" \
  "resources/[maps]" \
  "resources/[som]" \
  "resources/[managers]" \
  "resources/[system]" \
  "resources/[system]/[builders]" \
  "resources/[gameplay]" \
  "tmp"

CFX_SERVER_DATA_TMP="tmp/cfx-server-data"

bash "$ROOT_DIR/tools/install_ox.sh"
if [ "$OX_ONLY" -eq 1 ]; then
  echo "Dependencias ox instaladas sem sincronizar Cfx ou repositorios MZ."
  exit 0
fi

prepare_cfx_server_data "$CFX_SERVER_DATA_TMP"
sync_cfx_resource "mapmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/mapmanager" "resources/[managers]/mapmanager"
sync_cfx_resource "spawnmanager" "$CFX_SERVER_DATA_TMP/resources/[managers]/spawnmanager" "resources/[managers]/spawnmanager"
sync_cfx_resource "baseevents" "$CFX_SERVER_DATA_TMP/resources/[system]/baseevents" "resources/[system]/baseevents"
sync_git_repo "screenshot-basic" "https://github.com/citizenfx/screenshot-basic.git" "resources/[standalone]/screenshot-basic"
sync_pma_voice
sync_bob74_ipl
if [ "$RUNTIME_DEPS_ONLY" -eq 1 ]; then
  echo "Dependencias runtime instaladas sem sincronizar repositorios MZ."
  exit 0
fi
sync_mz_resource "mz_notify" "https://github.com/mz-core/mz_notify.git"
sync_mz_resource "mz_sync" "https://github.com/mz-core/mz_sync.git"
sync_mz_resource "mz_core" "https://github.com/mz-core/mz_core.git"
sync_mz_resource "mz_assets" ""
sync_mz_resource "mz_inventory" "https://github.com/mz-core/mz_inventory.git"
sync_mz_resource "mz_economy" "https://github.com/mz-core/mz_economy.git"
sync_mz_resource "mz_status" ""
sync_mz_resource "mz_progress" "https://github.com/mz-core/mz_progress.git"
sync_mz_resource "mz_settings" "https://github.com/mz-core/mz_settings.git"
sync_maps_resource "mz_cayo"
sync_mz_resource "mz_menu" "https://github.com/mz-core/mz_menu.git"
sync_mz_resource "mz_interact" "https://github.com/mz-core/mz_interact.git"
sync_mz_resource "mz_world_editor" ""
sync_mz_resource "mz_vehicles" "https://github.com/mz-core/mz_vehicles.git"
sync_mz_resource "mz_org" "https://github.com/mz-core/mz_org.git"
sync_mz_resource "mz_target" "https://github.com/mz-core/mz_target.git"
sync_mz_resource "mz_fuel" "https://github.com/mz-core/mz_fuel.git"
sync_mz_resource "mz_garagem" "https://github.com/mz-core/mz_garagem.git"
sync_mz_resource "mz_prison" ""
sync_mz_resource "mz_dispatch" ""
sync_mz_resource "mz_police" ""
sync_mz_resource "mz_medical" ""
sync_mz_resource "mz_bank" "https://github.com/mz-core/mz_bank.git"
sync_mz_resource "mz_phone" "https://github.com/mz-core/mz_phone.git"
sync_mz_resource "mz_transport" ""
sync_mz_resource "mz_interiors" "https://github.com/mz-core/mz_interiors.git"
sync_mz_resource "mz_houses" "https://github.com/mz-core/mz_houses.git"
sync_mz_resource "mz_furniture" ""
sync_mz_resource "mz_org_activities" "https://github.com/mz-core/mz_org_activities.git"
sync_mz_resource "mz_computer" ""
sync_mz_resource "mz_radio" "https://github.com/mz-core/mz_radio.git"
sync_mz_resource "mz_creator" "https://github.com/mz-core/mz_creator.git"
sync_mz_resource "mz_multichar" ""
sync_mz_resource "mz_clothing" "https://github.com/mz-core/mz_clothing.git"
sync_mz_resource "mz_hud" "https://github.com/mz-core/mz_hud.git"
sync_mz_resource "mz_banguard" "https://github.com/mz-core/mz_banguard.git"
sync_mz_resource "mz_warns" ""
sync_mz_resource "mz_admin" "https://github.com/mz-core/mz_admin.git"
sync_mz_resource "mz_feedback" ""

echo "Dependencias do projeto instaladas."
