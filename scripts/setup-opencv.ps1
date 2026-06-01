# Builds a slim OpenCV 4.11 (core+imgproc+imgcodecs) for gocv MatchTemplate.
# Skips compile when opencv/build/install already exists (CI cache restore).

param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$opencvRoot = Join-Path $RepoRoot "opencv"
$opencvBuild = Join-Path $opencvRoot "build"
$opencvInstall = Join-Path $opencvBuild "install"
$opencvBin = Join-Path $opencvInstall "x64/mingw/bin"
$opencvInclude = Join-Path $opencvInstall "include"
$opencvLib = Join-Path $opencvInstall "x64/mingw/lib"
$opencvVersion = "4.11.0"

# Modules required for MatchTemplate + CvtColor + ImageToMatRGBA
$opencvModules = @("core", "imgproc", "imgcodecs")
$opencvLibs = @("opencv_core4110", "opencv_imgproc4110", "opencv_imgcodecs4110")

function Test-OpenCVReady {
    $msvcDll = Join-Path $opencvBin "opencv_core4110.dll"
    $mingwDll = Join-Path $opencvBin "libopencv_core4110.dll"
    return (Test-Path $msvcDll) -or (Test-Path $mingwDll)
}

function Set-OpenCVBuildEnv {
    $ldflags = "-L$opencvLib " + (($opencvLibs | ForEach-Object { "-l$_" }) -join " ")
    $env:CGO_ENABLED = "1"
    $env:CGO_CXXFLAGS = "--std=c++11"
    $env:CGO_CPPFLAGS = "-I$opencvInclude"
    $env:CGO_LDFLAGS = $ldflags
    if ($env:PATH -notlike "*$opencvBin*") {
        $env:PATH = "$opencvBin;$env:PATH"
    }

    if ($env:GITHUB_ENV) {
        Add-Content -Path $env:GITHUB_ENV -Value "CGO_ENABLED=1"
        Add-Content -Path $env:GITHUB_ENV -Value "CGO_CXXFLAGS=--std=c++11"
        Add-Content -Path $env:GITHUB_ENV -Value "CGO_CPPFLAGS=-I$opencvInclude"
        Add-Content -Path $env:GITHUB_ENV -Value "CGO_LDFLAGS=$ldflags"
        Add-Content -Path $env:GITHUB_PATH -Value $opencvBin
    }

    Write-Host "OpenCV build env ready:"
    Write-Host "  include: $opencvInclude"
    Write-Host "  lib:     $opencvLib"
    Write-Host "  bin:     $opencvBin"
}

function Ensure-OpenCVSources {
    $srcDir = Join-Path $opencvRoot "opencv-$opencvVersion"
    if (Test-Path $srcDir) {
        return
    }

    New-Item -ItemType Directory -Force -Path $opencvRoot | Out-Null
    $srcZip = Join-Path $opencvRoot "opencv-$opencvVersion.zip"

    if (-not (Test-Path $srcZip)) {
        Write-Host "Downloading OpenCV $opencvVersion source..."
        Invoke-WebRequest -Uri "https://github.com/opencv/opencv/archive/$opencvVersion.zip" -OutFile $srcZip
    }

    Write-Host "Extracting OpenCV source..."
    Expand-Archive -Path $srcZip -DestinationPath $opencvRoot -Force
}

function Build-OpenCV {
    Ensure-OpenCVSources

    $srcDir = Join-Path $opencvRoot "opencv-$opencvVersion"
    New-Item -ItemType Directory -Force -Path $opencvBuild | Out-Null

    $buildList = ($opencvModules -join ",")

    Push-Location $opencvBuild
    try {
        Write-Host "Configuring slim OpenCV (BUILD_LIST=$buildList)..."
        cmake -G "MinGW Makefiles" `
            -DENABLE_CXX11=ON `
            -DBUILD_LIST="$buildList" `
            -DBUILD_SHARED_LIBS=ON `
            -DWITH_IPP=OFF `
            -DWITH_MSMF=OFF `
            -DWITH_FFMPEG=OFF `
            -DWITH_JPEG=ON `
            -DWITH_PNG=ON `
            -DBUILD_EXAMPLES=OFF `
            -DBUILD_TESTS=OFF `
            -DBUILD_PERF_TESTS=OFF `
            -DBUILD_opencv_apps=OFF `
            -DBUILD_opencv_java=OFF `
            -DBUILD_opencv_python=OFF `
            -DBUILD_opencv_python2=OFF `
            -DBUILD_opencv_python3=OFF `
            -DBUILD_DOCS=OFF `
            -DENABLE_PRECOMPILED_HEADERS=OFF `
            -DCPU_DISPATCH= `
            -DWITH_OPENCL=OFF `
            -DWITH_OPENCL_D3D11_NV=OFF `
            -DOPENCV_ALLOCATOR_STATS_COUNTER_TYPE=int64_t `
            -Wno-dev `
            $srcDir

        Write-Host "Building and installing OpenCV..."
        cmake --build . --target install -j $env:NUMBER_OF_PROCESSORS
    } finally {
        Pop-Location
    }
}

if (Test-OpenCVReady) {
    Write-Host "OpenCV install cache hit: $opencvBin"
} else {
    Build-OpenCV
    if (-not (Test-OpenCVReady)) {
        throw "OpenCV install failed; expected libopencv_core4110.dll in $opencvBin"
    }
    Write-Host "OpenCV build complete: $opencvBin"
}

Set-OpenCVBuildEnv
