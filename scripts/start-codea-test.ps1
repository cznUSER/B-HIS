# Sync the committed B-HIS revision to Code A, start its Host, tunnel it locally,
# and open the local browser against the real Code A API/database.

[CmdletBinding()]
param(
    [string]$SshHost = 'codeA',
    [string]$RemoteRoot = 'E:\B-HIS',
    [int]$WebPort = 8888,
    [int]$ApiPort = 9999,
    [switch]$NoOpen,
    [switch]$NoBuild,
    [switch]$NoSync,
    [switch]$KeepExistingTunnel
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Invoke-Checked {
    param([string]$FilePath, [string[]]$ArgumentList)
    & $FilePath @ArgumentList
    if ($LASTEXITCODE -ne 0) {
        throw "$FilePath failed with exit code $LASTEXITCODE"
    }
}

function Test-LocalPort {
    param([int]$Port)
    try {
        $client = [System.Net.Sockets.TcpClient]::new()
        $task = $client.ConnectAsync('127.0.0.1', $Port)
        if (-not $task.Wait(500)) { $client.Dispose(); return $false }
        $connected = $client.Connected
        $client.Dispose()
        return $connected
    } catch {
        return $false
    }
}

function Wait-LocalPort {
    param([int]$Port, [int]$Seconds = 20)
    for ($i = 0; $i -lt ($Seconds * 2); $i++) {
        if (Test-LocalPort $Port) { return }
        Start-Sleep -Milliseconds 500
    }
    throw "local port $Port did not become available"
}

$git = (Get-Command git -ErrorAction Stop).Source
$ssh = (Get-Command ssh -ErrorAction Stop).Source
$scp = (Get-Command scp -ErrorAction Stop).Source

$head = (& $git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $head -notmatch '^[0-9a-f]{40}$') { throw 'cannot resolve the committed revision' }

# The hook runs after commits. Refuse tracked edits so Code A never tests a
# different revision from the one uploaded to GitLab.
$trackedChanges = @(& $git diff --name-only; & $git diff --cached --name-only)
if ($trackedChanges.Count -gt 0) {
    throw "tracked changes are not committed; commit them before Code A testing: $($trackedChanges -join ', ')"
}

$stage = Join-Path $env:TEMP "b-his-codea-$PID"
$archive = Join-Path $stage 'b-his-source.zip'
$hostExe = Join-Path $stage 'b-his-host.exe'
$config = Join-Path $Root 'config.local.yaml'
New-Item -ItemType Directory -Force -Path $stage | Out-Null
try {
    if (-not $NoBuild) {
        Write-Host "Building Host from $($head.Substring(0, 8)) ..."
        Invoke-Checked $git @('archive', '--format=zip', "--output=$archive", 'HEAD')
        Expand-Archive -LiteralPath $archive -DestinationPath $stage -Force

        Push-Location (Join-Path $stage 'web')
        Invoke-Checked 'npm.cmd' @('ci', '--ignore-scripts')
        Invoke-Checked 'npm.cmd' @('run', 'build:demo')
        Pop-Location

        $assets = Join-Path $stage 'internal\webui\assets'
        if (Test-Path $assets) { Remove-Item -Recurse -Force $assets }
        New-Item -ItemType Directory -Force -Path $assets | Out-Null
        Copy-Item -Recurse -Force (Join-Path $stage 'web\dist\*') $assets

        Push-Location $stage
        $oldCgo = $env:CGO_ENABLED; $oldGoos = $env:GOOS; $oldGoarch = $env:GOARCH
        $env:CGO_ENABLED = '0'; $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
        try { Invoke-Checked 'go.exe' @('build', '-o', $hostExe, './cmd/host') }
        finally {
            $env:CGO_ENABLED = $oldCgo; $env:GOOS = $oldGoos; $env:GOARCH = $oldGoarch
            Pop-Location
        }
    }

    if (-not (Test-Path $hostExe)) { throw "Host executable not found: $hostExe" }
    if (-not (Test-Path $config)) { throw "missing local runtime config: $config" }

    Write-Host "Checking $SshHost ..."
    $probe = & $ssh -o BatchMode=yes -o ConnectTimeout=10 $SshHost 'echo CODE_A_OK'
    if ($LASTEXITCODE -ne 0 -or ($probe -notmatch 'CODE_A_OK')) { throw "cannot reach $SshHost" }

    if (-not $NoSync) {
        Write-Host "Uploading revision and runtime package to Code A ..."
        Invoke-Checked $scp @('-q', $hostExe, "${SshHost}:E:/B-HIS/b-his-host.exe")
        # config.local.yaml is gitignored and is sent only to the configured Code A host.
        Invoke-Checked $scp @('-q', $config, "${SshHost}:E:/B-HIS/config.local.yaml")
    }

    $remoteScript = @'
$ErrorActionPreference = 'Stop'
$root = '__REMOTE_ROOT__'
New-Item -ItemType Directory -Force -Path $root | Out-Null
$exe = Join-Path $root 'b-his-host.exe'
$running = Get-Process -Name 'b-his-host' -ErrorAction SilentlyContinue
if ($running) { $running | Stop-Process -Force; Start-Sleep -Milliseconds 500 }
$logDir = Join-Path $root 'logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$stdout = Join-Path $logDir 'codea-test-host.stdout.log'
$stderr = Join-Path $logDir 'codea-test-host.stderr.log'
$pidFile = Join-Path $logDir 'codea-test-host.pid'
$runnerScript = Join-Path $root 'run-codea-test-host.ps1'
Remove-Item -Force $stdout, $stderr -ErrorAction SilentlyContinue
$runnerContent = @(
  '$root = ''' + $root + ''''
  '$exe = Join-Path $root ''b-his-host.exe'''
  '$stdout = Join-Path $root ''logs\codea-test-host.stdout.log'''
  '$stderr = Join-Path $root ''logs\codea-test-host.stderr.log'''
  'Set-Location $root'
  '& $exe -no-browser 1>> $stdout 2>> $stderr'
) -join [Environment]::NewLine
Set-Content -LiteralPath $runnerScript -Value $runnerContent -Encoding UTF8
$commandLine = 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "' + $runnerScript + '"'
$created = Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{ CommandLine = $commandLine; CurrentDirectory = $root }
if ($created.ReturnValue -ne 0 -or $created.ProcessId -le 0) { throw "WMI could not start the Code A Host (return=$($created.ReturnValue))" }
Set-Content -LiteralPath $pidFile -Value $created.ProcessId -Encoding ASCII
for ($i = 0; $i -lt 40; $i++) {
  try {
    $health = Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:9999/health' -TimeoutSec 2
    if ($health.StatusCode -eq 200) { Write-Output ('RUNNER_PID=' + $created.ProcessId); Write-Output $health.Content; exit 0 }
  } catch { }
  if (-not (Get-Process -Id $created.ProcessId -ErrorAction SilentlyContinue)) { break }
  Start-Sleep -Milliseconds 500
}
Get-Content $stderr -Tail 80 -ErrorAction SilentlyContinue
throw 'Code A Host did not remain healthy on 127.0.0.1:9999'
'@
    $remoteScript = $remoteScript.Replace('__REMOTE_ROOT__', $RemoteRoot)
    $remoteLauncher = Join-Path $stage 'start-codea-test-remote.ps1'
    Set-Content -LiteralPath $remoteLauncher -Value $remoteScript -Encoding UTF8
    Invoke-Checked $scp @('-q', $remoteLauncher, "${SshHost}:E:/B-HIS/start-codea-test-remote.ps1")
    $health = & $ssh -o BatchMode=yes -o ConnectTimeout=10 $SshHost 'powershell -NoProfile -ExecutionPolicy Bypass -File E:\B-HIS\start-codea-test-remote.ps1'
    if ($LASTEXITCODE -ne 0) { throw 'Code A Host startup failed' }
    Write-Host "Code A health: $($health -join ' ')"

    if (-not $KeepExistingTunnel) {
        if (Test-LocalPort $WebPort) { throw "local web port $WebPort is already occupied" }
        if (Test-LocalPort $ApiPort) { throw "local API port $ApiPort is already occupied" }
        $tunnelArgs = @('-o', 'BatchMode=yes', '-o', 'ExitOnForwardFailure=yes', '-o', 'ServerAliveInterval=30', '-o', 'ServerAliveCountMax=3', '-N', '-L', "${WebPort}:127.0.0.1:8888", '-L', "${ApiPort}:127.0.0.1:9999", $SshHost)
        $tunnel = Start-Process -FilePath $ssh -ArgumentList $tunnelArgs -WindowStyle Hidden -PassThru
        Wait-LocalPort $WebPort
        Wait-LocalPort $ApiPort
        Write-Host "SSH tunnel PID=$($tunnel.Id)"
    }

    $localHealth = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$ApiPort/health" -TimeoutSec 5
    if ($localHealth.StatusCode -ne 200) { throw "local tunnel health check failed: $($localHealth.StatusCode)" }
    $url = "http://127.0.0.1:$WebPort/login"
    Write-Host "Code A test site: $url"
    if (-not $NoOpen) { Start-Process $url | Out-Null }
} finally {
    Remove-Item -Recurse -Force $stage -ErrorAction SilentlyContinue
}
