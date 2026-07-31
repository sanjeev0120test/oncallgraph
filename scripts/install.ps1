# Install oncallgraph from a GitHub Release (or a local dist/ directory).
# Free, no accounts. Verifies SHA256 when SHA256SUMS is present.
$ErrorActionPreference = "Stop"

$Repo = if ($env:ONCALLGRAPH_REPO) { $env:ONCALLGRAPH_REPO } else { "sanjeev0120test/oncallgraph" }
$Version = if ($env:ONCALLGRAPH_VERSION) { $env:ONCALLGRAPH_VERSION } else { "latest" }
$InstallDir = if ($env:ONCALLGRAPH_INSTALL_DIR) { $env:ONCALLGRAPH_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "oncallgraph\bin" }
$DistDir = $env:ONCALLGRAPH_DIST_DIR

$arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($arch) {
  "AMD64|X64" { $arch = "amd64" }
  "ARM64" { $arch = "arm64" }
  default { throw "unsupported arch: $arch" }
}
$os = "windows"

$tmp = New-Item -ItemType Directory -Path ([System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [guid]::NewGuid().ToString()))
try {
  if ($DistDir) {
    $src = Get-ChildItem -Path $DistDir -Filter "oncallgraph-$os-$arch*" -File | Select-Object -First 1
    if (-not $src) { throw "no local binary for $os/$arch in $DistDir" }
    $bin = Join-Path $tmp.FullName "oncallgraph.exe"
    Copy-Item $src.FullName $bin
  } else {
    if ($Version -eq "latest") {
      $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
      $tag = $rel.tag_name
    } else {
      $tag = $Version
    }
    if (-not $tag) { throw "could not resolve release tag for $Repo" }
    $asset = "oncallgraph_${tag}_${os}_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$tag/$asset"
    Write-Host "downloading $url"
    $zipPath = Join-Path $tmp.FullName $asset
    Invoke-WebRequest -Uri $url -OutFile $zipPath
    $sumsPath = Join-Path $tmp.FullName "SHA256SUMS"
    try {
      Invoke-WebRequest -Uri "https://github.com/$Repo/releases/download/$tag/SHA256SUMS" -OutFile $sumsPath
    } catch {
      # optional
    }
    if ($env:ONCALLGRAPH_INSECURE -ne "1") {
      if (-not (Test-Path $sumsPath) -or -not (Get-Item $sumsPath).Length) {
        throw "SHA256SUMS missing or empty (set ONCALLGRAPH_INSECURE=1 to skip)"
      }
      $expected = (Get-Content $sumsPath | Where-Object { $_ -match [regex]::Escape($asset) } | Select-Object -First 1)
      if (-not $expected) { throw "SHA256SUMS has no entry for $asset" }
      $want = ($expected -split '\s+')[0].ToLower()
      $got = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLower()
      if ($want -ne $got) { throw "SHA256 mismatch for $asset" }
      Write-Host "checksum ok: $asset"
    } else {
      Write-Warning "skipping checksum verification (ONCALLGRAPH_INSECURE=1)"
    }
    Expand-Archive -Path $zipPath -DestinationPath $tmp.FullName -Force
    $bin = Join-Path $tmp.FullName "oncallgraph_${tag}_${os}_${arch}.exe"
    if (-not (Test-Path $bin)) { throw "extracted binary missing: $bin" }
  }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  $dest = Join-Path $InstallDir "oncallgraph.exe"
  Copy-Item $bin $dest -Force
  Write-Host "installed $dest"
  & $dest version
} finally {
  Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}
