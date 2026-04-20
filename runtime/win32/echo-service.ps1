param(
  [string]$Message = $env:ECHO_MESSAGE
)

if ([string]::IsNullOrWhiteSpace($Message)) {
  $Message = 'hello from service-template'
}

Write-Output $Message
Start-Sleep -Seconds 2
