@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d %~dp0\..

set "OX_ONLY=0"
set "RUNTIME_DEPS_ONLY=0"
if /I "%~1"=="--help" (
  echo Uso: install_deps.bat [--ox-only^|--runtime-deps-only]
  echo --ox-only instala apenas oxmysql e ox_lib; nao sincroniza Cfx ou repositorios MZ.
  echo --runtime-deps-only instala ox, Cfx, screenshot-basic, pma-voice e bob74_ipl; nao sincroniza repositorios MZ.
  exit /b 0
)
if /I "%~1"=="--ox-only" set "OX_ONLY=1"
if /I "%~1"=="--runtime-deps-only" set "RUNTIME_DEPS_ONLY=1"
if not "%~1"=="" if "%OX_ONLY%"=="0" if "%RUNTIME_DEPS_ONLY%"=="0" (
  echo Argumento desconhecido: %~1
  exit /b 1
)
if not "%~2"=="" (
  echo Argumentos extras nao sao aceitos.
  exit /b 1
)

set "PMA_REPOSITORY=https://github.com/AvarianKnight/pma-voice.git"
set "PMA_VERSION=7.0.0"
set "PMA_TAG=v7.0.0"
set "PMA_COMMIT=3379e45dcf7aaa8b7651317190c108095a121553"
set "PMA_PATCH=%CD%\patches\pma-voice-server-authoritative-calls.patch"
set "BOB74_REPOSITORY=https://github.com/Bob74/bob74_ipl.git"
set "BOB74_VERSION=2.7.1"
set "BOB74_TAG=2.7.1"
set "BOB74_COMMIT=037dd6c71f6894be771e9343008223d7a3305d79"

call :preflight
if errorlevel 1 exit /b 1

if not exist "resources" mkdir "resources"
if not exist "resources\[ox]" mkdir "resources\[ox]"
if not exist "resources\[mz]" mkdir "resources\[mz]"
if not exist "resources\[standalone]" mkdir "resources\[standalone]"
if not exist "resources\[maps]" mkdir "resources\[maps]"
if not exist "resources\[som]" mkdir "resources\[som]"
if not exist "resources\[managers]" mkdir "resources\[managers]"
if not exist "resources\[system]" mkdir "resources\[system]"
if not exist "resources\[system]\[builders]" mkdir "resources\[system]\[builders]"
if not exist "resources\[gameplay]" mkdir "resources\[gameplay]"
if not exist "tmp" mkdir "tmp"

set "CFX_SERVER_DATA_TMP=tmp\cfx-server-data"

powershell -NoProfile -ExecutionPolicy Bypass -File "%CD%\tools\install_ox.ps1"
if errorlevel 1 goto :download_error
if "%OX_ONLY%"=="1" (
  echo Dependencias ox instaladas sem sincronizar Cfx ou repositorios MZ.
  exit /b 0
)

call :prepare_cfx_server_data
if errorlevel 1 goto :git_error

call :sync_cfx_resource mapmanager "%CFX_SERVER_DATA_TMP%\resources\[managers]\mapmanager" "resources\[managers]\mapmanager"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource spawnmanager "%CFX_SERVER_DATA_TMP%\resources\[managers]\spawnmanager" "resources\[managers]\spawnmanager"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource baseevents "%CFX_SERVER_DATA_TMP%\resources\[system]\baseevents" "resources\[system]\baseevents"
if errorlevel 1 goto :sync_error

call :sync_git_repo screenshot-basic https://github.com/citizenfx/screenshot-basic.git "resources\[standalone]\screenshot-basic"
if errorlevel 1 goto :git_error

call :sync_pma_voice
if errorlevel 1 goto :git_error

call :sync_bob74_ipl
if errorlevel 1 goto :git_error

if "%RUNTIME_DEPS_ONLY%"=="1" (
  echo Dependencias runtime instaladas sem sincronizar repositorios MZ.
  exit /b 0
)

