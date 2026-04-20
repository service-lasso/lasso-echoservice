$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$staging = Join-Path $dist 'echo-service-win32'
$zipPath = Join-Path $dist 'echo-service-win32.zip'
$binaryPath = Join-Path $staging 'echo-service.exe'

New-Item -ItemType Directory -Force -Path $dist | Out-Null
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
New-Item -ItemType Directory -Force -Path $staging | Out-Null

Push-Location $root
try {
  go build -o $binaryPath .
}
finally {
  Pop-Location
}

Copy-Item -Force (Join-Path $root 'service.json') (Join-Path $staging 'service.json')
Copy-Item -Force (Join-Path $root 'README.md') (Join-Path $staging 'README.md')
Copy-Item -Recurse -Force (Join-Path $root 'config') (Join-Path $staging 'config')

if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
Compress-Archive -Path (Join-Path $staging '*') -DestinationPath $zipPath
Write-Host "Created $zipPath"
