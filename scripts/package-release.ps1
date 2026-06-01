# Stage exe + runtime/ and create a clean zip (no nested zip, no duplicate exe).
param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$binDir = Join-Path $RepoRoot "build/bin"
$exeName = "游戏掉线监控-windows-amd64.exe"
$exePath = Join-Path $binDir $exeName
if (-not (Test-Path $exePath)) {
    throw "Missing $exePath — run wails build first"
}

$stage = Join-Path $RepoRoot "build/package-stage"
$runtimeSrc = Join-Path $binDir "runtime"
if (-not (Test-Path $runtimeSrc)) {
    throw "Missing $runtimeSrc — run copy-runtime-dlls.ps1 first"
}

Remove-Item -Recurse -Force $stage -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $stage | Out-Null

Copy-Item -Force $exePath (Join-Path $stage $exeName)
Copy-Item -Recurse -Force $runtimeSrc (Join-Path $stage "runtime")

# MinGW runtime must sit beside exe for Windows loader at process start.
$mingwNames = @("libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll")
foreach ($name in $mingwNames) {
    $src = Join-Path $binDir $name
    if (Test-Path $src) {
        Copy-Item -Force $src (Join-Path $stage $name)
    }
}

$zipOut = Join-Path $RepoRoot "build/游戏掉线监控-windows-amd64.zip"
Remove-Item -Force $zipOut -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zipOut -Force
Remove-Item -Recurse -Force $stage

Write-Host "Release zip: $zipOut"
Write-Host "Contents: $exeName + 3 MinGW DLLs + runtime/"
