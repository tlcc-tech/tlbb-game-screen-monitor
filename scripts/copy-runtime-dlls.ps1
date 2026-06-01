# Copy OpenCV + MinGW + DM files into build/bin/runtime/
param(
    [string]$DestRoot = "build/bin",
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$DestRoot = Join-Path $RepoRoot $DestRoot
$RuntimeDir = Join-Path $DestRoot "runtime"
$OpenCVDest = Join-Path $RuntimeDir "opencv"
$MingwDest = Join-Path $RuntimeDir "mingw"
$DmDest = Join-Path $RuntimeDir "dm"

New-Item -ItemType Directory -Force -Path $OpenCVDest, $MingwDest, $DmDest | Out-Null

$opencvBin = Join-Path $RepoRoot "opencv/build/install/x64/mingw/bin"
if (Test-Path $opencvBin) {
    Copy-Item -Force "$opencvBin/libopencv_*.dll" $OpenCVDest -ErrorAction SilentlyContinue
    Copy-Item -Force "$opencvBin/opencv_*.dll" $OpenCVDest -ErrorAction SilentlyContinue
    Write-Host "Copied OpenCV DLLs to $OpenCVDest"
}

$mingwRuntime = @(
    "libgcc_s_seh-1.dll",
    "libstdc++-6.dll",
    "libwinpthread-1.dll"
)

function Find-MinGWBin {
    $dirs = [System.Collections.Generic.List[string]]::new()
    $gpp = Get-Command g++.exe -ErrorAction SilentlyContinue
    if ($gpp) { $dirs.Add((Split-Path $gpp.Source -Parent)) }
    foreach ($pattern in @(
        "C:\mingw64\bin",
        "C:\msys64\mingw64\bin",
        "C:\ProgramData\chocolatey\lib\mingw\tools\install\mingw64\bin"
    )) {
        if (Test-Path $pattern) { $dirs.Add($pattern) }
    }
    Get-ChildItem "C:\Program Files\mingw-w64" -Directory -ErrorAction SilentlyContinue |
        ForEach-Object { $dirs.Add((Join-Path $_.FullName "mingw64\bin")) }
    foreach ($dir in $dirs) {
        if (-not (Test-Path $dir)) { continue }
        if (Test-Path (Join-Path $dir "libgcc_s_seh-1.dll")) { return $dir }
    }
    return $null
}

$mingwBin = Find-MinGWBin
if (-not $mingwBin) {
    throw "MinGW bin directory not found; cannot copy runtime DLLs"
}
foreach ($name in $mingwRuntime) {
    $src = Join-Path $mingwBin $name
    if (-not (Test-Path $src)) { throw "Missing MinGW runtime DLL: $src" }
    Copy-Item -Force $src $MingwDest
    Write-Host "Copied $name to mingw/"
}

$dmSrc = Join-Path $RepoRoot "third_party/dm"
if (Test-Path $dmSrc) {
    Copy-Item -Force "$dmSrc/*" $DmDest
    Write-Host "Copied dm DLLs to dm/"
}

$dmcapture = Join-Path $DestRoot "runtime/dmcapture.exe"
if (-not (Test-Path $dmcapture)) {
    Write-Warning "dmcapture.exe not found at $dmcapture"
}

Write-Host "Runtime layout ready under $RuntimeDir"
