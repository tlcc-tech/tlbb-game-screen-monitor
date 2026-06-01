# Builds OpenCV 4.11 + contrib for gocv on Windows.
# Skips compile when opencv/build/install already exists (CI cache restore).
# Creates C:\opencv\build\install junction so gocv default cgo paths work (no customenv).

param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
)

$ErrorActionPreference = "Stop"

$opencvRoot = Join-Path $RepoRoot "opencv"
$opencvBuild = Join-Path $opencvRoot "build"
$opencvInstall = Join-Path $opencvBuild "install"
$opencvBin = Join-Path $opencvInstall "x64/mingw/bin"
$opencvVersion = "4.11.0"

function Test-OpenCVReady {
    $msvcDll = Join-Path $opencvBin "opencv_core4110.dll"
    $mingwDll = Join-Path $opencvBin "libopencv_core4110.dll"
    return (Test-Path $msvcDll) -or (Test-Path $mingwDll)
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
        if (-not (Test-Path "CMakeCache.txt")) {
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
                -DBUILD_opencv_apps=OFF `
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
        } else {
            Write-Host "Reusing existing CMake cache in $opencvBuild"
        }

        Write-Host "Building and installing OpenCV..."
        cmake --build . --target install -j $env:NUMBER_OF_PROCESSORS
    } finally {
        Pop-Location
    }
}

function Ensure-GocvOpenCVPath {
    # gocv v0.41 default cgo expects C:/opencv/build/install
    $linkParent = "C:\opencv\build"
    $linkPath = Join-Path $linkParent "install"
    $target = (Resolve-Path $opencvInstall).Path

    New-Item -ItemType Directory -Force -Path $linkParent | Out-Null
    if (Test-Path $linkPath) {
        $item = Get-Item $linkPath -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
            if ($item.Target -contains $target -or $item.Target -eq $target) {
                Write-Host "gocv OpenCV junction already set: $linkPath -> $target"
                return
            }
            Remove-Item $linkPath -Force
        } else {
            Remove-Item $linkPath -Force -Recurse
        }
    }

    New-Item -ItemType Junction -Path $linkPath -Target $target | Out-Null
    Write-Host "Linked gocv OpenCV path: $linkPath -> $target"
}

function Set-OpenCVRuntimePath {
    if ($env:PATH -notlike "*$opencvBin*") {
        $env:PATH = "$opencvBin;$env:PATH"
    }
    if ($env:GITHUB_PATH) {
        Add-Content -Path $env:GITHUB_PATH -Value $opencvBin
    }
    Write-Host "OpenCV bin on PATH: $opencvBin"
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

Ensure-GocvOpenCVPath
Set-OpenCVRuntimePath
