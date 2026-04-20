param(
  [string]$Contract = ".\verify\service-harness.json"
)

$Harness = "service-lasso-harness.exe"

if (-not (Get-Command $Harness -ErrorAction SilentlyContinue)) {
  Write-Error "service-lasso-harness.exe not found in PATH"
  exit 1
}

& $Harness run --contract $Contract
exit $LASTEXITCODE
