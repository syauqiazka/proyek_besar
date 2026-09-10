@echo off
chcp 65001 >nul
echo ============================================
echo  Hermes Storage Monitor — Stop All
echo ============================================
echo.

echo Stopping hermes-backend.exe...
taskkill /F /IM hermes-backend.exe 2>nul
if %ERRORLEVEL% EQU 0 (echo   Backend stopped.) else (echo   Backend not running.)

echo Stopping Next.js dashboard...
taskkill /F /IM node.exe 2>nul
if %ERRORLEVEL% EQU 0 (echo   Dashboard stopped.) else (echo   Dashboard not running.)

echo.
echo ============================================
echo  All services stopped.
echo ============================================
echo.
pause