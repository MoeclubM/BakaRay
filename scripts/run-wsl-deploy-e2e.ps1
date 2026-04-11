param(
    [string]$Distro = "Debian",
    [string]$PanelRepo = "C:\Users\QwQ\Documents\GitHub\BakaRay",
    [string]$NodeRepo = "C:\Users\QwQ\Documents\GitHub\BakaRay-Node",
    [string]$GostVersion = "3.2.6",
    [switch]$KeepEnvironment
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
try {
    $PSNativeCommandUseErrorActionPreference = $false
} catch {
}

function Invoke-WslProcess {
    param(
        [string]$Command,
        [switch]$AllowFailure
    )

    $stdoutPath = Join-Path $env:TEMP ("bakaray-wsl-stdout-" + [Guid]::NewGuid().ToString("N") + ".log")
    $stderrPath = Join-Path $env:TEMP ("bakaray-wsl-stderr-" + [Guid]::NewGuid().ToString("N") + ".log")

    try {
        $process = Start-Process -FilePath wsl `
            -ArgumentList @('-d', $Distro, '-u', 'root', '--', 'bash', '-lc', $Command) `
            -RedirectStandardOutput $stdoutPath `
            -RedirectStandardError $stderrPath `
            -Wait `
            -NoNewWindow `
            -PassThru

        $stdout = if (Test-Path -LiteralPath $stdoutPath) {
            Get-Content -LiteralPath $stdoutPath -Raw -ErrorAction SilentlyContinue
        } else {
            ''
        }
        $stderr = if (Test-Path -LiteralPath $stderrPath) {
            Get-Content -LiteralPath $stderrPath -Raw -ErrorAction SilentlyContinue
        } else {
            ''
        }

        if (-not $AllowFailure -and $process.ExitCode -ne 0) {
            $details = $stderr.Trim()
            if (-not $details) {
                $details = $stdout.Trim()
            }
            if ($details) {
                throw "WSL command failed ($($process.ExitCode)): $Command`n$details"
            }
            throw "WSL command failed ($($process.ExitCode)): $Command"
        }

        return [PSCustomObject]@{
            ExitCode = $process.ExitCode
            Stdout   = $stdout
            Stderr   = $stderr
        }
    } finally {
        Remove-Item -LiteralPath $stdoutPath, $stderrPath -ErrorAction SilentlyContinue
    }
}

function Write-Step {
    param([string]$Message)

    Write-Host "==> $Message"
}

function Convert-ToWslPath {
    param([string]$WindowsPath)

    $normalized = $WindowsPath -replace "\\", "/"
    if ($normalized -match "^([A-Za-z]):/(.*)$") {
        return "/mnt/$($matches[1].ToLower())/$($matches[2])"
    }
    throw "Unsupported Windows path: $WindowsPath"
}

function Invoke-WslRoot {
    param([string]$Command)

    [void](Invoke-WslProcess -Command $Command)
}

function Invoke-WslBestEffort {
    param([string]$Command)

    return (Invoke-WslProcess -Command $Command -AllowFailure).ExitCode
}

function Start-WslBackgroundProcess {
    param(
        [string[]]$Arguments,
        [string]$StdoutPath,
        [string]$StderrPath
    )

    $fullArguments = @('-d', $Distro) + $Arguments
    return Start-Process -FilePath wsl `
        -ArgumentList $fullArguments `
        -RedirectStandardOutput $StdoutPath `
        -RedirectStandardError $StderrPath `
        -WindowStyle Hidden `
        -PassThru
}

function Stop-WslBackgroundProcesses {
    param([string[]]$Patterns)

    $processes = Get-CimInstance Win32_Process |
        Where-Object { $_.Name -eq 'wsl.exe' } |
        Where-Object {
            $commandLine = $_.CommandLine
            if (-not $commandLine) {
                return $false
            }

            foreach ($pattern in $Patterns) {
                if ($commandLine -like "*$pattern*") {
                    return $true
                }
            }
            return $false
        }

    foreach ($process in $processes) {
        Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
    }
}

function Stop-WindowsBackgroundProcesses {
    param([System.Diagnostics.Process[]]$Processes)

    foreach ($process in $Processes) {
        if ($null -eq $process) {
            continue
        }
        try {
            if (-not $process.HasExited) {
                Stop-Process -Id $process.Id -Force -ErrorAction Stop
            }
        } catch {
        }
    }
}

function Stop-WslRuntimeProcesses {
    param(
        [string]$RuntimeWsl,
        [string]$WslIp
    )

    $commands = @(
        "ss -ltnp '( sport = :19081 or sport = :18081 or sport = :18080 )' 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u | xargs -r kill -9 2>/dev/null || true",
        "pkill -f '$RuntimeWsl/node-current -c $RuntimeWsl/node1.yaml' || true",
        "pkill -f 'python3 -m http.server 19081 --bind 0.0.0.0 --directory $RuntimeWsl/target1' || true",
        "pkill -x gost || true"
    )

    [void](Invoke-WslBestEffort ($commands -join "; "))
}

function Show-LogFile {
    param(
        [string]$Path,
        [string]$Title,
        [int]$Tail = 80
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }

    $lines = @(Get-Content -LiteralPath $Path -Tail $Tail -ErrorAction SilentlyContinue)
    if ($lines.Count -eq 0) {
        return
    }

    Write-Host ""
    Write-Host "----- $Title -----"
    $lines | ForEach-Object { Write-Host $_ }
}

function Show-FailureContext {
    param(
        [string]$RuntimeWin,
        [string]$PanelRepoWsl
    )

    Show-LogFile -Path (Join-Path $RuntimeWin 'target1.out') -Title 'target1.out'
    Show-LogFile -Path (Join-Path $RuntimeWin 'target1.err') -Title 'target1.err'
    Show-LogFile -Path (Join-Path $RuntimeWin 'node1.out') -Title 'node1.out'
    Show-LogFile -Path (Join-Path $RuntimeWin 'node1.err') -Title 'node1.err'

    Write-Host ""
    Write-Host "----- docker ps -----"
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $dockerPs = & wsl -d $Distro -u root -- docker ps 2>&1
    $ErrorActionPreference = $previousErrorAction
    $dockerPs | ForEach-Object { Write-Host $_ }

    Write-Host ""
    Write-Host "----- bakaray-panel logs -----"
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $panelLogs = & wsl -d $Distro -u root -- bash -lc "cd '$PanelRepoWsl' && docker compose -f docker-compose.panel.yml logs --tail 120 bakaray-panel 2>&1 || true" 2>&1
    $ErrorActionPreference = $previousErrorAction
    $panelLogs | ForEach-Object { Write-Host $_ }
}

function Wait-HttpContent {
    param(
        [string]$Url,
        [string]$Expected,
        [int]$Retries = 60,
        [int]$DelaySeconds = 2
    )

    for ($i = 0; $i -lt $Retries; $i++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 5
            if ($response.Content -like "*$Expected*") {
                return
            }
        } catch {
        }
        Start-Sleep -Seconds $DelaySeconds
    }

    throw "Timed out waiting for $Url to contain '$Expected'"
}

function Wait-WslHttp {
    param(
        [string]$Url,
        [int]$Retries = 60,
        [int]$DelaySeconds = 2
    )

    for ($i = 0; $i -lt $Retries; $i++) {
        $previousErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        & wsl -d $Distro -u root -- curl -fsS $Url 1>$null 2>$null
        $curlExitCode = $LASTEXITCODE
        $ErrorActionPreference = $previousErrorAction

        if ($curlExitCode -eq 0) {
            return
        }
        Start-Sleep -Seconds $DelaySeconds
    }

    throw "Timed out waiting for WSL endpoint $Url"
}

$panelRepoWsl = Convert-ToWslPath $PanelRepo
$nodeRepoWsl = Convert-ToWslPath $NodeRepo
$runtimeWin = Join-Path $env:TEMP 'bakaray-e2e'
$runtimeWsl = Convert-ToWslPath $runtimeWin
$previousErrorAction = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
$hostnameOutput = (& wsl -d $Distro hostname -I 2>$null).Trim()
$ErrorActionPreference = $previousErrorAction
$wslIp = (($hostnameOutput -split '\s+') | Where-Object { $_ -match '^\d{1,3}(\.\d{1,3}){3}$' } | Select-Object -First 1)
$backgroundProcesses = [System.Collections.Generic.List[System.Diagnostics.Process]]::new()
$panelStackStarted = $false

if (-not $wslIp) {
    throw "Failed to resolve WSL IP"
}

try {
    Write-Step "Preparing runtime files"
    New-Item -ItemType Directory -Force -Path $runtimeWin, (Join-Path $runtimeWin 'target1') | Out-Null
    @'
backend-node-1
'@ | Set-Content -Path (Join-Path $runtimeWin 'target1\index.html') -NoNewline
    @'
panel:
  url: "http://127.0.0.1:8500"
  node_id: 1
  secret: "test-node-secret-change-in-production"
node:
  http_port: 18081
  report_interval: 5
  probe_interval: 5
  listen_ports:
    - 18080
logger:
  level: "info"
  output: "stdout"
'@ | Set-Content -Path (Join-Path $runtimeWin 'node1.yaml') -NoNewline

    $runtimeLogTarget1Out = Join-Path $runtimeWin 'target1.out'
    $runtimeLogTarget1Err = Join-Path $runtimeWin 'target1.err'
    $runtimeLogNode1Out = Join-Path $runtimeWin 'node1.out'
    $runtimeLogNode1Err = Join-Path $runtimeWin 'node1.err'

    Remove-Item $runtimeLogTarget1Out, $runtimeLogTarget1Err, $runtimeLogNode1Out, $runtimeLogNode1Err -ErrorAction SilentlyContinue

    Write-Step "Cleaning previous runtime"
    Stop-WslBackgroundProcesses @(
        'python3 -m http.server 19081',
        "$runtimeWsl/node-current -c $runtimeWsl/node1.yaml"
    )
    Stop-WslRuntimeProcesses -RuntimeWsl $runtimeWsl -WslIp $wslIp
    Invoke-WslRoot "rm -rf /tmp/gost && mkdir -p /tmp/gost; mkdir -p '$runtimeWsl/target1'"
    Invoke-WslRoot "cd '$panelRepoWsl' && docker compose -f docker-compose.panel.yml down -v --remove-orphans || true"

    Write-Step "Ensuring gost v3 binary is present"
    Invoke-WslRoot "test -x /tmp/gostv3/gost || (mkdir -p /tmp/gostv3 && curl -L --fail -o /tmp/gostv3/gost.tar.gz https://github.com/go-gost/gost/releases/download/v${GostVersion}/gost_${GostVersion}_linux_amd64.tar.gz && tar -xzf /tmp/gostv3/gost.tar.gz -C /tmp/gostv3 && chmod +x /tmp/gostv3/gost)"

    Write-Step "Building and starting panel stack"
    Invoke-WslRoot "cd '$panelRepoWsl' && docker build -t bakaray-panel:latest -f Dockerfile.panel . && docker compose -f docker-compose.panel.yml up -d"
    $panelStackStarted = $true

    Write-Step "Building current node binary"
    Invoke-WslRoot "docker run --rm -v '${nodeRepoWsl}:/src' -v '${runtimeWsl}:/out' -w /src golang:1.25 /bin/sh -c 'cd /src && /usr/local/go/bin/go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -buildvcs=false -o /out/node-current ./cmd/node'"

    Write-Step "Waiting for panel health endpoint"
    Wait-HttpContent -Url 'http://localhost:8500/health' -Expected '"status":"ok"'

    Write-Step "Starting WSL targets and nodes"
    $backgroundProcesses.Add((Start-WslBackgroundProcess -Arguments @('-u', 'root', '--', 'python3', '-m', 'http.server', '19081', '--bind', '0.0.0.0', '--directory', "$runtimeWsl/target1") -StdoutPath $runtimeLogTarget1Out -StderrPath $runtimeLogTarget1Err))
    $backgroundProcesses.Add((Start-WslBackgroundProcess -Arguments @('-u', 'root', '--', 'env', 'GOST_PATH=/tmp/gostv3/gost', "$runtimeWsl/node-current", '-c', "$runtimeWsl/node1.yaml") -StdoutPath $runtimeLogNode1Out -StderrPath $runtimeLogNode1Err))

    Write-Step "Waiting for target and node health endpoints"
    Wait-WslHttp -Url 'http://127.0.0.1:19081'
    Wait-WslHttp -Url 'http://127.0.0.1:18081/health'

    Write-Step "Running Playwright deploy E2E"
    $env:WSL_HOST_IP = $wslIp
    $env:BAKARAY_NODE_SECRET = 'test-node-secret-change-in-production'
    $env:BAKARAY_PANEL_BASE_URL = 'http://localhost:8500'
    $env:BAKARAY_API_BASE_URL = 'http://localhost:8500/api'

    & npx playwright test tests/wsl-deploy-e2e.spec.ts --reporter=list
    if ($LASTEXITCODE -ne 0) {
        throw "Playwright E2E failed with exit code $LASTEXITCODE"
    }

    Write-Step "WSL deploy E2E passed"
} catch {
    Write-Host ""
    Write-Host "WSL deploy E2E failed: $($_.Exception.Message)"
    Show-FailureContext -RuntimeWin $runtimeWin -PanelRepoWsl $panelRepoWsl
    exit 1
} finally {
    Remove-Item Env:WSL_HOST_IP, Env:BAKARAY_NODE_SECRET, Env:BAKARAY_PANEL_BASE_URL, Env:BAKARAY_API_BASE_URL -ErrorAction SilentlyContinue

    if ($KeepEnvironment) {
        Write-Step "Keeping runtime environment for inspection"
    } else {
        Write-Step "Cleaning up runtime environment"
        Stop-WindowsBackgroundProcesses -Processes $backgroundProcesses.ToArray()
        Stop-WslBackgroundProcesses @(
            'python3 -m http.server 19081',
            "$runtimeWsl/node-current -c $runtimeWsl/node1.yaml"
        )
        Stop-WslRuntimeProcesses -RuntimeWsl $runtimeWsl -WslIp $wslIp
        if ($panelStackStarted) {
            [void](Invoke-WslBestEffort "cd '$panelRepoWsl' && docker compose -f docker-compose.panel.yml down -v --remove-orphans || true")
        }
    }
}
