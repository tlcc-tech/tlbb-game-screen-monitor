# Stage release folder (extracted layout) + optional zip for auto-update.
param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$binDir = Join-Path $RepoRoot "build/bin"
$releaseDir = Join-Path $RepoRoot "build/release"
$exeName = "游戏掉线监控-windows-amd64.exe"
$exePath = Join-Path $binDir $exeName
if (-not (Test-Path $exePath)) {
    throw "Missing $exePath — run wails build first"
}

$runtimeSrc = Join-Path $binDir "runtime"
if (-not (Test-Path $runtimeSrc)) {
    throw "Missing $runtimeSrc — run copy-runtime-dlls.ps1 first"
}

Remove-Item -Recurse -Force $releaseDir -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

Copy-Item -Force $exePath (Join-Path $releaseDir $exeName)
Copy-Item -Recurse -Force $runtimeSrc (Join-Path $releaseDir "runtime")

# All DLLs beside exe — required by Windows loader for OpenCV + MinGW at process start.
Get-ChildItem -Path $binDir -Filter "*.dll" -File | ForEach-Object {
    Copy-Item -Force $_.FullName (Join-Path $releaseDir $_.Name)
}

$dllCount = (Get-ChildItem -Path $releaseDir -Filter "*.dll" -File).Count
if ($dllCount -lt 4) {
    throw "Too few DLLs in release dir ($dllCount). OpenCV copy may have failed."
}

$zipOut = Join-Path $releaseDir "游戏掉线监控-windows-amd64.zip"
Remove-Item -Force $zipOut -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $releaseDir "*") -DestinationPath $zipOut -Force

Write-Host "Release folder: $releaseDir ($dllCount DLLs + exe + runtime/ + zip)"
Write-Host "Release zip: $zipOut"
