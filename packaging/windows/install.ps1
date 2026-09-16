# Install OpenThaiID helper for the current Windows user (no admin).
# Usage:
#   powershell -ExecutionPolicy Bypass -File install.ps1
#   powershell -ExecutionPolicy Bypass -File install.ps1 -Autostart
#   powershell -ExecutionPolicy Bypass -File install.ps1 -DesktopShortcut
param(
    [switch]$Autostart,
    [switch]$DesktopShortcut
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$exeSrc = Join-Path $here "openthaiid.exe"
if (-not (Test-Path $exeSrc)) {
    throw "openthaiid.exe not found next to install.ps1"
}

$dest = Join-Path $env:LOCALAPPDATA "OpenThaiID"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item -Force $exeSrc (Join-Path $dest "openthaiid.exe")
$bat = Join-Path $here "Start-OpenThaiID.bat"
if (Test-Path $bat) {
    Copy-Item -Force $bat (Join-Path $dest "Start-OpenThaiID.bat")
}

$exe = Join-Path $dest "openthaiid.exe"
$programs = Join-Path ([Environment]::GetFolderPath("StartMenu")) "Programs"
New-Item -ItemType Directory -Force -Path $programs | Out-Null
$ws = New-Object -ComObject WScript.Shell
$lnk = $ws.CreateShortcut((Join-Path $programs "OpenThaiID.lnk"))
$lnk.TargetPath = $exe
$lnk.Arguments = "tray -open"
$lnk.WorkingDirectory = $dest
$lnk.WindowStyle = 7
$lnk.Description = "OpenThaiID local helper (loopback only)"
$lnk.Save()

if ($DesktopShortcut) {
    $desk = [Environment]::GetFolderPath("Desktop")
    $dlnk = $ws.CreateShortcut((Join-Path $desk "OpenThaiID.lnk"))
    $dlnk.TargetPath = $exe
    $dlnk.Arguments = "tray -open"
    $dlnk.WorkingDirectory = $dest
    $dlnk.WindowStyle = 7
    $dlnk.Description = "OpenThaiID local helper (loopback only)"
    $dlnk.Save()
}

if ($Autostart) {
    & $exe autostart enable
    if ($LASTEXITCODE -ne 0) { throw "autostart enable failed" }
}

Write-Host "Installed to $dest"
Write-Host "Start Menu: OpenThaiID (starts the tray helper and Ready page)."
Write-Host "Autostart is off unless you passed -Autostart."
Write-Host "The helper listens only on 127.0.0.1."
