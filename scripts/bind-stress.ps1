# PC/SC / ebfe/scard binding-audit stress wrapper (Windows).
# Never prints or stores citizen ID, names, address, portrait bytes, or raw APDUs.
param(
    [switch]$Synthetic,
    [int]$Cycles = 20,
    [string]$Report = "",
    [string]$Bin = ""
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
if (-not $Bin) {
    $Bin = Join-Path $Root "dist\openthaiid.exe"
    if (-not (Test-Path $Bin)) {
        $Bin = Join-Path $Root "dist\openthaiid"
    }
}
if (-not $Report) {
    $Report = Join-Path $Root "dist\openthaiid-bind-stress.json"
}

if (-not (Test-Path $Bin)) {
    Write-Error "missing $Bin (run make build)"
}

$cmdArgs = @("bind-stress", "-cycles", "$Cycles", "-report", $Report)
if ($Synthetic) {
    $cmdArgs += "-synthetic"
    Write-Host "OpenThaiID bind-stress: SYNTHETIC harness smoke (does not close P2-P10)"
} else {
    Write-Host "OpenThaiID bind-stress: hardware PC/SC (no personal fields, no APDUs)"
}

Write-Host "report will be: $Report"
& $Bin @cmdArgs
$code = $LASTEXITCODE
if (Test-Path $Report) {
    $blob = Get-Content -Raw -Path $Report
    if ($blob -match '1234567890121|data:image|/9j/|Somchai|00A4040008A000000054480001') {
        Remove-Item -Force $Report
        Write-Error "ERROR: report looks like it contains personal data or an APDU dump; deleting"
    }
    Write-Host "report: $Report"
}
exit $code
