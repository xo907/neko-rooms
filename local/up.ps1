# Windows wrapper for local/up.sh. Docker and tailscale run inside WSL, so the
# script runs there. Pass the same arguments as up.sh (--no-build, down, logs).
#
#   .\local\up.ps1
#   .\local\up.ps1 -Distro Ubuntu down
param(
  [string]$Distro = "",
  [Parameter(ValueFromRemainingArguments = $true)][string[]]$Rest
)

$root = Split-Path -Parent $PSScriptRoot
$wslArgs = @()
if ($Distro) { $wslArgs += @("-d", $Distro) }

$wslRoot = (& wsl.exe @wslArgs -- wslpath -a ($root -replace '\\', '/')).Trim()
& wsl.exe @wslArgs -- bash "$wslRoot/local/up.sh" @Rest
exit $LASTEXITCODE
