# Builds OpenCV 4.11 for gocv into <repo>/opencv/build/install (Windows).
# Idempotent: skips download/build when install artifacts already exist.

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

function Test-OpenCVReady {
    return (Test-Path (Join-Path $opencvBin "opencv_core4110.dll"))
}

function Set-OpenCVBuildEnv {
    $libs = @(
        "opencv_core4110",
        "opencv_face4110",
        "opencv_videoio4110",
        "opencv_imgproc4110",
        "opencv_highgui4110",
        "opencv_imgcodecs4110",
        "opencv_objdetect4110",
        "opencv_features2d4110",
        "opencv_video4110",
        "opencv_dnn4110",
        "opencv_xfeatures2d4110",
        "opencv_plot4110",
        "opencv_tracking4110",
        "opencv_img_hash4110",
        "opencv_calib3d4110",
        "opencv_bgsegm4110",
        "opencv_photo4110",
        "opencv_aruco4110",
        "opencv_wechat_qrcode4110",
        "opencv_ximgproc4110",
        "opencv_xphoto4110",
        "opencv_xobjdetect4110"
    )

    $ldflags = "-L$opencvLib " + (($libs | ForEach-Object { "-l$_" }) -join " ")
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
    $contribDir = Join-Path $opencvRoot "opencv_contrib-$opencvVersion"
    if ((Test-Path $srcDir) -and (Test-Path $contribDir)) {
        return
    }

    New-Item -ItemType Directory -Force -Path $opencvRoot | Out-Null

    $srcZip = Join-Path $opencvRoot "opencv-$opencvVersion.zip"
    $contribZip = Join-Path $opencvRoot "opencv_contrib-$opencvVersion.zip"

    if (-not (Test-Path $srcZip)) {
        Write-Host "Downloading OpenCV $opencvVersion source..."
        Invoke-WebRequest -Uri "https://github.com/opencv/opencv/archive/$opencvVersion.zip" -OutFile $srcZip
    }
    if (-not (Test-Path $contribZip)) {
        Write-Host "Downloading OpenCV contrib $opencvVersion source..."
        Invoke-WebRequest -Uri "https://github.com/opencv/opencv_contrib/archive/$opencvVersion.zip" -OutFile $contribZip
    }

    if (-not (Test-Path $srcDir)) {
        Write-Host "Extracting OpenCV source..."
        Expand-Archive -Path $srcZip -DestinationPath $opencvRoot -Force
    }
    if (-not (Test-Path $contribDir)) {
        Write-Host "Extracting OpenCV contrib source..."
        Expand-Archive -Path $contribZip -DestinationPath $opencvRoot -Force
    }
}

function Build-OpenCV {
    Ensure-OpenCVSources

    $srcDir = Join-Path $opencvRoot "opencv-$opencvVersion"
    $contribDir = Join-Path $opencvRoot "opencv_contrib-$opencvVersion"
    New-Item -ItemType Directory -Force -Path $opencvBuild | Out-Null

    Push-Location $opencvBuild
    try {
        Write-Host "Configuring OpenCV with CMake..."
        cmake -G "MinGW Makefiles" `
            -DENABLE_CXX11=ON `
            -DOPENCV_EXTRA_MODULES_PATH="$contribDir/modules" `
            -DBUILD_SHARED_LIBS=ON `
            -DWITH_IPP=OFF `
            -DWITH_MSMF=OFF `
            -DBUILD_EXAMPLES=OFF `
            -DBUILD_TESTS=OFF `
            -DBUILD_PERF_TESTS=OFF `
            -DBUILD_opencv_java=OFF `
            -DBUILD_opencv_python=OFF `
            -DBUILD_opencv_python2=OFF `
            -DBUILD_opencv_python3=OFF `
            -DBUILD_DOCS=OFF `
            -DENABLE_PRECOMPILED_HEADERS=OFF `
            -DBUILD_opencv_saliency=OFF `
            -DBUILD_opencv_wechat_qrcode=ON `
            -DCPU_DISPATCH= `
            -DOPENCV_GENERATE_PKGCONFIG=ON `
            -DWITH_OPENCL_D3D11_NV=OFF `
            -DOPENCV_ALLOCATOR_STATS_COUNTER_TYPE=int64_t `
            -DOPENCV_ENABLE_NONFREE=ON `
            -Wno-dev `
            $srcDir

        Write-Host "Building and installing OpenCV (this may take a while)..."
        cmake --build . --target install
    } finally {
        Pop-Location
    }
}

if (Test-OpenCVReady) {
    Write-Host "OpenCV already built: $opencvBin"
} else {
    Build-OpenCV
    if (-not (Test-OpenCVReady)) {
        throw "OpenCV install failed; expected: $(Join-Path $opencvBin 'opencv_core4110.dll')"
    }
    Write-Host "OpenCV build complete: $opencvBin"
}

Set-OpenCVBuildEnv
