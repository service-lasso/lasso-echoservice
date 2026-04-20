param(
  [string]$Contract = ".\verify\service-harness.json",
  [string]$OutputDir = ".\output\verify"
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Resolve-HarnessBinary {
  if ($env:SERVICE_LASSO_HARNESS_BIN) {
    return $env:SERVICE_LASSO_HARNESS_BIN
  }

  $cmd = Get-Command service-lasso-harness -ErrorAction SilentlyContinue
  if ($cmd) {
    return $cmd.Source
  }

  $cmdExe = Get-Command service-lasso-harness.exe -ErrorAction SilentlyContinue
  if ($cmdExe) {
    return $cmdExe.Source
  }

  throw "service-lasso-harness binary not found. Set SERVICE_LASSO_HARNESS_BIN or add it to PATH."
}

$contractPath = [System.IO.Path]::GetFullPath((Join-Path $root $Contract))
$outputPath = [System.IO.Path]::GetFullPath((Join-Path $root $OutputDir))
New-Item -ItemType Directory -Force -Path $outputPath | Out-Null

$resolvedContractPath = Join-Path $root 'verify\service-harness.ci.json'
$runOutputDir = Join-Path $outputPath 'harness-run'

$doc = Get-Content $contractPath -Raw | ConvertFrom-Json
$doc.artifact.path = '..\dist\echo-service-win32.zip'
$doc | ConvertTo-Json -Depth 10 | Set-Content $resolvedContractPath

$harness = Resolve-HarnessBinary
& $harness validate-contract --contract $resolvedContractPath
& $harness run --contract $resolvedContractPath --output-dir $runOutputDir
