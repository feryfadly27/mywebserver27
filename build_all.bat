@echo off
setlocal
echo ========================================================
echo   ⚡ MyLokalWebserver - Multi-Platform Build Script
echo ========================================================
echo.

if not exist "dist" mkdir dist
if not exist "dist\windows" mkdir dist\windows
if not exist "dist\macos-arm64" mkdir dist\macos-arm64
if not exist "dist\macos-intel" mkdir dist\macos-intel
if not exist "dist\linux-amd64" mkdir dist\linux-amd64

echo [1/4] Compiling Windows x64 (mylokalwebserver.exe)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\windows\mylokalwebserver.exe .
copy /Y mylokalwebserver.exe dist\windows\ >nul 2>&1
copy /Y start.bat dist\windows\ >nul 2>&1

echo [2/4] Compiling macOS Apple Silicon (M1/M2/M3/M4 - ARM64)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\macos-arm64\mylokalwebserver .
copy /Y start.sh dist\macos-arm64\ >nul 2>&1

echo [3/4] Compiling macOS Intel (AMD64)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\macos-intel\mylokalwebserver .
copy /Y start.sh dist\macos-intel\ >nul 2>&1

echo [4/4] Compiling Linux x64 (AMD64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\linux-amd64\mylokalwebserver .
copy /Y start.sh dist\linux-amd64\ >nul 2>&1

echo.
echo ========================================================
echo   [SUCCESS] Seluruh binary multi-platform telah dibuat!
echo ========================================================
echo   - Windows 64-bit     : dist\windows\mylokalwebserver.exe
echo   - macOS Apple Silicon: dist\macos-arm64\mylokalwebserver
echo   - macOS Intel        : dist\macos-intel\mylokalwebserver
echo   - Linux 64-bit       : dist\linux-amd64\mylokalwebserver
echo ========================================================
echo.
pause
