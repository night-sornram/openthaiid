#Requires -RunAsAdministrator
# Install OpenThaiID as a Windows service (always-on, no tray).
# Clinic IT: run this instead of relying on a logged-in Start Menu tray.
#
#   powershell -ExecutionPolicy Bypass -File Install-OpenThaiID-Service.ps1
#
# The helper listens only on 127.0.0.1. Ready page: http://127.0.0.1:17891/
# Technician tools: http://127.0.0.1:17891/#technician
# No citizen ID, names, address, or portrait are stored.
param()

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$exe = Join-Path $here "openthaiid.exe"
if (-not (Test-Path $exe)) {
    $exe = Join-Path $env:LOCALAPPDATA "OpenThaiID\openthaiid.exe"
}
if (-not (Test-Path $exe)) {
    throw "openthaiid.exe not found next to this script or in %LOCALAPPDATA%\OpenThaiID"
}

& $exe service install
if ($LASTEXITCODE -ne 0) { throw "openthaiid service install failed" }
Write-Host "OpenThaiID Windows service is installed and starting."
Write-Host "There is no tray icon (Session 0). Open http://127.0.0.1:17891/ after sign-in."
Write-Host "Remove with: openthaiid.exe service uninstall   (or Uninstall-OpenThaiID-Service.ps1)"
