# Copy OpenCV + MinGW runtime DLLs next to the built exe.
param(
    [string]$DestDir = "build/bin",
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$DestDir = Join-Path $RepoRoot $DestDir
New-Item -ItemType Directory -Force -Path $DestDir | Out-Null

$opencvBin = Join-Path $RepoRoot "opencv/build/install/x64/mingw/bin"
if (Test-Path $opencvBin) {
    Copy-Item -Force "$opencvBin/libopencv_*.dll" $DestDir -ErrorAction SilentlyContinue
    Copy-Item -Force "$opencvBin/opencv_*.dll" $DestDir -ErrorAction SilentlyContinue
    Write-Host "Copied OpenCV DLLs from $opencvBin"
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
        $probe = Join-Path $dir "libgcc_s_seh-1.dll"
        if (Test-Path $probe) { return $dir }
    }
    return $null
}

$mingwBin = Find-MinGWBin
if (-not $mingwBin) {
    throw "MinGW bin directory not found; cannot copy runtime DLLs"
}

foreach ($name in $mingwRuntime) {
    $src = Join-Path $mingwBin $name
    if (-not (Test-Path $src)) {
        throw "Missing MinGW runtime DLL: $src"
    }
    Copy-Item -Force $src $DestDir
    Write-Host "Copied $name"
}

Write-Host "Runtime DLLs ready in $DestDir"
