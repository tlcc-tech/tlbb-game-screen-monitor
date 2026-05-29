# Installs prebuilt OpenCV for gocv (Windows only).
# Default install path: C:\opencv\build\install

$ErrorActionPreference = "Stop"

$opencvBin = "C:\opencv\build\install\x64\mingw\bin"
if (Test-Path $opencvBin) {
    Write-Host "OpenCV already present: $opencvBin"
    exit 0
}

$gocvTag = "v0.41.0"
$gocvDir = Join-Path $env:TEMP "gocv-$gocvTag"

if (-not (Test-Path $gocvDir)) {
    Write-Host "Cloning gocv $gocvTag..."
    git clone --depth 1 --branch $gocvTag https://github.com/hybridgroup/gocv.git $gocvDir
}

Push-Location $gocvDir
try {
    Write-Host "Downloading OpenCV prebuilt package..."
    cmd /c win_download_opencv.cmd
} finally {
    Pop-Location
}

if (-not (Test-Path $opencvBin)) {
    throw "OpenCV install failed; expected bin dir: $opencvBin"
}

Write-Host "OpenCV ready: $opencvBin"
