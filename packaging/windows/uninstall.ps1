# Remove current-user Start Menu / Desktop shortcuts, login item, and
# %LOCALAPPDATA%\OpenThaiID. Does not require admin.
$ErrorActionPreference = "Stop"
$exe = Join-Path $env:LOCALAPPDATA "OpenThaiID\openthaiid.exe"
if (Test-Path $exe) {
    & $exe quit 2>$null
    & $exe autostart disable 2>$null
    # User install does not remove a machine Windows service; that needs
    # Uninstall-OpenThaiID-Service.ps1 (Administrator).
}
$programs = Join-Path ([Environment]::GetFolderPath("StartMenu")) "Programs\OpenThaiID.lnk"
$desk = Join-Path ([Environment]::GetFolderPath("Desktop")) "OpenThaiID.lnk"
foreach ($p in @($programs, $desk)) {
    if (Test-Path $p) { Remove-Item -Force $p }
}
$dest = Join-Path $env:LOCALAPPDATA "OpenThaiID"
if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
Write-Host "OpenThaiID helper removed for this user."