call :sync_mz_resource mz_notify https://github.com/mz-core/mz_notify.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_sync https://github.com/mz-core/mz_sync.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_core https://github.com/mz-core/mz_core.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_assets ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_inventory https://github.com/mz-core/mz_inventory.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_economy https://github.com/mz-core/mz_economy.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_status ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_progress https://github.com/mz-core/mz_progress.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_settings https://github.com/mz-core/mz_settings.git
if errorlevel 1 goto :git_error

call :sync_maps_resource mz_cayo
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_menu https://github.com/mz-core/mz_menu.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_interact https://github.com/mz-core/mz_interact.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_world_editor ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_vehicles https://github.com/mz-core/mz_vehicles.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_org https://github.com/mz-core/mz_org.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_target https://github.com/mz-core/mz_target.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_fuel https://github.com/mz-core/mz_fuel.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_garagem https://github.com/mz-core/mz_garagem.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_prison ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_dispatch ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_police ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_medical ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_bank https://github.com/mz-core/mz_bank.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_phone https://github.com/mz-core/mz_phone.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_transport ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_interiors https://github.com/mz-core/mz_interiors.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_houses https://github.com/mz-core/mz_houses.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_furniture ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_org_activities https://github.com/mz-core/mz_org_activities.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_computer ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_radio https://github.com/mz-core/mz_radio.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_creator https://github.com/mz-core/mz_creator.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_multichar ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_clothing https://github.com/mz-core/mz_clothing.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_hud https://github.com/mz-core/mz_hud.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_banguard https://github.com/mz-core/mz_banguard.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_warns ""
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_admin https://github.com/mz-core/mz_admin.git
if errorlevel 1 goto :git_error

call :sync_mz_resource mz_feedback ""
if errorlevel 1 goto :git_error

echo Dependencias do projeto instaladas.
pause
exit /b 0

:preflight
if not exist "server.cfg" (
  echo [preflight] server.cfg nao foi encontrado na raiz do projeto.
  echo [preflight] Use o mz_starter como raiz do server-data antes de rodar este instalador.
  exit /b 1
)

if not exist "cfg\resources.cfg" (
  echo [preflight] cfg\resources.cfg nao foi encontrado.
  echo [preflight] A estrutura minima esperada do mz_starter esta incompleta.
  exit /b 1
)

if "%OX_ONLY%"=="0" (
  where git >nul 2>nul
  if errorlevel 1 (
    echo [preflight] Git nao foi encontrado no PATH.
    echo [preflight] Instale o Git antes de rodar este instalador.
    exit /b 1
  )
)

if "%OX_ONLY%"=="0" if "%RUNTIME_DEPS_ONLY%"=="0" (
  call :preflight_distribution_resources
  if errorlevel 1 exit /b 1
)

