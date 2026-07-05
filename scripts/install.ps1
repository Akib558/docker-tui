#Requires -Version 5.1
<#
.SYNOPSIS
  docker-tui installer for Windows.
.DESCRIPTION
  Downloads the latest docker-tui release for your architecture and installs it
  to %LOCALAPPDATA%\docker-tui\bin, adding that directory to your user PATH.

  Usage:
    irm https://raw.githubusercontent.com/Akib558/docker-tui/main/scripts/install.ps1 | iex
#>
[CmdletBinding()]
param(
  [string]$Version = "latest",
  [string]$BinDir  = "$env:LOCALAPPDATA\docker-tui\bin"
)

$ErrorActionPreference = "Stop"
$repo = "Akib558/docker-tui"
function Info($m) { Write-Host "==> $m" -ForegroundColor Green }
function Fail($m) { Write-Host "error: $m" -ForegroundColor Red; exit 1 }

# --- detect arch ---------------------------------------------------------
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { Fail "unsupported architecture '$($env:PROCESSOR_ARCHITECTURE)'. Try: go install github.com/$repo@latest" }
}

# --- resolve release -----------------------------------------------------
$api = if ($Version -eq "latest") {
  "https://api.github.com/repos/$repo/releases/latest"
} else {
  "https://api.github.com/repos/$repo/releases/tags/$Version"
}

Info "Looking up the $Version release of $repo..."
try { $release = Invoke-RestMethod -Uri $api -Headers @{ "User-Agent" = "docker-tui-installer" } }
catch { Fail "no published release found yet. Install with Go instead:`n    go install github.com/$repo@latest" }

$asset = $release.assets | Where-Object { $_.name -match "_windows_$arch\.zip$" } | Select-Object -First 1
if (-not $asset) { Fail "no prebuilt binary for windows/$arch in the latest release.`n    Install with Go instead: go install github.com/$repo@latest" }

# --- download + extract --------------------------------------------------
$tmp = Join-Path $env:TEMP ("docker-tui-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
try {
  $zip = Join-Path $tmp "docker-tui.zip"
  Info "Downloading $($asset.name)..."
  Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing
  Expand-Archive -Path $zip -DestinationPath $tmp -Force

  $exe = Get-ChildItem -Path $tmp -Recurse -Filter "docker-tui.exe" | Select-Object -First 1
  if (-not $exe) { Fail "docker-tui.exe not found in the archive." }

  New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
  Copy-Item -Path $exe.FullName -Destination (Join-Path $BinDir "docker-tui.exe") -Force
  Info "Installed docker-tui.exe to $BinDir"
}
finally { Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue }

# --- add to user PATH ----------------------------------------------------
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$BinDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$BinDir", "User")
  Info "Added $BinDir to your user PATH. Restart your terminal to use 'docker-tui'."
}

Info "Done. Run 'docker-tui' to start (Docker Desktop / daemon must be running)."
