# Fallback always-on helper when a Windows service is not used.
# Prefer: Install-OpenThaiID-Service.ps1  (openthaiid service install)
#
# Default (no admin): At logon for the current user — same idea as autostart,
# still needs a signed-in session (no tray if the task runs hidden).
#
# Admin "run whether user is logged on or not":
#   powershell -ExecutionPolicy Bypass -File Register-OpenThaiID-Task.ps1 -AtStartup
#
# The helper still listens only on 127.0.0.1. No card data is stored.
param(
    [switch]$AtStartup,
    [switch]$Unregister
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$exe = Join-Path $here "openthaiid.exe"
if (-not (Test-Path $exe)) {
    $exe = Join-Path $env:LOCALAPPDATA "OpenThaiID\openthaiid.exe"
}
if (-not (Test-Path $exe)) {
    throw "openthaiid.exe not found"
}

$taskName = "OpenThaiID-Helper"
if ($Unregister) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Write-Host "Scheduled task $taskName removed (if it existed)."
    exit 0
}

$arg = "serve"
$action = New-ScheduledTaskAction -Execute $exe -Argument $arg
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
if ($AtStartup) {
    $trigger = New-ScheduledTaskTrigger -AtStartup
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -RunLevel Highest -Force | Out-Null
    Write-Host "Registered $taskName at startup (admin). Runs openthaiid serve (no tray)."
    Write-Host "This still does not store citizen ID, names, address, or portrait."
} else {
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null
    Write-Host "Registered $taskName at logon for this user. Runs openthaiid serve (no tray)."
    Write-Host "For a true no-login helper, use Install-OpenThaiID-Service.ps1 (Administrator)."
}
Write-Host "Ready page after the helper is up: http://127.0.0.1:17891/"
