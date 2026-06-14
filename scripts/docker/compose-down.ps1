param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$EnvName
)
. "$PSScriptRoot\_lib.ps1"
Assert-EnvName -Name $EnvName
Write-Host "docker: stopping mycourse-be-$EnvName..."
Invoke-DockerCompose -EnvName $EnvName -ComposeArgs @('down', '--remove-orphans')
Write-Host 'docker: stack stopped.'