echo [preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto...
exit /b 0

:preflight_distribution_resources
set "DISTRIBUTION_MISSING=0"
for %%R in (
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
) do (
  if not exist "resources\[mz]\%%R\fxmanifest.lua" (
    if not defined MZ_RESOURCE_SOURCE_ROOT (
      echo [preflight] %%R nao esta instalado e MZ_RESOURCE_SOURCE_ROOT nao foi definido.
      set "DISTRIBUTION_MISSING=1"
    ) else if not exist "%MZ_RESOURCE_SOURCE_ROOT%\%%R\fxmanifest.lua" (
      echo [preflight] %%R nao foi encontrado em "%MZ_RESOURCE_SOURCE_ROOT%\%%R".
      set "DISTRIBUTION_MISSING=1"
    )
  )
)
if not exist "resources\[maps]\mz_cayo\fxmanifest.lua" (
  if not defined MZ_RESOURCE_SOURCE_ROOT (
    echo [preflight] mz_cayo nao esta instalado e MZ_RESOURCE_SOURCE_ROOT nao foi definido.
    set "DISTRIBUTION_MISSING=1"
  ) else if not exist "!MZ_RESOURCE_SOURCE_ROOT!\mz_cayo\fxmanifest.lua" (
    echo [preflight] mz_cayo nao foi encontrado em "!MZ_RESOURCE_SOURCE_ROOT!\mz_cayo".
    set "DISTRIBUTION_MISSING=1"
  )
)
if "%DISTRIBUTION_MISSING%"=="1" (
  echo [preflight] A distribuicao MZ esta incompleta; nenhuma dependencia foi alterada.
  echo [preflight] Aponte MZ_RESOURCE_SOURCE_ROOT para a raiz que contem os resources mz_* atuais.
  exit /b 1
)
exit /b 0

:prepare_cfx_server_data
if exist "%CFX_SERVER_DATA_TMP%\.git" (
  echo Atualizando cfx-server-data oficial...
  git -C "%CFX_SERVER_DATA_TMP%" pull --ff-only
  if errorlevel 1 exit /b 1
) else (
  if exist "%CFX_SERVER_DATA_TMP%" rmdir /s /q "%CFX_SERVER_DATA_TMP%"
  echo Clonando cfx-server-data oficial...
  git clone --depth 1 https://github.com/citizenfx/cfx-server-data.git "%CFX_SERVER_DATA_TMP%"
  if errorlevel 1 exit /b 1
)

exit /b 0

:sync_cfx_resource
set "RESOURCE_NAME=%~1"
set "RESOURCE_SOURCE=%~2"
set "RESOURCE_TARGET=%~3"
for %%I in ("%RESOURCE_TARGET%") do set "RESOURCE_TARGET_FULL=%%~fI"
set "FOUND_RESOURCE_PATH="

for /f "delims=" %%D in ('dir /ad /b /s "resources\%RESOURCE_NAME%" 2^>nul') do (
  if not defined FOUND_RESOURCE_PATH set "FOUND_RESOURCE_PATH=%%D"
)

if defined FOUND_RESOURCE_PATH (
  if /I not "!FOUND_RESOURCE_PATH!"=="!RESOURCE_TARGET_FULL!" (
    echo %RESOURCE_NAME% ja existe em "!FOUND_RESOURCE_PATH!"; mantendo recurso existente para evitar duplicidade.
    exit /b 0
  )
  echo Atualizando %RESOURCE_NAME% do cfx-server-data oficial...
) else (
  echo Instalando %RESOURCE_NAME% do cfx-server-data oficial...
)

if exist "%RESOURCE_TARGET%" rmdir /s /q "%RESOURCE_TARGET%"
robocopy "%RESOURCE_SOURCE%" "%RESOURCE_TARGET%" /E >nul
set "ROBOCOPY_EXIT=!errorlevel!"
if !ROBOCOPY_EXIT! GEQ 8 exit /b 1
exit /b 0

:sync_git_repo
set "REPO_LABEL=%~1"
set "REPO_URL=%~2"
set "REPO_TARGET=%~3"

if exist "%REPO_TARGET%\.git" (
  echo Atualizando %REPO_LABEL%...
  git -C "%REPO_TARGET%" pull --ff-only
  if errorlevel 1 exit /b 1
) else (
  if exist "%REPO_TARGET%" (
    echo %REPO_LABEL% ja existe sem .git; mantendo como esta.
  ) else (
    echo Clonando %REPO_LABEL%...
    git clone --depth 1 "%REPO_URL%" "%REPO_TARGET%"
    if errorlevel 1 exit /b 1
  )
)

exit /b 0

:sync_mz_resource
set "MZ_RESOURCE_NAME=%~1"
set "MZ_RESOURCE_REPOSITORY=%~2"
set "MZ_RESOURCE_TARGET=resources\[mz]\%~1"

if exist "%MZ_RESOURCE_TARGET%\" goto :sync_mz_existing
if defined MZ_RESOURCE_SOURCE_ROOT if exist "%MZ_RESOURCE_SOURCE_ROOT%\%MZ_RESOURCE_NAME%\fxmanifest.lua" goto :sync_mz_local
if not "%MZ_RESOURCE_REPOSITORY%"=="" goto :sync_mz_git

echo %MZ_RESOURCE_NAME% nao esta instalado e nao possui repositorio publico configurado.
echo Defina MZ_RESOURCE_SOURCE_ROOT para uma distribuicao MZ completa.
exit /b 1

:sync_mz_existing
if "%MZ_RESOURCE_REPOSITORY%"=="" goto :sync_mz_verify
if not exist "%MZ_RESOURCE_TARGET%\.git\" (
  echo %MZ_RESOURCE_NAME% ja existe sem repositorio atualizavel; mantendo como esta.
  goto :sync_mz_verify
)
call :sync_git_repo "%MZ_RESOURCE_NAME%" "%MZ_RESOURCE_REPOSITORY%" "%MZ_RESOURCE_TARGET%"
if errorlevel 1 exit /b 1
goto :sync_mz_verify

