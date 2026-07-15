$ErrorActionPreference = 'Stop'

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Invoke-Git {
    param(
        [string]$Repository,
        [string[]]$Arguments
    )

    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& git -C $Repository @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    if ($exitCode -ne 0) {
        throw "git command failed: $($output | Out-String)"
    }
    return $output
}

function Decode-FakeDockerCall {
    param([string]$Line)

    $payload = $Line.Substring(5)
    if ([string]::IsNullOrEmpty($payload)) {
        return @()
    }
    return @($payload.Split(';') | ForEach-Object {
        [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($_))
    })
}

function Invoke-RunnerProcess {
    param(
        [string]$Shell,
        [string[]]$Arguments
    )

    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $Shell @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    return [pscustomobject]@{
        ExitCode = $exitCode
        Output = $output
    }
}

$runner = Join-Path $PSScriptRoot 'invoke-revision-evidence.ps1'
$runnerSource = Get-Content -LiteralPath $runner -Raw -Encoding UTF8
$tokens = $null
$parseErrors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile($runner, [ref]$tokens, [ref]$parseErrors)
Assert-True ($parseErrors.Count -eq 0) "evidence runner has parse errors: $($parseErrors | Out-String)"
$parameterNames = @($ast.ParamBlock.Parameters | ForEach-Object { $_.Name.VariablePath.UserPath })
Assert-True ($parameterNames -notcontains 'ContainerImage') 'approved evidence image must not be caller-selectable'
Assert-True ($parameterNames -contains 'MaxSnapshotBytes') 'runner must expose a bounded snapshot size policy'

$approvedImage = 'docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a'
$imageReferences = @([regex]::Matches($runnerSource, '[^''"\s]+@sha256:[0-9a-f]{64}') | ForEach-Object Value | Sort-Object -Unique)
Assert-True ($imageReferences.Count -eq 1 -and $imageReferences[0] -eq $approvedImage) 'runner must contain exactly one approved image digest'
Assert-True ($runnerSource -match "'--entrypoint', '/usr/bin/env'") 'runner must override the image entrypoint explicitly'
Assert-True ($runnerSource -notmatch 'source=\$repoRoot,target=/repo') 'runner must not mount the host repository'
Assert-True ($runnerSource -match 'source=\$snapshotRoot,target=/evidence,readonly') 'runner must mount only the sanitized snapshot read-only'
Assert-True ($runnerSource.Contains('-WallClockTimeoutSeconds $TimeoutSeconds')) 'runner must apply its wall-clock timeout to docker run'
Assert-True ($runnerSource.Contains('-OutputLimitBytes $MaxOutputBytes')) 'runner must apply its bounded output limit to docker run'
Assert-True ($runnerSource.Contains("'snapshot-size-limit-exceeded'")) 'runner must fail closed when the snapshot exceeds its size bound'
Assert-True ($runnerSource.Contains("'rm', '--force', `$Name")) 'runner must force-remove the named container'

$testRoot = Join-Path ([IO.Path]::GetTempPath()) "allinme-evidence-runner-tests-$([guid]::NewGuid().ToString('N'))"
$fixtureRepo = Join-Path $testRoot 'repo'
$fakeBin = Join-Path $testRoot 'fake-bin'
$fakeDocker = Join-Path $fakeBin 'docker.exe'
$fakeLog = Join-Path $testRoot 'docker.log'
$previousPath = $env:PATH
$previousMode = $env:FAKE_DOCKER_MODE
$previousLog = $env:FAKE_DOCKER_LOG

