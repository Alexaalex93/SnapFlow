# SnapFlow release build script
# Usage (from repo root):
#   .\scripts\build-release.ps1              # builds exe + installer
#   .\scripts\build-release.ps1 -ExeOnly     # builds exe only (no Inno Setup needed)
#   .\scripts\build-release.ps1 -Version 1.2.3

param(
    [string]$Version = "1.0.0",
    [switch]$ExeOnly
)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path "$PSScriptRoot\..").Path
Set-Location $Root

# ── 1. Generate icon (if it doesn't exist or is outdated) ────────────────────
if (-not (Test-Path "assets\icon.ico")) {
    Write-Host "Generating icon..." -ForegroundColor Cyan
    & "$PSScriptRoot\make-icon.ps1"
}

# ── 2. Install goversioninfo if needed ───────────────────────────────────────
$gvPath = Join-Path $env:GOPATH "bin\goversioninfo.exe"
if (-not (Test-Path $gvPath)) {
    Write-Host "Installing goversioninfo..." -ForegroundColor Cyan
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo install failed" }
}

# ── 3. Patch versioninfo.json with the current version ───────────────────────
$vi = Get-Content versioninfo.json -Raw | ConvertFrom-Json
$parts = $Version -split '\.'
$maj = [int]($parts[0] ?? 1)
$min = [int]($parts[1] ?? 0)
$pat = [int]($parts[2] ?? 0)

$vi.FixedFileInfo.FileVersion.Major    = $maj
$vi.FixedFileInfo.FileVersion.Minor    = $min
$vi.FixedFileInfo.FileVersion.Patch    = $pat
$vi.FixedFileInfo.ProductVersion.Major = $maj
$vi.FixedFileInfo.ProductVersion.Minor = $min
$vi.FixedFileInfo.ProductVersion.Patch = $pat
$vi.StringFileInfo.FileVersion         = "$Version.0"
$vi.StringFileInfo.ProductVersion      = "$Version.0"
$vi | ConvertTo-Json -Depth 10 | Set-Content versioninfo.json -Encoding utf8

# ── 4. go generate (creates resource.syso) ───────────────────────────────────
Write-Host "Running go generate..." -ForegroundColor Cyan
go generate
if ($LASTEXITCODE -ne 0) { throw "go generate failed" }

# ── 5. Build the exe ─────────────────────────────────────────────────────────
$dist = "dist"
if (-not (Test-Path $dist)) { New-Item -ItemType Directory $dist | Out-Null }

$exe = "$dist\snapflow.exe"
Write-Host "Building $exe  (v$Version)..." -ForegroundColor Cyan
go build -ldflags "-H=windowsgui -s -w -X main.appVersion=$Version" -o $exe .
if ($LASTEXITCODE -ne 0) { throw "go build failed" }
Write-Host "  Built: $exe  ($([int]((Get-Item $exe).Length/1KB)) KB)" -ForegroundColor Green

if ($ExeOnly) { Write-Host "Done (exe only)." -ForegroundColor Green; exit 0 }

# ── 6. Create installer with Inno Setup ──────────────────────────────────────
# Try common install locations, then PATH
$iscc = @(
    "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
    "${env:ProgramFiles}\Inno Setup 6\ISCC.exe",
    "iscc"
) | Where-Object { Test-Path $_ -ErrorAction SilentlyContinue } | Select-Object -First 1

if (-not $iscc) {
    Write-Host "Inno Setup not found — installing via winget..." -ForegroundColor Yellow
    winget install --id JRSoftware.InnoSetup --silent --accept-source-agreements --accept-package-agreements
    if ($LASTEXITCODE -ne 0) { throw "Inno Setup install failed. Install manually from https://jrsoftware.org/isdl.php" }
    $iscc = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
}

# Copy exe to root so installer can find it
Copy-Item $exe snapflow.exe -Force

Write-Host "Creating installer..." -ForegroundColor Cyan
& $iscc "scripts\installer.iss" "/DAppVersion=$Version" | Tee-Object -Variable isccOut
$ok = $LASTEXITCODE -eq 0

Remove-Item snapflow.exe -Force -ErrorAction SilentlyContinue
if (-not $ok) { throw "ISCC failed" }

$installer = Get-ChildItem dist\SnapFlow-Setup*.exe | Sort-Object LastWriteTime -Descending | Select-Object -First 1
Write-Host "Installer: $($installer.FullName)  ($([int]($installer.Length/1KB)) KB)" -ForegroundColor Green
Write-Host "Done." -ForegroundColor Green