:sync_mz_local
echo Instalando %MZ_RESOURCE_NAME% da distribuicao local MZ...
robocopy "%MZ_RESOURCE_SOURCE_ROOT%\%MZ_RESOURCE_NAME%" "%MZ_RESOURCE_TARGET%" /E /XD .git graphify-out node_modules >nul
set "ROBOCOPY_EXIT=!errorlevel!"
if !ROBOCOPY_EXIT! GEQ 8 exit /b 1
goto :sync_mz_verify

:sync_mz_git
call :sync_git_repo "%MZ_RESOURCE_NAME%" "%MZ_RESOURCE_REPOSITORY%" "%MZ_RESOURCE_TARGET%"
if errorlevel 1 exit /b 1

:sync_mz_verify
if not exist "%MZ_RESOURCE_TARGET%\fxmanifest.lua" (
  echo %MZ_RESOURCE_NAME% foi resolvido sem fxmanifest.lua em "%MZ_RESOURCE_TARGET%".
  exit /b 1
)
exit /b 0

:sync_bob74_ipl
set "BOB74_COUNT=0"
set "BOB74_TARGET="
set "BOB74_PATH="
set "BOB74_TARGET=resources\[maps]\bob74_ipl"
for %%I in ("%BOB74_TARGET%") do set "BOB74_TARGET_FULL=%%~fI"
for /d /r "resources" %%D in (bob74_ipl) do (
  if exist "%%~D\" (
    set /a BOB74_COUNT+=1
    set "BOB74_PATH=%%D"
  )
)
if !BOB74_COUNT! GTR 1 (
  echo Foram encontradas varias instalacoes de bob74_ipl; nenhuma foi alterada.
  exit /b 1
)
if "!BOB74_COUNT!"=="0" (
  if exist "!BOB74_TARGET!" (
    echo O destino !BOB74_TARGET! existe sem ser uma pasta Bob reconhecida; nenhuma alteracao foi feita.
    exit /b 1
  )
  echo Clonando bob74_ipl %BOB74_TAG%...
  git clone --branch "%BOB74_TAG%" --depth 1 "%BOB74_REPOSITORY%" "!BOB74_TARGET!"
  if errorlevel 1 exit /b 1
) else (
  for %%I in ("!BOB74_PATH!") do set "BOB74_PATH_FULL=%%~fI"
  if /I not "!BOB74_PATH_FULL!"=="!BOB74_TARGET_FULL!" (
    for %%I in ("%CD%\resources\[standalone]\bob74_ipl") do set "BOB74_LEGACY_FULL=%%~fI"
    if /I not "!BOB74_PATH_FULL!"=="!BOB74_LEGACY_FULL!" (
      echo bob74_ipl existe em caminho inesperado "!BOB74_PATH!"; mova-o manualmente para "!BOB74_TARGET!".
      exit /b 1
    )
    if not exist "!BOB74_PATH!\.git\" (
      echo bob74_ipl no caminho antigo nao tem .git; pin/limpeza nao podem ser verificados. Migre manualmente apos backup.
      exit /b 1
    )
    call :verify_bob74_ipl "!BOB74_PATH!"
    if errorlevel 1 exit /b 1
    if exist "!BOB74_TARGET!" (
      echo Destino "!BOB74_TARGET!" ja existe; nenhuma copia Bob foi movida ou sobrescrita.
      exit /b 1
    )
    echo Movendo bob74_ipl validado de "!BOB74_PATH!" para "!BOB74_TARGET!"...
    move "!BOB74_PATH!" "!BOB74_TARGET!" >nul
    if errorlevel 1 exit /b 1
  ) else (
    echo Reutilizando bob74_ipl existente em "!BOB74_TARGET!"...
  )
)

