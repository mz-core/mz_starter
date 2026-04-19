@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d %~dp0\..

call :preflight
if errorlevel 1 exit /b 1

if not exist "resources\[ox]" mkdir "resources\[ox]"
if not exist "resources\[mz]" mkdir "resources\[mz]"
if not exist "tmp" mkdir "tmp"

set "OXMYSQL_ZIP=tmp\oxmysql.zip"
set "OXLIB_ZIP=tmp\ox_lib.zip"

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

if exist "resources\[mz]\mz_core\.git" (
  echo Atualizando mz_core...
  git -C "resources\[mz]\mz_core" pull --ff-only
) else (
  if exist "resources\[mz]\mz_core" (
    echo mz_core ja existe sem .git; mantendo como esta.
  ) else (
    echo Clonando mz_core...
    git clone https://github.com/Mazus-Ofc/mz_core.git "resources\[mz]\mz_core"
    if errorlevel 1 goto :git_error
  )
)

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

if not exist "resources" (
  echo [preflight] A pasta resources nao foi encontrada.
  echo [preflight] Este instalador nao cria a base stock do FiveM.
  echo [preflight] Prepare o server-data com cfx-server-data ^(ou equivalente^) antes de continuar.
  exit /b 1
)

call :resource_exists mapmanager
if errorlevel 1 (
  echo [preflight] O recurso stock obrigatorio 'mapmanager' nao foi encontrado em resources.
  echo [preflight] Prepare o server-data com cfx-server-data ^(ou equivalente^) antes de continuar.
  exit /b 1
)

call :resource_exists spawnmanager
if errorlevel 1 (
  echo [preflight] O recurso stock obrigatorio 'spawnmanager' nao foi encontrado em resources.
  echo [preflight] Prepare o server-data com cfx-server-data ^(ou equivalente^) antes de continuar.
  exit /b 1
)

echo [preflight] Estrutura minima encontrada. Prosseguindo com a instalacao das dependencias do projeto...
exit /b 0

:resource_exists
set "FOUND_RESOURCE="
for /f "delims=" %%D in ('dir /ad /b /s "resources\%~1" 2^>nul') do (
  set "FOUND_RESOURCE=1"
)

if defined FOUND_RESOURCE exit /b 0
exit /b 1

:download_error
echo Erro ao baixar uma das dependencias release.
pause
exit /b 1

:extract_error
echo Erro ao extrair uma das dependencias release.
pause
exit /b 1

:git_error
echo Erro ao clonar mz_core.
pause
exit /b 1
