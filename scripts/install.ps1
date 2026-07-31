# Install opsgraph from a GitHub Release (or a local dist/ directory).
# Free, no accounts. Verifies SHA256 when SHA256SUMS is present.
$ErrorActionPreference = "Stop"

$Repo = if ($env:OPSGRAPH_REPO) { $env:OPSGRAPH_REPO } else { "sanjeev0120test/opsgraph" }
$Version = if ($env:OPSGRAPH_VERSION) { $env:OPSGRAPH_VERSION } else { "latest" }
$InstallDir = if ($env:OPSGRAPH_INSTALL_DIR) { $env:OPSGRAPH_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "opsgraph\bin" }
$DistDir = $env:OPSGRAPH_DIST_DIR
$ReleaseDir = $env:OPSGRAPH_RELEASE_DIR

$arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($arch) {
  "AMD64|X64" { $arch = "amd64" }
  "ARM64" { $arch = "arm64" }
  default { throw "unsupported arch: $arch" }
}
$os = "windows"

function Install-FromArchive {
  param(
    [string]$Tag,
    [string]$ZipPath,
    [string]$SumsPath
  )
  $asset = Split-Path $ZipPath -Leaf
  if ($env:OPSGRAPH_INSECURE -ne "1") {
    if (-not (Test-Path $SumsPath) -or -not (Get-Item $SumsPath).Length) {
      throw "SHA256SUMS missing or empty (set OPSGRAPH_INSECURE=1 to skip)"
    }
    $expected = (Get-Content $SumsPath | Where-Object { $_ -match [regex]::Escape($asset) } | Select-Object -First 1)
    if (-not $expected) { throw "SHA256SUMS has no entry for $asset" }
    $want = ($expected -split '\s+')[0].ToLower()
    $got = (Get-FileHash -Algorithm SHA256 -Path $ZipPath).Hash.ToLower()
    if ($want -ne $got) { throw "SHA256 mismatch for $asset" }
    Write-Host "checksum ok: $asset"
  } else {
    Write-Warning "skipping checksum verification (OPSGRAPH_INSECURE=1)"
  }
  Expand-Archive -Path $ZipPath -DestinationPath $tmp.FullName -Force
  $path = Join-Path $tmp.FullName "opsgraph_${Tag}_${os}_${arch}.exe"
  if (-not (Test-Path $path)) { throw "extracted binary missing: $path" }
  return $path
}

$tmp = New-Item -ItemType Directory -Path ([System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [guid]::NewGuid().ToString()))
try {
  if ($DistDir) {
    $src = Get-ChildItem -Path $DistDir -Filter "opsgraph-$os-$arch*" -File | Select-Object -First 1
    if (-not $src) { throw "no local binary for $os/$arch in $DistDir" }
    $bin = Join-Path $tmp.FullName "opsgraph.exe"
    Copy-Item $src.FullName $bin
  } elseif ($ReleaseDir) {
    if (-not $Version -or $Version -eq "latest") {
      throw "OPSGRAPH_VERSION must be set when using OPSGRAPH_RELEASE_DIR"
    }
    $tag = $Version
    $asset = "opsgraph_${tag}_${os}_${arch}.zip"
    $zipPath = Join-Path $ReleaseDir $asset
    if (-not (Test-Path $zipPath)) { throw "missing release asset $asset in $ReleaseDir" }
    $sumsPath = Join-Path $ReleaseDir "SHA256SUMS"
    $bin = Install-FromArchive -Tag $tag -ZipPath $zipPath -SumsPath $sumsPath
  } else {
    if ($Version -eq "latest") {
      $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
      $tag = $rel.tag_name
    } else {
      $tag = $Version
    }
    if (-not $tag) { throw "could not resolve release tag for $Repo" }
    $asset = "opsgraph_${tag}_${os}_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$tag/$asset"
    Write-Host "downloading $url"
    $zipPath = Join-Path $tmp.FullName $asset
    Invoke-WebRequest -Uri $url -OutFile $zipPath
    $sumsPath = Join-Path $tmp.FullName "SHA256SUMS"
    try {
      Invoke-WebRequest -Uri "https://github.com/$Repo/releases/download/$tag/SHA256SUMS" -OutFile $sumsPath
    } catch {
      # optional when insecure
    }
    $bin = Install-FromArchive -Tag $tag -ZipPath $zipPath -SumsPath $sumsPath
  }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  $dest = Join-Path $InstallDir "opsgraph.exe"
  Copy-Item $bin $dest -Force
  Write-Host "installed $dest"
  & $dest version
} finally {
  Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}