call :verify_bob74_ipl "!BOB74_TARGET!"
if errorlevel 1 exit /b 1
exit /b 0

:sync_maps_resource
set "MAPS_RESOURCE_NAME=%~1"
set "MAPS_RESOURCE_TARGET=resources\[maps]\%~1"
set "MAPS_RESOURCE_SOURCE=%MZ_RESOURCE_SOURCE_ROOT%\%~1"
set "MAPS_RESOURCE_COUNT=0"
set "MAPS_RESOURCE_PATH="
for /d /r "resources" %%D in (%~1) do (
  if exist "%%~D\" (
    set /a MAPS_RESOURCE_COUNT+=1
    set "MAPS_RESOURCE_PATH=%%D"
  )
)
if !MAPS_RESOURCE_COUNT! GTR 1 (
  echo Foram encontradas varias instalacoes de %~1; nenhuma foi alterada.
  exit /b 1
)
if "!MAPS_RESOURCE_COUNT!"=="1" (
  for %%I in ("!MAPS_RESOURCE_TARGET!") do set "MAPS_RESOURCE_TARGET_FULL=%%~fI"
  for %%I in ("!MAPS_RESOURCE_PATH!") do set "MAPS_RESOURCE_PATH_FULL=%%~fI"
  if /I not "!MAPS_RESOURCE_PATH_FULL!"=="!MAPS_RESOURCE_TARGET_FULL!" (
    echo %~1 existe fora de "!MAPS_RESOURCE_TARGET!"; mova a copia manualmente apos validar o destino.
    exit /b 1
  )
  goto :verify_maps_resource
)
if not defined MZ_RESOURCE_SOURCE_ROOT (
  echo %~1 nao esta instalado e MZ_RESOURCE_SOURCE_ROOT nao foi definido.
  exit /b 1
)
if not exist "!MAPS_RESOURCE_SOURCE!\fxmanifest.lua" (
  echo %~1 nao foi encontrado em "!MAPS_RESOURCE_SOURCE!"; nenhuma copia foi instalada.
  exit /b 1
)
if exist "!MAPS_RESOURCE_TARGET!" (
  echo O destino "!MAPS_RESOURCE_TARGET!" existe e nao e uma pasta de resource valida.
  exit /b 1
)
echo Instalando %~1 da distribuicao local MZ...
robocopy "!MAPS_RESOURCE_SOURCE!" "!MAPS_RESOURCE_TARGET!" /E /XD .git graphify-out node_modules >nul
set "ROBOCOPY_EXIT=!errorlevel!"
if !ROBOCOPY_EXIT! GEQ 8 exit /b 1

:verify_maps_resource
for %%F in (
  "fxmanifest.lua"
  "config.lua"
  "client\ipl_data.lua"
  "client\main.lua"
) do (
  if not exist "!MAPS_RESOURCE_TARGET!\%%~F" (
    echo %~1 sem arquivo obrigatorio: "!MAPS_RESOURCE_TARGET!\%%~F".
    exit /b 1
  )
)
exit /b 0

