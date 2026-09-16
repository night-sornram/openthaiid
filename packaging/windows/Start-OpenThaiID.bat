@echo off
setlocal
cd /d "%~dp0"
if not exist "%~dp0openthaiid.exe" (
  echo openthaiid.exe is missing from this folder.
  pause
  exit /b 1
)
start "" "%~dp0openthaiid.exe" tray -open
