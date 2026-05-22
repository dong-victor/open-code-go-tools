@echo off
REM build.bat - Build ocgt with automatic version injection (Windows)
REM Usage: build.bat [version]
REM   If version is not provided, it will be read from wails.json

setlocal enabledelayedexpansion

cd /d "%~dp0"

REM Get version from argument or wails.json
if not "%~1"=="" (
    set VERSION=%~1
) else (
    REM Parse version from wails.json (simple approach)
    for /f "tokens=2 delims=:," %%a in ('findstr "productVersion" wails.json') do (
        set VER_RAW=%%a
    )
    REM Remove quotes and spaces
    set VERSION=!VER_RAW:"=!
    set VERSION=!VERSION: =!
)

if "!VERSION!"=="" (
    echo Error: Could not determine version
    exit /b 1
)

echo Building ocgt version: !VERSION!

REM Build with version injection via ldflags
set LDFLAGS=-X github.com/ethan-blue/open-code-go-tools/internal/version.Version=!VERSION!

echo Building with ldflags: !LDFLAGS!

where wails >nul 2>nul
if errorlevel 1 (
    echo Wails CLI not found in PATH, using go run fallback...
    go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -ldflags "!LDFLAGS!"
) else (
    wails build -ldflags "!LDFLAGS!"
)

if errorlevel 1 (
    echo Build failed
    exit /b 1
)

echo.
echo Build complete!
echo Output: build\bin\ocgt_v!VERSION!.exe

endlocal