:verify_bob74_ipl
set "BOB74_VERIFY_TARGET=%~1"
for %%F in (
  "fxmanifest.lua"
  "LICENSE"
  "client.lua"
  "gtav\base.lua"
  "dlc_cayoperico\base.lua"
) do (
  if not exist "!BOB74_VERIFY_TARGET!\%%~F" (
    echo bob74_ipl sem arquivo obrigatorio: "!BOB74_VERIFY_TARGET!\%%~F".
    exit /b 1
  )
)
findstr /L /C:"version '%BOB74_VERSION%'" "!BOB74_VERIFY_TARGET!\fxmanifest.lua" >nul
if errorlevel 1 (
  echo bob74_ipl em "!BOB74_VERIFY_TARGET!" nao corresponde a versao %BOB74_VERSION%.
  exit /b 1
)
if exist "!BOB74_VERIFY_TARGET!\.git\" (
  set "BOB74_CURRENT_COMMIT="
  for /f "delims=" %%H in ('git -C "!BOB74_VERIFY_TARGET!" rev-parse HEAD') do set "BOB74_CURRENT_COMMIT=%%H"
  if /I not "!BOB74_CURRENT_COMMIT!"=="%BOB74_COMMIT%" (
    echo bob74_ipl commit inesperado: !BOB74_CURRENT_COMMIT! ^(esperado: %BOB74_COMMIT%^).
    exit /b 1
  )
  set "BOB74_DIRTY=0"
  for /f "delims=" %%S in ('git -C "!BOB74_VERIFY_TARGET!" status --porcelain --untracked-files^=all') do set "BOB74_DIRTY=1"
  if not "!BOB74_DIRTY!"=="0" (
    echo bob74_ipl possui alteracoes locais; nenhuma atualizacao automatica foi feita.
    exit /b 1
  )
)
exit /b 0

:verify_pma_voice
set "PMA_TARGET=%~1"
set "PMA_CURRENT_COMMIT="
for /f "delims=" %%H in ('git -C "%PMA_TARGET%" rev-parse HEAD') do set "PMA_CURRENT_COMMIT=%%H"
if /I not "%PMA_CURRENT_COMMIT%"=="%PMA_COMMIT%" (
  echo pma-voice commit inesperado: %PMA_CURRENT_COMMIT% ^(esperado: %PMA_COMMIT%^).
  exit /b 1
)

set "PMA_CHANGED_COUNT=0"
set "PMA_CHANGED_CODE="
set "PMA_CHANGED_PATH="
for /f "tokens=1,*" %%A in ('git -C "%PMA_TARGET%" status --porcelain --untracked-files^=all') do (
  set /a PMA_CHANGED_COUNT+=1
  set "PMA_CHANGED_CODE=%%A"
  set "PMA_CHANGED_PATH=%%B"
)
if not "%PMA_CHANGED_COUNT%"=="1" (
  echo pma-voice possui quantidade inesperada de arquivos alterados: %PMA_CHANGED_COUNT%.
  exit /b 1
)
if /I not "%PMA_CHANGED_PATH%"=="server/module/phone.lua" (
  echo pma-voice possui alteracao inesperada em %PMA_CHANGED_PATH%.
  exit /b 1
)

git -C "%PMA_TARGET%" apply --reverse --check "%PMA_PATCH%" >nul 2>nul
if errorlevel 1 (
  echo O patch server-authoritative do pma-voice nao esta aplicado como esperado.
  exit /b 1
)

findstr /c:"RegisterNetEvent('pma-voice:setPlayerCall'" "%PMA_TARGET%\server\module\phone.lua" >nul
if not errorlevel 1 (
  echo O handler client-authoritative de call channel ainda esta presente no pma-voice.
  exit /b 1
)
findstr /c:"exports('setPlayerCall', setPlayerCall)" "%PMA_TARGET%\server\module\phone.lua" >nul
if errorlevel 1 (
  echo O export server setPlayerCall esperado nao foi encontrado no pma-voice.
  exit /b 1
)
exit /b 0

