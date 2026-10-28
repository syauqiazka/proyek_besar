@echo off
chcp 65001 >nul
title Register Server Baru — Hermes
echo ============================================
echo  Register Server Baru ke Hermes Backend
echo ============================================
echo.

set "ROOT=C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar"

:: Cek backend jalan
curl -s -m 3 http://localhost:3001/health >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Backend belum jalan!
    echo         Jalankan start.bat dulu.
    echo.
    pause
    exit /b 1
)

:: Input nama server
set /p "HOSTNAME=Masukkan nama server (misal: web-01): "
if "%HOSTNAME%"=="" (
    echo [ERROR] Nama server tidak boleh kosong!
    pause
    exit /b 1
)

:: Input IP backend (default: localhost)
set /p "BACKEND_IP=IP backend (Enter untuk localhost:3001): "
if "%BACKEND_IP%"=="" set "BACKEND_IP=localhost:3001"

:: Register ke backend
echo.
echo [1/3] Register "%HOSTNAME%" ke backend...

set RESP=
for /f "delims=" %%A in ('curl -s -X POST http://localhost:3001/api/v1/agent/register -H "Content-Type: application/json" -d "{\"hostname\":\"%HOSTNAME%\",\"tenant_id\":\"b39f9091-eb31-4c18-b77c-d08ced5c8fb1\"}"') do set RESP=%%A

:: Cek response
echo %RESP% | findstr "server_id" >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Register gagal!
    echo Response: %RESP%
    pause
    exit /b 1
)

:: Generate .env via python
echo [2/3] Generate file konfigurasi (.env & hermes-agent.env)...

if not exist "%ROOT%\deploy\%HOSTNAME%" mkdir "%ROOT%\deploy\%HOSTNAME%"

echo %RESP% | python -c "import json, sys; r=json.load(sys.stdin); c=f'HERMES_SERVER_ID={r[\"server_id\"]}\nHERMES_API_KEY={r[\"api_key\"]}\nHERMES_BACKEND_URL=http://%BACKEND_IP%/api/v1\nHERMES_SCAN_INTERVAL=*/5 * * * *\nHERMES_MOUNT_PATHS=/,/home\n'; open(r'%ROOT%\deploy\%HOSTNAME%\.env','w').write(c); open(r'%ROOT%\deploy\%HOSTNAME%\hermes-agent.env','w').write(c); print(f'  server_id: {r[\"server_id\"]}'); print(f'  api_key:   {r[\"api_key\"][:20]}...')" 2>nul

:: Copy binary & background scripts
echo [3/3] Menyiapkan paket deploy...
if exist "%ROOT%\agent\hermes-agent" (
    copy "%ROOT%\agent\hermes-agent" "%ROOT%\deploy\%HOSTNAME%\hermes-agent" >nul
    echo   Binary agent berhasil disalin.
) else (
    echo   [WARNING] Binary agent belum ada. Build dulu:
    echo   cd agent ^&^& GOOS=linux go build -o hermes-agent cmd/agent\main.go
)

if exist "%ROOT%\deploy\linux\start-agent.sh" (
    copy "%ROOT%\deploy\linux\start-agent.sh" "%ROOT%\deploy\%HOSTNAME%\" >nul
    copy "%ROOT%\deploy\linux\stop-agent.sh" "%ROOT%\deploy\%HOSTNAME%\" >nul
    copy "%ROOT%\deploy\linux\status-agent.sh" "%ROOT%\deploy\%HOSTNAME%\" >nul
    copy "%ROOT%\deploy\linux\install-agent-service.sh" "%ROOT%\deploy\%HOSTNAME%\" >nul
    copy "%ROOT%\deploy\linux\hermes-agent.service" "%ROOT%\deploy\%HOSTNAME%\" >nul
    echo   Skrip background Linux berhasil disalin.
)

:: Tampilkan instruksi deploy
echo.
echo ============================================
echo  SERVER "%HOSTNAME%" BERHASIL DIDAFTARKAN!
echo ============================================
echo.
echo  File siap deploy di:
echo    deploy\%HOSTNAME%\
echo.
echo  Kirim ke server Linux:
echo    scp deploy\%HOSTNAME%\* user@IP-SERVER:~/~
echo.
echo  Di server Linux:
echo    chmod +x ~/hermes-agent
echo    ./hermes-agent
echo.
echo  Cek dashboard: http://localhost:3000
echo ============================================
echo.
pause
