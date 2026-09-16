#Requires -RunAsAdministrator
# Remove the OpenThaiID Windows service installed by Install-OpenThaiID-Service.ps1.
param()

$ErrorActionPreference = "Stop"
$candidates = @(
    (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "openthaiid.exe"),
    (Join-Path $env:LOCALAPPDATA "OpenThaiID\openthaiid.exe")
)
$exe = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $exe) {
    throw "openthaiid.exe not found"
}
& $exe service uninstall
if ($LASTEXITCODE -ne 0) { throw "openthaiid service uninstall failed" }
Write-Host "OpenThaiID Windows service removed."
