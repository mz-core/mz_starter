@echo off
setlocal EnableExtensions EnableDelayedExpansion
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
  "resources\[system]\baseevents"
) do (
  if exist "%%~D\" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D
    set "FAILED=1"
  )
)

echo.
echo Verificando recursos fornecidos pelo FXServer artifact...
if exist "server\citizen\system_resources\chat\" (
  echo [OK] server\citizen\system_resources\chat
) else if exist "resources\[gameplay]\chat\" (
  echo [OK LEGACY] resources\[gameplay]\chat
) else (
  echo [MISSING] FXServer system_resources\chat
  set "FAILED=1"
)
if exist "server\citizen\system_resources\yarn\" (
  echo [OK] server\citizen\system_resources\yarn
) else if exist "resources\[system]\[builders]\yarn\" (
  echo [OK LEGACY] resources\[system]\[builders]\yarn
) else (
  echo [MISSING] FXServer system_resources\yarn
  set "FAILED=1"
)
if exist "server\citizen\system_resources\webpack\" (
  echo [OK] server\citizen\system_resources\webpack
) else if exist "resources\[system]\[builders]\webpack\" (
  echo [OK LEGACY] resources\[system]\[builders]\webpack
) else (
  echo [MISSING] FXServer system_resources\webpack
  set "FAILED=1"
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
echo Verificando dependencias adicionais...
for %%D in (
  "resources\[standalone]\screenshot-basic"
) do (
  if exist "%%~D\" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D
    set "FAILED=1"
  )
)

echo.
echo Verificando grupo de mapas...
if exist "resources\[maps]\" (
  echo [OK] resources\[maps]
) else (
  echo [MISSING] resources\[maps]
  set "FAILED=1"
)

echo.
echo Verificando bob74_ipl...
set "BOB74_COUNT=0"
set "BOB74_PATH="
for /d /r "resources" %%D in (bob74_ipl) do (
  if exist "%%~D\" (
    set /a BOB74_COUNT+=1
    set "BOB74_PATH=%%D"
  )
)
if not "!BOB74_COUNT!"=="1" (
  echo [MISSING/CONFLICT] esperado exatamente um bob74_ipl; encontrados !BOB74_COUNT!.
  set "FAILED=1"
) else (
  for %%I in ("!BOB74_PATH!") do set "BOB74_PATH_FULL=%%~fI"
  for %%I in ("%CD%\resources\[maps]\bob74_ipl") do set "BOB74_TARGET_FULL=%%~fI"
  if /I not "!BOB74_PATH_FULL!"=="!BOB74_TARGET_FULL!" (
    echo [INVALID] bob74_ipl deve estar em resources\[maps]\bob74_ipl, encontrado em "!BOB74_PATH!".
    set "FAILED=1"
  ) else (
    findstr /L /C:"version '2.7.1'" "!BOB74_PATH!\fxmanifest.lua" >nul
    if errorlevel 1 (
      echo [INVALID] versao bob74_ipl diferente de 2.7.1 em "!BOB74_PATH!".
      set "FAILED=1"
    ) else if not exist "!BOB74_PATH!\LICENSE" (
      echo [INVALID] LICENSE ausente em "!BOB74_PATH!".
      set "FAILED=1"
    ) else if not exist "!BOB74_PATH!\client.lua" (
      echo [INVALID] client.lua ausente em "!BOB74_PATH!".
      set "FAILED=1"
    ) else if not exist "!BOB74_PATH!\gtav\base.lua" (
      echo [INVALID] gtav\base.lua ausente em "!BOB74_PATH!".
      set "FAILED=1"
    ) else if not exist "!BOB74_PATH!\dlc_cayoperico\base.lua" (
      echo [INVALID] dlc_cayoperico\base.lua ausente em "!BOB74_PATH!".
      set "FAILED=1"
    ) else (
      if exist "!BOB74_PATH!\.git\" (
        set "BOB74_CURRENT_COMMIT="
        for /f "delims=" %%H in ('git -C "!BOB74_PATH!" rev-parse HEAD') do set "BOB74_CURRENT_COMMIT=%%H"
        if /I not "!BOB74_CURRENT_COMMIT!"=="037dd6c71f6894be771e9343008223d7a3305d79" (
          echo [INVALID] bob74_ipl commit inesperado em "!BOB74_PATH!".
          set "FAILED=1"
        ) else (
          set "BOB74_DIRTY=0"
          for /f "delims=" %%S in ('git -C "!BOB74_PATH!" status --porcelain --untracked-files^=all') do set "BOB74_DIRTY=1"
          if not "!BOB74_DIRTY!"=="0" (
            echo [INVALID] bob74_ipl possui alteracoes locais em "!BOB74_PATH!".
            set "FAILED=1"
          ) else (
            echo [OK] !BOB74_PATH! ^(Bob74 IPL 2.7.1, commit verificado^)
          )
        )
      ) else (
        echo [OK] !BOB74_PATH! ^(Bob74 IPL 2.7.1^)
      )
    )
  )
)

