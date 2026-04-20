$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot

$required = @(
  (Join-Path $root 'service.json'),
  (Join-Path $root 'verify\service-harness.json'),
  (Join-Path $root 'runtime\win32\echo-service.ps1')
)

foreach ($path in $required) {
  if (-not (Test-Path $path)) {
    throw "Missing required file: $path"
  }
}

$service = Get-Content (Join-Path $root 'service.json') -Raw | ConvertFrom-Json
if ($service.id -ne 'echo-service') {
  throw 'service.json id mismatch'
}

$contract = Get-Content (Join-Path $root 'verify\service-harness.json') -Raw | ConvertFrom-Json
if ($contract.serviceId -ne 'echo-service') {
  throw 'service-harness.json serviceId mismatch'
}

$env:ECHO_MESSAGE = 'pipeline test message'
$output = & (Join-Path $root 'runtime\win32\echo-service.ps1') | Out-String
if ($output -notmatch 'pipeline test message') {
  throw 'Echo runtime output mismatch'
}

Write-Host 'Template tests passed (Windows)'
