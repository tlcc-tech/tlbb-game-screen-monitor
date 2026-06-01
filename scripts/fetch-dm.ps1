# Ensures dm.dll and DmReg.dll exist in third_party/dm/.
# Files are vendored in the repo; download only as fallback when missing.

$ErrorActionPreference = "Stop"

$BaseUrl = "https://raw.githubusercontent.com/xxxxue/xDM/main/C%23%E5%A4%A7%E6%BC%A0%E5%85%8D%E6%B3%A8%E5%86%8C(%E4%BD%BF%E7%94%A8DmReg.dll)/dll"
$OutDir = Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..")).Path "third_party\dm"
$OutDir = [System.IO.Path]::GetFullPath($OutDir)
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$Files = @("dm.dll", "DmReg.dll")
$allPresent = $true
foreach ($name in $Files) {
    $dest = Join-Path $OutDir $name
    if (-not (Test-Path $dest) -or (Get-Item $dest).Length -lt 1024) {
        $allPresent = $false
        break
    }
}

if ($allPresent) {
    Write-Host "DM files already present in $OutDir (using vendored copies)"
    foreach ($name in $Files) {
        $dest = Join-Path $OutDir $name
        Write-Host "  -> $dest ($((Get-Item $dest).Length) bytes)"
    }
    exit 0
}

Write-Host "Vendored DM files missing; downloading from xxxxue/xDM ..."
foreach ($name in $Files) {
    $url = "$BaseUrl/$name"
    $dest = Join-Path $OutDir $name
    Write-Host "Downloading $name ..."
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
    if (-not (Test-Path $dest) -or (Get-Item $dest).Length -lt 1024) {
        throw "Download failed or file too small: $dest"
    }
    Write-Host "  -> $dest ($((Get-Item $dest).Length) bytes)"
}

Write-Host "DM files ready in $OutDir"