echo.
echo Verificando mz_cayo...
set "CAYO_COUNT=0"
set "CAYO_PATH="
for /d /r "resources" %%D in (mz_cayo) do (
  if exist "%%~D\" (
    set /a CAYO_COUNT+=1
    set "CAYO_PATH=%%D"
  )
)
if not "!CAYO_COUNT!"=="1" (
  echo [MISSING/CONFLICT] esperado exatamente um mz_cayo; encontrados !CAYO_COUNT!.
  set "FAILED=1"
) else (
  for %%I in ("!CAYO_PATH!") do set "CAYO_PATH_FULL=%%~fI"
  for %%I in ("%CD%\resources\[maps]\mz_cayo") do set "CAYO_TARGET_FULL=%%~fI"
  if /I not "!CAYO_PATH_FULL!"=="!CAYO_TARGET_FULL!" (
    echo [INVALID] mz_cayo deve estar em resources\[maps]\mz_cayo, encontrado em "!CAYO_PATH!".
    set "FAILED=1"
  ) else (
    for %%F in ("fxmanifest.lua" "config.lua" "client\ipl_data.lua" "client\main.lua") do (
      if not exist "!CAYO_PATH!\%%~F" (
        echo [MISSING] "!CAYO_PATH!\%%~F"
        set "FAILED=1"
      )
    )
  )
)

echo.
echo Verificando resources MZ...
for %%D in (
  "resources\[mz]\mz_notify"
  "resources\[mz]\mz_sync"
  "resources\[mz]\mz_core"
  "resources\[mz]\mz_economy"
  "resources\[mz]\mz_assets"
  "resources\[mz]\mz_inventory"
  "resources\[mz]\mz_status"
  "resources\[mz]\mz_progress"
  "resources\[mz]\mz_medical"
  "resources\[mz]\mz_bank"
  "resources\[mz]\mz_settings"
  "resources\[mz]\mz_menu"
  "resources\[mz]\mz_interact"
  "resources\[mz]\mz_world_editor"
  "resources\[mz]\mz_phone"
  "resources\[mz]\mz_transport"
  "resources\[mz]\mz_org"
  "resources\[mz]\mz_police"
  "resources\[mz]\mz_prison"
  "resources\[mz]\mz_dispatch"
  "resources\[mz]\mz_interiors"
  "resources\[mz]\mz_houses"
  "resources\[mz]\mz_furniture"
  "resources\[mz]\mz_org_activities"
  "resources\[mz]\mz_computer"
  "resources\[mz]\mz_vehicles"
  "resources\[mz]\mz_target"
  "resources\[mz]\mz_radio"
  "resources\[mz]\mz_creator"
  "resources\[mz]\mz_multichar"
  "resources\[mz]\mz_clothing"
  "resources\[mz]\mz_fuel"
  "resources\[mz]\mz_garagem"
  "resources\[mz]\mz_hud"
  "resources\[mz]\mz_banguard"
  "resources\[mz]\mz_warns"
  "resources\[mz]\mz_admin"
  "resources\[mz]\mz_feedback"
) do (
  if exist "%%~D\fxmanifest.lua" (
    echo [OK] %%~D
  ) else (
    echo [MISSING] %%~D\fxmanifest.lua
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
