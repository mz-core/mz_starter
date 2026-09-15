@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d %~dp0\..

set "PMA_REPOSITORY=https://github.com/AvarianKnight/pma-voice.git"
set "PMA_VERSION=7.0.0"
set "PMA_TAG=v7.0.0"
set "PMA_COMMIT=3379e45dcf7aaa8b7651317190c108095a121553"
set "PMA_PATCH=%CD%\patches\pma-voice-server-authoritative-calls.patch"

call :preflight
if errorlevel 1 exit /b 1

if not exist "resources" mkdir "resources"
if not exist "resources\[ox]" mkdir "resources\[ox]"
if not exist "resources\[mz]" mkdir "resources\[mz]"
if not exist "resources\[standalone]" mkdir "resources\[standalone]"
if not exist "resources\[som]" mkdir "resources\[som]"
if not exist "resources\[managers]" mkdir "resources\[managers]"
if not exist "resources\[system]" mkdir "resources\[system]"
if not exist "resources\[system]\[builders]" mkdir "resources\[system]\[builders]"
if not exist "resources\[gameplay]" mkdir "resources\[gameplay]"
if not exist "tmp" mkdir "tmp"

set "OXMYSQL_ZIP=tmp\oxmysql.zip"
set "OXLIB_ZIP=tmp\ox_lib.zip"
set "OXMYSQL_TMP=tmp\extract_oxmysql"
set "OXLIB_TMP=tmp\extract_ox_lib"
set "CFX_SERVER_DATA_TMP=tmp\cfx-server-data"

echo Removendo instalacoes antigas do oxmysql e ox_lib...
if exist "resources\[ox]\oxmysql" rmdir /s /q "resources\[ox]\oxmysql"
if exist "resources\[ox]\ox_lib" rmdir /s /q "resources\[ox]\ox_lib"
if exist "%OXMYSQL_ZIP%" del /f /q "%OXMYSQL_ZIP%"
if exist "%OXLIB_ZIP%" del /f /q "%OXLIB_ZIP%"
if exist "%OXMYSQL_TMP%" rmdir /s /q "%OXMYSQL_TMP%"
if exist "%OXLIB_TMP%" rmdir /s /q "%OXLIB_TMP%"

echo Baixando oxmysql release...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://github.com/overextended/oxmysql/releases/latest/download/oxmysql.zip' -OutFile '%OXMYSQL_ZIP%'"
if errorlevel 1 goto :download_error

echo Baixando ox_lib release...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://github.com/overextended/ox_lib/releases/latest/download/ox_lib.zip' -OutFile '%OXLIB_ZIP%'"
if errorlevel 1 goto :download_error

rem Os zips do overextended ja incluem a pasta raiz do recurso.
rem No Windows extraimos primeiro em tmp e movemos a pasta final para evitar duplicidade.
echo Extraindo oxmysql...
call :extract_release_resource "%OXMYSQL_ZIP%" "%OXMYSQL_TMP%" "oxmysql"
if errorlevel 1 goto :extract_error

echo Extraindo ox_lib...
call :extract_release_resource "%OXLIB_ZIP%" "%OXLIB_TMP%" "ox_lib"
if errorlevel 1 goto :extract_error

if exist "%OXMYSQL_ZIP%" del /f /q "%OXMYSQL_ZIP%"
if exist "%OXLIB_ZIP%" del /f /q "%OXLIB_ZIP%"

call :prepare_cfx_server_data
if errorlevel 1 goto :git_error

call :sync_cfx_resource mapmanager "%CFX_SERVER_DATA_TMP%\resources\[managers]\mapmanager" "resources\[managers]\mapmanager"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource spawnmanager "%CFX_SERVER_DATA_TMP%\resources\[managers]\spawnmanager" "resources\[managers]\spawnmanager"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource sessionmanager "%CFX_SERVER_DATA_TMP%\resources\[system]\sessionmanager" "resources\[system]\sessionmanager"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource baseevents "%CFX_SERVER_DATA_TMP%\resources\[system]\baseevents" "resources\[system]\baseevents"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource yarn "%CFX_SERVER_DATA_TMP%\resources\[system]\[builders]\yarn" "resources\[system]\[builders]\yarn"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource webpack "%CFX_SERVER_DATA_TMP%\resources\[system]\[builders]\webpack" "resources\[system]\[builders]\webpack"
if errorlevel 1 goto :sync_error

