# Requires: Go 1.22+, Node.js 18+, Wails CLI v2
# Run this in PowerShell from the project root.

$ErrorActionPreference = "Stop"

$Version = "dev"
$match = Select-String -Path "wails.json" -Pattern '"productVersion"\s*:\s*"([^"]+)"' | Select-Object -First 1
if ($match -and $match.Matches.Count -gt 0) {
	$Version = $match.Matches[0].Groups[1].Value
}
wails build -platform windows/amd64 -clean -ldflags "-X main.AppVersion=$Version"
Copy-Item -Force "build/bin/tlbb-game-screen-monitor.exe" "build/bin/游戏掉线监控-windows-amd64.exe"
Write-Host "Build output is under build/bin/"
