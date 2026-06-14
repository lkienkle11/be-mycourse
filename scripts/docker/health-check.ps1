param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$EnvName
)
. "$PSScriptRoot\_lib.ps1"
Assert-EnvName -Name $EnvName
Wait-HealthEndpoint -Url $script:HealthCheckUrl -TimeoutSec $script:RollbackHealthTimeoutSec
