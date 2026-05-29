# Requires: Go 1.22+, Node.js 18+, Wails CLI v2, MinGW + CMake (Windows)
# Run this in PowerShell from the project root.

$ErrorActionPreference = "Stop"

& "$PSScriptRoot/setup-opencv.ps1"

$Version = "dev"
$match = Select-String -Path "wails.json" -Pattern '"productVersion"\s*:\s*"([^"]+)"' | Select-Object -First 1
if ($match -and $match.Matches.Count -gt 0) {
	$Version = $match.Matches[0].Groups[1].Value
}

wails build -platform windows/amd64 -clean -tags customenv -ldflags "-X main.AppVersion=$Version"
Copy-Item -Force "build/bin/tlbb-game-screen-monitor.exe" "build/bin/游戏掉线监控-windows-amd64.exe"

$opencvBin = Join-Path $PSScriptRoot "..\opencv\build\install\x64\mingw\bin" | Resolve-Path
if (Test-Path $opencvBin) {
    Copy-Item -Force "$opencvBin\opencv_*.dll" "build/bin/"
    Write-Host "Copied OpenCV DLLs to build/bin/"
}

Write-Host "Build output is under build/bin/"
