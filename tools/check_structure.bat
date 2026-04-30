@echo off
setlocal EnableExtensions
cd /d %~dp0\..

set "FAILED=0"

echo Verificando arquivos base do mz_starter...
for %%F in (
  "server.cfg"
  "cfg\base.cfg"
  "cfg\database.cfg"
  "cfg\endpoints.cfg"
  "cfg\onesync.cfg"
  "cfg\permissions.cfg"
  "cfg\resources.cfg"
  "cfg\tags.cfg"
) do (
  if exist "%%~F" (
    echo [OK] %%~F
  ) else (
    echo [MISSING] %%~F
    set "FAILED=1"
  )
)

echo.
echo Verificando recursos CFX base...
for %%D in (
  "resources\[managers]\mapmanager"
  "resources\[managers]\spawnmanager"
  "resources\[system]\sessionmanager"
  "resources\[system]\baseevents"
  "resources\[gameplay]\chat"
  "resources\[system]\[builders]\yarn"
  "resources\[system]\[builders]\webpack"
) do (
  if exist "%%~D\" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D
    set "FAILED=1"
  )
)

echo.
echo Verificando dependencias...
for %%D in (
  "resources\[ox]\oxmysql"
  "resources\[ox]\ox_lib"
  "resources\[som]\pma-voice"
) do (
  if exist "%%~D\" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D
    set "FAILED=1"
  )
)

echo.
echo Verificando resources MZ...
for %%D in (
  "resources\[mz]\mz_notify"
  "resources\[mz]\mz_sync"
  "resources\[mz]\mz_core"
  "resources\[mz]\mz_settings"
  "resources\[mz]\mz_interact"
  "resources\[mz]\mz_vehicles"
  "resources\[mz]\mz_inventory"
  "resources\[mz]\mz_target"
  "resources\[mz]\mz_radio"
  "resources\[mz]\mz_creator"
  "resources\[mz]\mz_garagem"
  "resources\[mz]\mz_hud"
  "resources\[mz]\mz_admin"
) do (
  if exist "%%~D\" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D
    set "FAILED=1"
  )
)

echo.
if "%FAILED%"=="0" (
  echo [check] Estrutura OK.
  exit /b 0
)

echo [check] Estrutura incompleta. Rode tools\install_deps.bat ou confira os resources ausentes.
exit /b 1
