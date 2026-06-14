# Windows 10/11: docker compose up (local|dev|staging|prod)
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$EnvName
)
. "$PSScriptRoot\_lib.ps1"
Assert-EnvName -Name $EnvName
Assert-EnvFiles -Stage $EnvName
Write-Host "docker: building and starting mycourse-be-$EnvName..."
Invoke-DockerCompose -EnvName $EnvName -ComposeArgs @('up', '--build', '-d')
Write-Host "docker: stack started. Run: $PSScriptRoot\health-check.ps1 $EnvName"