call :sync_cfx_resource chat "%CFX_SERVER_DATA_TMP%\resources\[gameplay]\chat" "resources\[gameplay]\chat"
if errorlevel 1 goto :sync_error

call :sync_git_repo screenshot-basic https://github.com/citizenfx/screenshot-basic.git "resources\[standalone]\screenshot-basic"
if errorlevel 1 goto :git_error

call :sync_pma_voice
if errorlevel 1 goto :git_error

call :sync_git_repo mz_notify https://github.com/mz-core/mz_notify.git "resources\[mz]\mz_notify"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_sync https://github.com/mz-core/mz_sync.git "resources\[mz]\mz_sync"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_core https://github.com/mz-core/mz_core.git "resources\[mz]\mz_core"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_settings https://github.com/mz-core/mz_settings.git "resources\[mz]\mz_settings"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_interact https://github.com/mz-core/mz_interact.git "resources\[mz]\mz_interact"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_vehicles https://github.com/mz-core/mz_vehicles.git "resources\[mz]\mz_vehicles"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_inventory https://github.com/mz-core/mz_inventory.git "resources\[mz]\mz_inventory"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_target https://github.com/mz-core/mz_target.git "resources\[mz]\mz_target"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_radio https://github.com/mz-core/mz_radio.git "resources\[mz]\mz_radio"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_creator https://github.com/mz-core/mz_creator.git "resources\[mz]\mz_creator"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_clothing https://github.com/mz-core/mz_clothing.git "resources\[mz]\mz_clothing"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_garagem https://github.com/mz-core/mz_garagem.git "resources\[mz]\mz_garagem"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_hud https://github.com/mz-core/mz_hud.git "resources\[mz]\mz_hud"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_banguard https://github.com/mz-core/mz_banguard.git "resources\[mz]\mz_banguard"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_admin https://github.com/mz-core/mz_admin.git "resources\[mz]\mz_admin"
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

where git >nul 2>nul
if errorlevel 1 (
  echo [preflight] Git nao foi encontrado no PATH.
  echo [preflight] Instale o Git antes de rodar este instalador.
  exit /b 1
)

echo [preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto...
exit /b 0

:extract_release_resource
set "ARCHIVE_PATH=%~1"
set "EXTRACT_TMP=%~2"
set "RESOURCE_NAME=%~3"

if exist "%EXTRACT_TMP%" rmdir /s /q "%EXTRACT_TMP%"
powershell -NoProfile -ExecutionPolicy Bypass -Command "Expand-Archive -Path '%ARCHIVE_PATH%' -DestinationPath '%EXTRACT_TMP%' -Force"
if errorlevel 1 exit /b 1

if not exist "%EXTRACT_TMP%\%RESOURCE_NAME%" (
  echo Nao foi possivel localizar a pasta %RESOURCE_NAME% dentro do zip baixado.
  exit /b 1
)

robocopy "%EXTRACT_TMP%\%RESOURCE_NAME%" "resources\[ox]\%RESOURCE_NAME%" /E >nul
set "ROBOCOPY_EXIT=%errorlevel%"
if %ROBOCOPY_EXIT% GEQ 8 exit /b 1

if exist "%EXTRACT_TMP%" rmdir /s /q "%EXTRACT_TMP%"
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
echo Erro ao baixar uma das dependencias release.
pause
exit /b 1

:extract_error
echo Erro ao extrair uma das dependencias release.
pause
exit /b 1

:sync_error
echo Erro ao sincronizar um dos recursos oficiais.
pause
exit /b 1

:git_error
echo Erro ao atualizar ou clonar um dos repositorios Git.
pause
exit /b 1