try {
    New-Item -ItemType Directory -Path (Join-Path $fixtureRepo 'docs\tools') -Force | Out-Null
    New-Item -ItemType Directory -Path $fakeBin -Force | Out-Null
    Copy-Item -LiteralPath $runner -Destination (Join-Path $fixtureRepo 'docs\tools\invoke-revision-evidence.ps1')
    [IO.File]::WriteAllText((Join-Path $fixtureRepo '.gitignore'), "ignored-secret.txt`n", [Text.UTF8Encoding]::new($false))
    [IO.File]::WriteAllText((Join-Path $fixtureRepo 'tracked.txt'), "tracked snapshot content`n", [Text.UTF8Encoding]::new($false))
    [IO.File]::WriteAllText((Join-Path $fixtureRepo 'ignored-secret.txt'), "must never enter the snapshot`n", [Text.UTF8Encoding]::new($false))
    & git -C $fixtureRepo init --quiet
    if ($LASTEXITCODE -ne 0) { throw 'unable to initialize runner fixture repository' }
    & git -C $fixtureRepo config user.name evidence-runner-test
    & git -C $fixtureRepo config user.email evidence-runner-test@example.invalid
    Invoke-Git $fixtureRepo @('add', '.') | Out-Null
    Invoke-Git $fixtureRepo @('commit', '--quiet', '-m', 'fixture') | Out-Null

    $fakeDockerSource = @'
using System;
using System.IO;
using System.Linq;
using System.Text;
using System.Threading;

public static class Program
{
    private static void Log(string line)
    {
        File.AppendAllText(Environment.GetEnvironmentVariable("FAKE_DOCKER_LOG"), line + Environment.NewLine, new UTF8Encoding(false));
    }

    private static string Encode(string value)
    {
        return Convert.ToBase64String(Encoding.UTF8.GetBytes(value));
    }

    public static int Main(string[] args)
    {
        Log("CALL\t" + string.Join(";", args.Select(Encode)));
        if (args.Length == 0) return 2;
        if (args[0] == "version")
        {
            Console.Out.WriteLine("26.1.0");
            return 0;
        }
        if (args[0] == "image" && args.Length > 1 && args[1] == "inspect")
        {
            Console.Out.WriteLine("sha256:" + new string('a', 64));
            return 0;
        }
        if (args[0] == "rm") return 0;
        if (args[0] == "ps") return 0;
        if (args[0] != "run") return 2;

        string mount = args.FirstOrDefault(value => value.StartsWith("type=bind,source=", StringComparison.Ordinal));
        if (mount == null) return 3;
        int sourceStart = "type=bind,source=".Length;
        int targetStart = mount.IndexOf(",target=", StringComparison.Ordinal);
        if (targetStart <= sourceStart) return 4;
        string snapshotRoot = mount.Substring(sourceStart, targetStart - sourceStart);
        Log("MOUNT\t" + Encode(snapshotRoot));
        Log("FILES\t" + string.Join(",", Directory.GetFileSystemEntries(snapshotRoot).Select(Path.GetFileName).OrderBy(value => value)));
        Log("MANIFEST\t" + Convert.ToBase64String(File.ReadAllBytes(Path.Combine(snapshotRoot, "manifest.json"))));

        string mode = Environment.GetEnvironmentVariable("FAKE_DOCKER_MODE") ?? "success";
        if (mode == "timeout")
        {
            Thread.Sleep(10000);
            return 9;
        }
        if (mode == "output-limit")
        {
            Console.Out.Write(new string('x', 65536));
            Console.Out.Flush();
            Thread.Sleep(10000);
            return 9;
        }

        string markerArgument = args.FirstOrDefault(value => value.StartsWith("EVIDENCE_STATUS_MARKER=", StringComparison.Ordinal));
        if (markerArgument == null) return 5;
        string marker = markerArgument.Substring("EVIDENCE_STATUS_MARKER=".Length);
        Console.Out.Write("subject stdout\n");
        Console.Error.Write("\n" + marker + "true\n");
        return 0;
    }
}
'@
    Add-Type -TypeDefinition $fakeDockerSource -OutputAssembly $fakeDocker -OutputType ConsoleApplication

    $env:PATH = "$fakeBin;$previousPath"
    $env:FAKE_DOCKER_LOG = $fakeLog
    $shell = (Get-Command powershell.exe -ErrorAction Stop).Source
    $fixtureRunner = Join-Path $fixtureRepo 'docs\tools\invoke-revision-evidence.ps1'

    $successRunId = [guid]::NewGuid().ToString()
    $env:FAKE_DOCKER_MODE = 'success'
    $successResult = Invoke-RunnerProcess -Shell $shell -Arguments @(
        '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $fixtureRunner, '-Revision', 'HEAD', '-Command', 'git',
        '-CommandArgs', 'status', '-EvidenceRunId', $successRunId, '-TimeoutSeconds', '5', '-MaxOutputBytes', '4096'
    )
    Assert-True ($successResult.ExitCode -eq 0) "sanitized snapshot success fixture failed: $($successResult.Output | Out-String)"

    $successArtifactPath = Join-Path $fixtureRepo "docs\evidence\runs\$successRunId\evidence.json"
    $successArtifact = Get-Content -LiteralPath $successArtifactPath -Raw -Encoding UTF8 | ConvertFrom-Json
    Assert-True ($successArtifact.isolation.image -ceq $approvedImage) 'artifact did not bind the approved image digest'
    Assert-True ($successArtifact.isolation.host_repository_mounted -is [bool] -and -not $successArtifact.isolation.host_repository_mounted) 'artifact did not prove the host repository was absent'
    Assert-True ($successArtifact.isolation.snapshot_source -ceq 'git-archive-tar+manifest/v1') 'artifact did not identify the snapshot format'
    Assert-True ($successArtifact.isolation.snapshot_mount -ceq 'read-only') 'artifact did not prove a read-only snapshot mount'
    Assert-True ($successArtifact.isolation.entrypoint -ceq '/usr/bin/env') 'artifact did not bind the safe entrypoint'
    Assert-True ($successArtifact.isolation.timeout_seconds -eq 5 -and $successArtifact.isolation.max_output_bytes -eq 4096) 'artifact did not bind execution limits'
    Assert-True ($successArtifact.isolation.output_capture -ceq 'streaming-bounded' -and $successArtifact.isolation.failure_kind -ceq 'none') 'artifact did not identify successful bounded capture'
    Assert-True ($successArtifact.output.truncated -is [bool] -and -not $successArtifact.output.truncated -and $successArtifact.output.capture_complete) 'successful artifact incorrectly marked output incomplete'

    $logLines = @(Get-Content -LiteralPath $fakeLog -Encoding UTF8)
    $runCallLine = @($logLines | Where-Object { $_ -like 'CALL*' } | Where-Object {
        $decoded = @(Decode-FakeDockerCall $_)
        $decoded.Count -gt 0 -and $decoded[0] -eq 'run'
    }) | Select-Object -First 1
    Assert-True ($null -ne $runCallLine) 'fake docker did not observe docker run'
    $runArguments = @(Decode-FakeDockerCall $runCallLine)
    $entrypointIndex = [Array]::IndexOf($runArguments, '--entrypoint')
    $imageIndex = [Array]::IndexOf($runArguments, $approvedImage)
    Assert-True ($entrypointIndex -ge 0 -and $runArguments[$entrypointIndex + 1] -eq '/usr/bin/env' -and $entrypointIndex -lt $imageIndex) 'docker run did not override ENTRYPOINT before the approved image'
    Assert-True (@($runArguments | Where-Object { $_ -match 'target=/repo(?:,|$)' }).Count -eq 0) 'docker run exposed a host repository mount'
    Assert-True (@($runArguments | Where-Object { $_.IndexOf($fixtureRepo, [StringComparison]::OrdinalIgnoreCase) -ge 0 }).Count -eq 0) 'docker run arguments exposed the fixture worktree path'
    Assert-True (@($logLines | Where-Object { $_ -eq "FILES`tmanifest.json,source.tar" }).Count -gt 0) 'snapshot mount contained unexpected host files'

    $manifestLine = $logLines | Where-Object { $_ -like 'MANIFEST*' } | Select-Object -First 1
    $manifestJson = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($manifestLine.Substring(9)))
    $manifest = $manifestJson | ConvertFrom-Json
    $manifestPaths = @(for ($manifestIndex = 0; $manifestIndex -lt $manifest.Count; $manifestIndex++) {
        [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($manifest[$manifestIndex].path_base64))
    })
    Assert-True ($manifestPaths -contains 'tracked.txt') 'snapshot manifest omitted the committed fixture file'
    Assert-True ($manifestPaths -notcontains 'ignored-secret.txt') 'snapshot manifest leaked an ignored host file'
    Assert-True (@($manifestPaths | Where-Object { $_.Split('/') -contains '.git' }).Count -eq 0) 'snapshot manifest leaked a .git path'

    $mountLine = $logLines | Where-Object { $_ -like 'MOUNT*' } | Select-Object -First 1
    $snapshotPath = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($mountLine.Substring(6)))
    Assert-True (-not (Test-Path -LiteralPath $snapshotPath)) 'sanitized snapshot temp directory was not removed after success'

    $timeoutRunId = [guid]::NewGuid().ToString()
    $env:FAKE_DOCKER_MODE = 'timeout'
    $timeoutResult = Invoke-RunnerProcess -Shell $shell -Arguments @(
        '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $fixtureRunner, '-Revision', 'HEAD', '-Command', 'git',
        '-CommandArgs', 'status', '-EvidenceRunId', $timeoutRunId, '-TimeoutSeconds', '1', '-MaxOutputBytes', '4096'
    )
    Assert-True ($timeoutResult.ExitCode -eq 125) "timeout fixture did not fail closed: $($timeoutResult.Output | Out-String)"
    $timeoutArtifact = Get-Content -LiteralPath (Join-Path $fixtureRepo "docs\evidence\runs\$timeoutRunId\evidence.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    Assert-True ($timeoutArtifact.isolation.failure_kind -ceq 'wall-clock-timeout') 'timeout artifact did not record failure_kind'
    Assert-True ($timeoutArtifact.output.capture_complete -is [bool] -and -not $timeoutArtifact.output.capture_complete) 'timeout artifact incorrectly marked capture complete'

    $limitRunId = [guid]::NewGuid().ToString()
    $env:FAKE_DOCKER_MODE = 'output-limit'
    $limitResult = Invoke-RunnerProcess -Shell $shell -Arguments @(
        '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $fixtureRunner, '-Revision', 'HEAD', '-Command', 'git',
        '-CommandArgs', 'status', '-EvidenceRunId', $limitRunId, '-TimeoutSeconds', '5', '-MaxOutputBytes', '1024'
    )
    Assert-True ($limitResult.ExitCode -eq 125) "output-limit fixture did not fail closed: $($limitResult.Output | Out-String)"
    $limitArtifact = Get-Content -LiteralPath (Join-Path $fixtureRepo "docs\evidence\runs\$limitRunId\evidence.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    Assert-True ($limitArtifact.isolation.failure_kind -ceq 'output-limit-exceeded') 'output-limit artifact did not record failure_kind'
    Assert-True ($limitArtifact.output.truncated -is [bool] -and $limitArtifact.output.truncated) 'output-limit artifact did not record truncation'
    Assert-True ($limitArtifact.output.captured_bytes -eq 1024) 'bounded output capture retained more or less than its declared maximum'

    $allCalls = @(Get-Content -LiteralPath $fakeLog -Encoding UTF8 | Where-Object { $_ -like 'CALL*' } | ForEach-Object { ,@(Decode-FakeDockerCall $_) })
    $removeCalls = @($allCalls | Where-Object { $_.Count -gt 0 -and $_[0] -eq 'rm' })
    Assert-True ($removeCalls.Count -eq 3) 'runner did not clean the named container after every success/failure run'

    Write-Output 'invoke-revision-evidence tests passed.'
} finally {
    $env:PATH = $previousPath
    $env:FAKE_DOCKER_MODE = $previousMode
    $env:FAKE_DOCKER_LOG = $previousLog
    if (Test-Path -LiteralPath $testRoot) {
        $resolvedTestRoot = [IO.Path]::GetFullPath($testRoot)
        $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
        if (-not $resolvedTestRoot.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase) -or
            [IO.Path]::GetFileName($resolvedTestRoot) -notmatch '^allinme-evidence-runner-tests-[0-9a-f]{32}$') {
            throw "refusing to remove unexpected test directory: $resolvedTestRoot"
        }
        Remove-Item -LiteralPath $resolvedTestRoot -Recurse -Force
    }
}