:sync_pma_voice
set "PMA_TARGET=resources\[som]\pma-voice"
if exist "%PMA_TARGET%" if not exist "%PMA_TARGET%\.git" (
  echo pma-voice existe sem .git; nao e possivel validar tag, commit e patch.
  echo Remova ou mova %PMA_TARGET% e rode o instalador novamente.
  exit /b 1
)

if not exist "%PMA_TARGET%\.git" (
  echo Clonando pma-voice %PMA_TAG%...
  git clone --branch "%PMA_TAG%" --depth 1 "%PMA_REPOSITORY%" "%PMA_TARGET%"
  if errorlevel 1 exit /b 1
) else (
  set "PMA_CURRENT_COMMIT="
  for /f "delims=" %%H in ('git -C "%PMA_TARGET%" rev-parse HEAD') do set "PMA_CURRENT_COMMIT=%%H"
  set "PMA_CHANGED_COUNT=0"
  for /f "delims=" %%S in ('git -C "%PMA_TARGET%" status --porcelain --untracked-files^=all') do set /a PMA_CHANGED_COUNT+=1
  if not "!PMA_CHANGED_COUNT!"=="0" (
    if /I "!PMA_CURRENT_COMMIT!"=="%PMA_COMMIT%" (
      call :verify_pma_voice "%PMA_TARGET%"
      if not errorlevel 1 (
        echo pma-voice %PMA_TAG% ja esta pinado e hardened.
        exit /b 0
      )
    )
    echo pma-voice possui alteracoes locais inesperadas; pin abortado sem sobrescrever arquivos.
    exit /b 1
  )

  echo Fixando pma-voice em %PMA_TAG% ^(%PMA_COMMIT%^)...
  git -C "%PMA_TARGET%" fetch --depth 1 origin "refs/tags/%PMA_TAG%"
  if errorlevel 1 exit /b 1
  set "PMA_FETCHED_COMMIT="
  for /f "delims=" %%H in ('git -C "%PMA_TARGET%" rev-parse FETCH_HEAD') do set "PMA_FETCHED_COMMIT=%%H"
  if /I not "!PMA_FETCHED_COMMIT!"=="%PMA_COMMIT%" (
    echo A tag %PMA_TAG% nao corresponde ao commit aprovado.
    exit /b 1
  )
  git -C "%PMA_TARGET%" checkout --detach "%PMA_COMMIT%"
  if errorlevel 1 exit /b 1
)

set "PMA_CURRENT_COMMIT="
for /f "delims=" %%H in ('git -C "%PMA_TARGET%" rev-parse HEAD') do set "PMA_CURRENT_COMMIT=%%H"
if /I not "%PMA_CURRENT_COMMIT%"=="%PMA_COMMIT%" (
  echo Clone do pma-voice nao corresponde ao commit aprovado.
  exit /b 1
)

git -C "%PMA_TARGET%" apply --check "%PMA_PATCH%"
if errorlevel 1 exit /b 1
git -C "%PMA_TARGET%" apply "%PMA_PATCH%"
if errorlevel 1 exit /b 1
call :verify_pma_voice "%PMA_TARGET%"
if errorlevel 1 exit /b 1
echo pma-voice %PMA_VERSION% instalado com call membership server-authoritative.
exit /b 0

:download_error
echo Erro ao instalar dependencias ox. Recursos anteriores foram preservados ou restaurados.
if "%OX_ONLY%"=="0" pause
exit /b 1

:sync_error
echo Erro ao sincronizar um dos recursos oficiais.
pause
exit /b 1

:git_error
echo Erro ao atualizar ou clonar um dos repositorios Git.
pause
exit /b 1
