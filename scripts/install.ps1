# SnapFlow installer
# Run from the repo root: .\scripts\install.ps1

$ErrorActionPreference = "Stop"
$installDir = "$env:LOCALAPPDATA\SnapFlow"
$exe = "$installDir\snapflow.exe"

Write-Host "Building SnapFlow..."
Push-Location $PSScriptRoot\..
go build -ldflags="-H windowsgui -s -w" -o snapflow.exe .
if (-not $?) { Write-Error "Build failed"; exit 1 }
Pop-Location

Write-Host "Installing to $installDir"
if (-not (Test-Path $installDir)) { New-Item -ItemType Directory -Path $installDir | Out-Null }

# Stop any running instance
$running = Get-Process -Name snapflow -ErrorAction SilentlyContinue
if ($running) {
    Write-Host "Stopping running instance..."
    $running | Stop-Process -Force
    Start-Sleep -Milliseconds 500
}

Copy-Item "$PSScriptRoot\..\snapflow.exe" $exe -Force
Remove-Item "$PSScriptRoot\..\snapflow.exe" -Force

Write-Host "Installed: $exe"
Write-Host "Launching..."
Start-Process $exe
Write-Host "Done. SnapFlow is running in the system tray."
