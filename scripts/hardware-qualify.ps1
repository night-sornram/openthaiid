# Hardware qualification wrapper (Windows).
# Never prints or stores citizen ID, names, address, or portrait bytes.
param(
    [switch]$Synthetic,
    [switch]$Operator,
    [int]$E2 = 100,
    [int]$E3 = 20,
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
    $Report = Join-Path $Root "dist\openthaiid-qualify.json"
}

$here = Join-Path $Root "packaging\windows\Qualify-OpenThaiID.ps1"
& $here -Synthetic:$Synthetic -Operator:$Operator -E2 $E2 -E3 $E3 -Report $Report -Bin $Bin
exit $LASTEXITCODE
