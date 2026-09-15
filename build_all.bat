@echo off
setlocal
echo ========================================================
echo   MyLokalWebserver - Multi-Platform and App Bundler
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
copy /Y dist\windows\mylokalwebserver.exe mylokalwebserver.exe >nul 2>&1
copy /Y start.bat dist\windows\ >nul 2>&1

echo [2/4] Compiling macOS Apple Silicon (M1/M2/M3/M4 - ARM64)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\macos-arm64\mylokalwebserver .
copy /Y start.sh dist\macos-arm64\ >nul 2>&1

:: Create macOS .app bundle for ARM64
set "APP_ARM64=dist\macos-arm64\MyLokalWebserver.app"
if not exist "%APP_ARM64%\Contents\MacOS" mkdir "%APP_ARM64%\Contents\MacOS"
if not exist "%APP_ARM64%\Contents\Resources" mkdir "%APP_ARM64%\Contents\Resources"
copy /Y dist\macos-arm64\mylokalwebserver "%APP_ARM64%\Contents\MacOS\mylokalwebserver" >nul 2>&1
copy /Y start.sh "%APP_ARM64%\Contents\MacOS\launcher" >nul 2>&1

echo ^<?xml version="1.0" encoding="UTF-8"?^> > "%APP_ARM64%\Contents\Info.plist"
echo ^<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"^> >> "%APP_ARM64%\Contents\Info.plist"
echo ^<plist version="1.0"^> >> "%APP_ARM64%\Contents\Info.plist"
echo ^<dict^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleExecutable^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>launcher^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleIdentifier^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>com.fery.mylokalwebserver^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleName^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>MyLokalWebserver^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleDisplayName^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>MyLokalWebserver^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundlePackageType^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>APPL^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleShortVersionString^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>1.0.0^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>CFBundleVersion^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>1^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>LSMinimumSystemVersion^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<string^>10.15^</string^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>LSUIElement^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<false/^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<key^>NSHighResolutionCapable^</key^> >> "%APP_ARM64%\Contents\Info.plist"
echo     ^<true/^> >> "%APP_ARM64%\Contents\Info.plist"
echo ^</dict^> >> "%APP_ARM64%\Contents\Info.plist"
echo ^</plist^> >> "%APP_ARM64%\Contents\Info.plist"

echo|set /p="APPL????" > "%APP_ARM64%\Contents\PkgInfo"

echo [3/4] Compiling macOS Intel (AMD64)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\macos-intel\mylokalwebserver .
copy /Y start.sh dist\macos-intel\ >nul 2>&1

:: Create macOS .app bundle for Intel
set "APP_INTEL=dist\macos-intel\MyLokalWebserver.app"
if not exist "%APP_INTEL%\Contents\MacOS" mkdir "%APP_INTEL%\Contents\MacOS"
if not exist "%APP_INTEL%\Contents\Resources" mkdir "%APP_INTEL%\Contents\Resources"
copy /Y dist\macos-intel\mylokalwebserver "%APP_INTEL%\Contents\MacOS\mylokalwebserver" >nul 2>&1
copy /Y start.sh "%APP_INTEL%\Contents\MacOS\launcher" >nul 2>&1
copy /Y "%APP_ARM64%\Contents\Info.plist" "%APP_INTEL%\Contents\Info.plist" >nul 2>&1
copy /Y "%APP_ARM64%\Contents\PkgInfo" "%APP_INTEL%\Contents\PkgInfo" >nul 2>&1

echo [4/4] Compiling Linux x64 (AMD64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\linux-amd64\mylokalwebserver .
copy /Y start.sh dist\linux-amd64\ >nul 2>&1

echo.
echo ========================================================
echo   [SUCCESS] Seluruh package and app bundle berhasil dibuat!
echo ========================================================
echo   - Windows 64-bit     : dist\windows\mylokalwebserver.exe
echo   - macOS M1/M2/M3/M4  : dist\macos-arm64\MyLokalWebserver.app
echo   - macOS Intel        : dist\macos-intel\MyLokalWebserver.app
echo   - Linux 64-bit       : dist\linux-amd64\mylokalwebserver
echo ========================================================
echo.
