@echo off
title MyLokalWebserver Launcher
cd /d "%~dp0"

if not exist "mylokalwebserver.exe" (
    echo Mengompilasi mylokalwebserver.exe terlebih dahulu...
    go build -ldflags="-s -w" -o mylokalwebserver.exe .
)

start "" mylokalwebserver.exe
