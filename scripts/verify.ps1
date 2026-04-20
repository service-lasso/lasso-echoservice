param(
  [string]$Contract = ".\verify\service-harness.json",
  [string]$OutputDir = ".\output\verify"
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$contractPath = [System.IO.Path]::GetFullPath((Join-Path $root $Contract))
$outputPath = [System.IO.Path]::GetFullPath((Join-Path $root $OutputDir))
New-Item -ItemType Directory -Force -Path $outputPath | Out-Null

go run .\cmd\verify-harness --contract $contractPath --output-dir $outputPath
