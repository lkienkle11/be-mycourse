# Shared helpers for docker/ compose scripts (Windows PowerShell 5.1+ / PowerShell 7+).
# Health URL/timeout aligned with scripts/pm2-reload-with-binary-rollback.sh and _lib.sh.
$ErrorActionPreference = 'Stop'

$script:RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$script:ValidEnvs = @('local', 'dev', 'staging', 'prod')
$script:HealthCheckUrl = if ($env:HEALTHCHECK_URL) { $env:HEALTHCHECK_URL } else { 'http://127.0.0.1:8080/api/v1/health' }
$script:RollbackHealthTimeoutSec = if ($env:ROLLBACK_HEALTH_TIMEOUT_SEC) { [int]$env:ROLLBACK_HEALTH_TIMEOUT_SEC } else { 90 }

function Assert-EnvName {
    param([Parameter(Mandatory)][string]$Name)
    if ($script:ValidEnvs -notcontains $Name) {
        $list = $script:ValidEnvs -join ' '
        throw "docker: invalid environment '$Name'. Expected one of: $list"
    }
}

function Assert-EnvFiles {
    param([Parameter(Mandatory)][string]$Stage)
    $base = Join-Path $script:RepoRoot '.env'
    $stageFile = Join-Path $script:RepoRoot ".env.$Stage"
    if (-not (Test-Path -LiteralPath $base)) {
        throw "docker: missing $base — copy from .env.example"
    }
    if (-not (Test-Path -LiteralPath $stageFile)) {
        throw "docker: missing $stageFile — copy from .env.$Stage.example"
    }
}

function Get-ComposeFilePath {
    param([Parameter(Mandatory)][string]$EnvName)
    Join-Path $script:RepoRoot "docker\compose.$EnvName.yml"
}

function Get-StackFilePath {
    param([Parameter(Mandatory)][string]$EnvName)
    Join-Path $script:RepoRoot "docker\stack.$EnvName.yml"
}

function Get-ComposeProjectName {
    param([Parameter(Mandatory)][string]$EnvName)
    "mycourse-be-$EnvName"
}

function Invoke-DockerCompose {
    param(
        [Parameter(Mandatory)][string]$EnvName,
        [Parameter(ValueFromRemainingArguments = $true)][string[]]$ComposeArgs
    )
    $file = Get-ComposeFilePath -EnvName $EnvName
    $project = Get-ComposeProjectName -EnvName $EnvName
    & docker compose -f $file -p $project @ComposeArgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Test-HealthOk {
    param([Parameter(Mandatory)][string]$Url)
    # curl.exe ships with Windows 10+ — same flags as _lib.sh / pm2 rollback script.
    if (Get-Command curl.exe -ErrorAction SilentlyContinue) {
        & curl.exe -fsS --connect-timeout 2 --max-time 5 $Url 2>$null | Out-Null
        return ($LASTEXITCODE -eq 0)
    }
    try {
        Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5 | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

function Wait-HealthEndpoint {
    param(
        [string]$Url = $script:HealthCheckUrl,
        [int]$TimeoutSec = $script:RollbackHealthTimeoutSec
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    Write-Host "docker: polling $Url (timeout ${TimeoutSec}s)..."
    while ((Get-Date) -lt $deadline) {
        if (Test-HealthOk -Url $Url) {
            Write-Host 'docker: health OK'
            return
        }
        Start-Sleep -Seconds 2
    }
    throw "docker: health check failed ($Url, timeout ${TimeoutSec}s)"
}
