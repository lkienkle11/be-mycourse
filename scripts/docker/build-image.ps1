param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$EnvName
)
. "$PSScriptRoot\_lib.ps1"
Assert-EnvName -Name $EnvName
$imageTag = "mycourse-io-be:$EnvName"
Write-Host "docker: building image $imageTag (STAGE=$EnvName)..."
& docker build `
    --build-arg "STAGE=$EnvName" `
    -t $imageTag `
    -f (Join-Path $script:RepoRoot 'Dockerfile') `
    $script:RepoRoot
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "docker: built $imageTag"
