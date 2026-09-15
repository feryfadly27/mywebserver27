@echo off
echo ========================================================
echo   Compiling MyLokalWebserver (Go Single Executable)
echo ========================================================
echo.

go build -ldflags="-s -w" -o mylokalwebserver.exe .

if %ERRORLEVEL% EQU 0 (
    echo [BERHASIL] mylokalwebserver.exe berhasil dibuat!
    echo Ukuran file:
    dir mylokalwebserver.exe | findstr "mylokalwebserver.exe"
    echo.
    echo Anda dapat langsung menjalankan mylokalwebserver.exe atau start.bat
) else (
    echo [GAGAL] Terjadi kesalahan saat kompilasi.
)
pause
