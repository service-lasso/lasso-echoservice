$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$runtime = Join-Path $root 'runtime'
$staging = Join-Path $dist 'echo-service-win32'
$zipPath = Join-Path $dist 'echo-service-win32.zip'

New-Item -ItemType Directory -Force -Path $dist | Out-Null
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
New-Item -ItemType Directory -Force -Path $staging | Out-Null

Copy-Item -Recurse -Force (Join-Path $runtime 'win32') (Join-Path $staging 'runtime')
Copy-Item -Recurse -Force (Join-Path $root 'config') (Join-Path $staging 'config')

if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
Compress-Archive -Path (Join-Path $staging '*') -DestinationPath $zipPath
Write-Host "Created $zipPath"
