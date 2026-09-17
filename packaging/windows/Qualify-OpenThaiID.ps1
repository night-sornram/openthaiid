# Hardware qualification (Windows).
# Never prints or stores citizen ID, names, address, or portrait bytes.
# Usage:
#   powershell -ExecutionPolicy Bypass -File Qualify-OpenThaiID.ps1
#   powershell -ExecutionPolicy Bypass -File Qualify-OpenThaiID.ps1 -Operator
#   powershell -ExecutionPolicy Bypass -File Qualify-OpenThaiID.ps1 -Synthetic -E2 3 -E3 2
param(
    [switch]$Synthetic,
    [switch]$Operator,
    [int]$E2 = 100,
    [int]$E3 = 20,
    [string]$Report = "",
    [string]$Bin = ""
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not $Bin) {
    if (Test-Path (Join-Path $Root "openthaiid.exe")) {
        $Bin = Join-Path $Root "openthaiid.exe"
    } else {
        $Bin = Join-Path $Root "..\..\dist\openthaiid.exe"
    }
}
if (-not (Test-Path $Bin)) {
    Write-Error "missing $Bin (unzip the Windows clinic package, or make build)"
}

if (-not $Report) {
    $Report = Join-Path $Root "openthaiid-qualify.json"
}
$Report = [System.IO.Path]::GetFullPath($Report)

$qualifyArgs = @("qualify", "-e2-count", "$E2", "-e3-count", "$E3", "-report", $Report)
if ($Synthetic) {
    $qualifyArgs += "-synthetic"
    Write-Host "OpenThaiID qualify: SYNTHETIC kit smoke (not a hardware pass)"
} else {
    Write-Host "OpenThaiID qualify: hardware (no personal fields in output)"
    Write-Host "Platform order: macOS + ACR39U first, then Windows, then Linux."
}
if ($Operator) {
    $qualifyArgs += "-operator"
    Write-Host "Operator mode: type pass, fail, or skip only. Do not type card data."
}

Write-Host "See docs\hardware-checklist.md (one page) and docs\hardware-qualification.md."
Write-Host "report will be: $Report"

& $Bin @qualifyArgs
$code = $LASTEXITCODE

if (Test-Path $Report) {
    $blob = Get-Content -Raw -Path $Report
    foreach ($needle in @("1234567890121", "data:image", "/9j/", "สมชาย", "Somchai")) {
        if ($blob.Contains($needle)) {
            Write-Error "report looks like it contains personal or photo data; deleting"
            Remove-Item -Force $Report
            exit 1
        }
    }
    Write-Host "report: $Report"
}

exit $code
