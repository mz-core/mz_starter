@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d %~dp0\..

call :preflight
if errorlevel 1 exit /b 1

if not exist "resources" mkdir "resources"
if not exist "resources\[ox]" mkdir "resources\[ox]"
if not exist "resources\[mz]" mkdir "resources\[mz]"
if not exist "resources\[som]" mkdir "resources\[som]"
if not exist "resources\[managers]" mkdir "resources\[managers]"
if not exist "resources\[system]" mkdir "resources\[system]"
if not exist "tmp" mkdir "tmp"

set "OXMYSQL_ZIP=tmp\oxmysql.zip"
set "OXLIB_ZIP=tmp\ox_lib.zip"
set "CFX_SERVER_DATA_TMP=tmp\cfx-server-data"

echo Removendo instalacoes antigas do oxmysql e ox_lib...
if exist "resources\[ox]\oxmysql" rmdir /s /q "resources\[ox]\oxmysql"
if exist "resources\[ox]\ox_lib" rmdir /s /q "resources\[ox]\ox_lib"
if exist "%OXMYSQL_ZIP%" del /f /q "%OXMYSQL_ZIP%"
if exist "%OXLIB_ZIP%" del /f /q "%OXLIB_ZIP%"

echo Baixando oxmysql release...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://github.com/overextended/oxmysql/releases/latest/download/oxmysql.zip' -OutFile '%OXMYSQL_ZIP%'"
if errorlevel 1 goto :download_error

echo Baixando ox_lib release...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://github.com/overextended/ox_lib/releases/latest/download/ox_lib.zip' -OutFile '%OXLIB_ZIP%'"
if errorlevel 1 goto :download_error

echo Extraindo oxmysql...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Expand-Archive -Path '%OXMYSQL_ZIP%' -DestinationPath 'resources\[ox]\oxmysql' -Force"
if errorlevel 1 goto :extract_error

echo Extraindo ox_lib...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Expand-Archive -Path '%OXLIB_ZIP%' -DestinationPath 'resources\[ox]\ox_lib' -Force"
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

call :sync_git_repo mz_core https://github.com/mz-core/mz_core.git "resources\[mz]\mz_core"
if errorlevel 1 goto :git_error

call :sync_git_repo mz_notify https://github.com/mz-core/mz_notify.git "resources\[mz]\mz_notify"
if errorlevel 1 goto :git_error

call :sync_git_repo pma-voice https://github.com/AvarianKnight/pma-voice.git "resources\[som]\pma-voice"
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
