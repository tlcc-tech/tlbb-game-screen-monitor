# Requires: Go 1.22+, Node.js 18+, Wails CLI v2
# Run this in PowerShell from the project root.

$ErrorActionPreference = "Stop"

& "$PSScriptRoot/fetch-dm.ps1"

Set-Location (Join-Path $PSScriptRoot ".." "frontend")
npm ci
npm run build
Set-Location (Join-Path $PSScriptRoot "..")

$Version = "dev"
$match = Select-String -Path "wails.json" -Pattern '"productVersion"\s*:\s*"([^"]+)"' | Select-Object -First 1
if ($match -and $match.Matches.Count -gt 0) {
	$Version = $match.Matches[0].Groups[1].Value
}

New-Item -ItemType Directory -Force -Path build/bin/dm | Out-Null
Copy-Item -Force third_party/dm/* build/bin/dm/

$env:GOOS = "windows"
$env:GOARCH = "386"
go build -o build/bin/dmfindpic.exe ./cmd/dmfindpic

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

wails build -platform windows/amd64 -clean -ldflags "-X main.AppVersion=$Version"
Copy-Item -Force "build/bin/tlbb-game-screen-monitor.exe" "build/bin/游戏掉线监控-windows-amd64.exe"

Write-Host "Build output is under build/bin/"
