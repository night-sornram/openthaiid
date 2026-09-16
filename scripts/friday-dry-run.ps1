# Friday morning software dry-run (synthetic). No ACR39U required.
# Never prints citizen ID, names, address, or photo bytes.
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts/friday-dry-run.ps1
#   powershell -ExecutionPolicy Bypass -File scripts/friday-dry-run.ps1 -TwoReaders
param(
    [switch]$TwoReaders,
    [int]$Port = 17981,
    [switch]$SkipQualify,
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

$Forbidden = @(
    "1234567890121", "1101701234567",
    "สมชาย", "ใจดี", "Somchai", "Jaidee",
    "เทศบาลนครนิทาน", "ถนนนิทาน", "ตำบลบ้านใหม่", "อำเภอเมือง", "จังหวัดนิทาน",
    "25330412", "25600115", "25700114",
    "FICTIONAL-OPENTHAIID-PORTRAIT",
    "/9j/", "data:image", "image/jpeg;base64,", "iVBORw0KGgo", "FFD8FF"
)

function Assert-NoPii {
    param([string]$Text, [string]$Label)
    foreach ($needle in $Forbidden) {
        if ($Text.Contains($needle)) {
            throw "ERROR: $Label leaked personal or photo data; refusing to print"
        }
    }
    if ($Text -match '(?<![0-9])[0-9]{13}(?![0-9])') {
        throw "ERROR: $Label leaked personal or photo data; refusing to print"
    }
}

function Get-Json {
    param([string]$Path)
    $raw = Get-Content -Raw -Encoding UTF8 -Path $Path
    Assert-NoPii -Text $raw -Label $Path
    return $raw | ConvertFrom-Json
}

Write-Host "== OpenThaiID Friday dry-run (synthetic; not a hardware pass) =="
Write-Host "docs: getting-started.md · training.md · friday-checklist.md"
Write-Host "never print citizen ID, names, address, or photo bytes"

if (-not (Test-Path $Bin)) {
    Write-Host "== build helper =="
    Push-Location $Root
    try {
        make build
    } finally {
        Pop-Location
    }
}
if (-not (Test-Path $Bin)) {
    throw "missing $Bin (run make build)"
}

$ver = & $Bin version
Assert-NoPii -Text $ver -Label "version"
Write-Host "version: $ver"

if (-not $SkipQualify) {
    Write-Host "== qualify-smoke (synthetic kit; not ACR39U) =="
    $qualify = Join-Path $Root "scripts\hardware-qualify.ps1"
    $report = Join-Path $env:TEMP ("openthaiid-qualify-dryrun-" + [guid]::NewGuid().ToString() + ".json")
    & $qualify -Synthetic -E2 3 -E3 2 -Report $report -Bin $Bin
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    if (Test-Path $report) { Remove-Item -Force $report }
}

$HomeDir = Join-Path $env:TEMP ("openthaiid-friday-dry-run-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $HomeDir | Out-Null
$env:OPENTHAIID_HOME = $HomeDir
$env:OPENTHAIID_LANG = "th"
if ($TwoReaders) {
    $env:OPENTHAIID_SYNTHETIC_READERS = "2"
    Write-Host "== helper -synthetic (two practice readers) port=$Port =="
} else {
    Remove-Item Env:OPENTHAIID_SYNTHETIC_READERS -ErrorAction SilentlyContinue
    Write-Host "== helper -synthetic port=$Port =="
}

$Ready = "http://127.0.0.1:$Port/"
$helper = $null
function Stop-Helper {
    try { & $Bin quit -port $Port | Out-Null } catch {}
    if ($helper -and -not $helper.HasExited) {
        try { $helper.Kill() } catch {}
    }
}
Register-EngineEvent PowerShell.Exiting -Action { Stop-Helper } | Out-Null

$outLog = Join-Path $HomeDir "helper.out"
$errLog = Join-Path $HomeDir "helper.err"
$helper = Start-Process -FilePath $Bin -ArgumentList @("-synthetic", "-port", "$Port", "-open=false") -PassThru -WindowStyle Hidden -RedirectStandardOutput $outLog -RedirectStandardError $errLog

$up = $false
for ($i = 0; $i -lt 80; $i++) {
    try {
        $null = Invoke-WebRequest -UseBasicParsing -TimeoutSec 1 -Uri ($Ready + "healthz")
        $up = $true
        break
    } catch {
        if ($helper.HasExited) { break }
        Start-Sleep -Milliseconds 100
    }
}
if (-not $up) {
    Stop-Helper
    throw "helper did not start on $Ready"
}

Write-Host "== status + Ready + summary =="
$status = & $Bin status -port $Port | Out-String
Assert-NoPii -Text $status -Label "status"
if ($TwoReaders) {
    if ($status -notmatch "ยังไม่พร้อม|Not ready") { throw "status: expected Not ready until reader pick" }
} else {
    if ($status -notmatch "บัตรฝึกซ้อม|practice card") { throw "status: expected practice Ready" }
}
Write-Host "OK status (no personal fields)"

$readyPath = Join-Path $HomeDir "ready.html"
Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri $Ready -OutFile $readyPath
$readyHtml = Get-Content -Raw -Encoding UTF8 -Path $readyPath
Assert-NoPii -Text $readyHtml -Label "Ready page"
if ($readyHtml -notmatch 'id="test-connection"') { throw "Ready page missing test-connection" }
if ($readyHtml -notmatch "ทดสอบการเชื่อมต่อ") { throw "Ready page missing Thai test-connection" }
if ($readyHtml -notmatch "ฝึกใช้ก่อนได้เครื่องจริง") { throw "Ready page missing training copy" }
Write-Host "OK Ready page (test-connection present; no personal fields)"

$sumPath = Join-Path $HomeDir "summary.json"
Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri ($Ready + "v1/helper/summary") -OutFile $sumPath
$sum = Get-Json $sumPath
if ($sum.helper -ne $true) { throw "summary: helper is not up" }
if ($sum.practice -ne $true) { throw "summary: expected practice/synthetic" }
if ($sum.reader -ne $true) { throw "summary: expected a practice reader" }
if ($TwoReaders) {
    if ($sum.card -eq $true) { throw "summary: two readers must not look Ready until picked" }
} else {
    if ($sum.card -ne $true) { throw "summary: expected practice card present" }
    if ("$($sum.headline)" -notmatch "พร้อม|Ready") { throw "summary: expected Ready (practice)" }
}
Write-Host "OK helper-summary practice=true"

Write-Host "== connection test (CSRF manager status; WS probe if python is available) =="
if ($readyHtml -notmatch 'name="openthaiid-csrf"\s+content="([a-f0-9]+)"') {
    throw "Ready page missing CSRF token"
}
$token = $Matches[1]
$headers = @{ "X-OpenThaiID-Manager" = $token }
$stPath = Join-Path $HomeDir "manager-status.json"
Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri ($Ready + "v1/manager/status") -Headers $headers -OutFile $stPath
$mgr = Get-Json $stPath
if ($mgr.running -ne $true) { throw "manager status: helper not running" }
if ($mgr.synthetic -ne $true) { throw "manager status: expected synthetic" }
if ($mgr.backend.status -ne "available") { throw "manager status: backend not available" }
if (-not $mgr.readers) { throw "manager status: no readers" }
Write-Host "OK manager-status probe-only (WS bridge.probe is scripts/friday-dry-run.sh)"

if ($TwoReaders) {
    Write-Host "== pick practice reader (ใช้เครื่องนี้) =="
    $selectHeaders = @{
        "X-OpenThaiID-Manager" = $token
        "Origin" = "http://127.0.0.1:$Port"
        "Content-Type" = "application/json"
    }
    $selectPath = Join-Path $HomeDir "select.json"
    Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Method POST -Uri ($Ready + "v1/manager/reader/select") -Headers $selectHeaders -Body '{"readerId":"synthetic-acr39u"}' -OutFile $selectPath
    Assert-NoPii -Text (Get-Content -Raw -Encoding UTF8 -Path $selectPath) -Label "select"
    Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri ($Ready + "v1/helper/summary") -OutFile $sumPath
    $sum = Get-Json $sumPath
    if ($sum.card -ne $true) { throw "summary after pick: expected practice card present" }
    Write-Host "OK reader selected → Ready (practice)"
}

Stop-Helper

Write-Host ""
Write-Host "======== ผ่าน / PASS — ซ้อมก่อนได้เครื่อง ========"
Write-Host "ซอฟต์แวร์เช้าวันศุกร์พร้อมบนโหมดฝึกซ้อม (ยังไม่ใช่ผ่าน ACR39U)"
Write-Host "Friday-morning software path rehearsed on -synthetic (not a hardware pass)."
Write-Host ""
Write-Host "หน้า พร้อม / Ready: $Ready"
Write-Host "ดัชนีเช้านี้: docs/getting-started.md"
Write-Host "ฝึกกรอกข้อมูลต่อ: docs/training.md"
Write-Host "เช้าที่มีเครื่องจริง: docs/friday-checklist.md  (เปิดตัวช่วยโดยไม่ใส่ -synthetic)"
Write-Host ""
Write-Host "ห้ามส่งเลขบัตร ชื่อ ที่อยู่ วันเกิด รูปถ่าย ภาพหน้าจอฟอร์ม หรือ HAR ที่มียอด card.read"
Write-Host "Do not send citizen ID, names, address, date of birth, photos, filled-form"
Write-Host "screenshots, or HARs that contain card.read payloads."
