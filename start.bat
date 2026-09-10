@echo off
chcp 65001 >nul
title Hermes Storage Monitor
echo ============================================
echo  Hermes Storage Monitor — Start All
echo ============================================
echo.

set "ROOT=C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar"

echo [1/2] Starting backend...
cd /d "%ROOT%"
start "Hermes Backend" /min cmd /c "cd /d "%ROOT%" && backend\hermes-backend.exe"
echo    Backend starting on port 3001...

timeout /t 3 /nobreak >nul

echo [2/2] Starting dashboard...
cd /d "%ROOT%\dashboard"
start "Hermes Dashboard" /min cmd /c "cd /d "%ROOT%\dashboard" && npm run dev"
echo    Dashboard starting on port 3000...

echo.
echo ============================================
echo  Wait ~10 seconds then open:
echo  http://localhost:3000
echo ============================================
echo.
start http://localhost:3000
