param(
    [Parameter(Mandatory = $true)]
    [string]$Revision,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$Command,

    [string[]]$CommandArgs = @(),

    [ValidatePattern('^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$')]
    [string]$EvidenceRunId = [guid]::NewGuid().ToString(),

    [ValidateRange(64, 4096)]
    [int]$MemoryMegabytes = 1024,

    [ValidateRange(0.1, 4.0)]
    [double]$CpuLimit = 1.0,

    [ValidateRange(16, 1024)]
    [int]$PidsLimit = 256,

    [ValidateRange(1, 3600)]
    [int]$TimeoutSeconds = 900,

    [ValidateRange(1024, 16777216)]
    [long]$MaxOutputBytes = 4194304,

    [ValidateRange(1048576, 4294967296)]
    [long]$MaxSnapshotBytes = 536870912
)

$ErrorActionPreference = 'Stop'
$OutputEncoding = [Text.UTF8Encoding]::new($false)

# This is the only repository-approved evidence image. It is intentionally not a parameter.
$approvedContainerImage = 'docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a'
$infrastructureTimeoutSeconds = 60
$infrastructureOutputLimit = 1048576

function Get-Sha256Hex {
    param([byte[]]$Bytes)

    [byte[]]$hashBytes = $Bytes
    if ($null -eq $hashBytes) {
        $hashBytes = [byte[]]::new(0)
    }
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha256.ComputeHash($hashBytes))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha256.Dispose()
    }
}

function Get-FileSha256Hex {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $stream = [IO.File]::Open($Path, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha256.ComputeHash($stream))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha256.Dispose()
        $stream.Dispose()
    }
}

function ConvertTo-NativeArgumentString {
    param([string[]]$Arguments)

    return (($Arguments | ForEach-Object {
        if ($_.Length -eq 0) {
            '""'
        } elseif ($_ -notmatch '[\s"]') {
            $_
        } else {
            $quoted = New-Object Text.StringBuilder
            [void]$quoted.Append([char]34)
            $backslashCount = 0
            foreach ($character in $_.ToCharArray()) {
                if ($character -eq [char]92) {
                    $backslashCount++
                    continue
                }
                if ($character -eq [char]34) {
                    [void]$quoted.Append([char]92, (2 * $backslashCount) + 1)
                    [void]$quoted.Append([char]34)
                    $backslashCount = 0
                    continue
                }
                if ($backslashCount -gt 0) {
                    [void]$quoted.Append([char]92, $backslashCount)
                    $backslashCount = 0
                }
                [void]$quoted.Append($character)
            }
            if ($backslashCount -gt 0) {
                [void]$quoted.Append([char]92, 2 * $backslashCount)
            }
            [void]$quoted.Append([char]34)
            $quoted.ToString()
        }
    }) -join ' ')
}

function Invoke-NativeCapture {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [Parameter(Mandatory = $true)]
        [string[]]$Arguments,

        [ValidateRange(1, 3600)]
        [int]$WallClockTimeoutSeconds = $infrastructureTimeoutSeconds,

        [ValidateRange(1024, 16777216)]
        [long]$OutputLimitBytes = $infrastructureOutputLimit
    )

    $startInfo = New-Object Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.CreateNoWindow = $true
    $startInfo.Arguments = ConvertTo-NativeArgumentString $Arguments

    $process = New-Object Diagnostics.Process
    $process.StartInfo = $startInfo
    $stdoutCapture = New-Object IO.MemoryStream
    $stderrCapture = New-Object IO.MemoryStream
    try {
        if (-not $process.Start()) {
            throw "Unable to start native command: $FilePath"
        }

        $stdoutBuffer = New-Object byte[] 8192
        $stderrBuffer = New-Object byte[] 8192
        $stdoutOpen = $true
        $stderrOpen = $true
        $stdoutTask = $process.StandardOutput.BaseStream.ReadAsync($stdoutBuffer, 0, $stdoutBuffer.Length)
        $stderrTask = $process.StandardError.BaseStream.ReadAsync($stderrBuffer, 0, $stderrBuffer.Length)
        $capturedBytes = [long]0
        $observedBytesAtLeast = [long]0
        $timedOut = $false
        $outputLimitExceeded = $false
        $readFailure = $null
        $stopwatch = [Diagnostics.Stopwatch]::StartNew()

        while ($true) {
            if ($stdoutOpen -and $stdoutTask.IsCompleted) {
                try {
                    $count = $stdoutTask.GetAwaiter().GetResult()
                } catch {
                    $readFailure = "stdout capture failed: $($_.Exception.Message)"
                    $count = 0
                }
                if ($null -ne $readFailure -or $count -eq 0) {
                    $stdoutOpen = $false
                } else {
                    $observedBytesAtLeast += $count
                    $remaining = [Math]::Max([long]0, $OutputLimitBytes - $capturedBytes)
                    $writeCount = [int][Math]::Min([long]$count, $remaining)
                    if ($writeCount -gt 0) {
                        $stdoutCapture.Write($stdoutBuffer, 0, $writeCount)
                        $capturedBytes += $writeCount
                    }
                    if ($writeCount -lt $count) {
                        $outputLimitExceeded = $true
                    } else {
                        $stdoutTask = $process.StandardOutput.BaseStream.ReadAsync($stdoutBuffer, 0, $stdoutBuffer.Length)
                    }
                }
            }

            if ($stderrOpen -and $stderrTask.IsCompleted -and -not $outputLimitExceeded) {
                try {
                    $count = $stderrTask.GetAwaiter().GetResult()
                } catch {
                    $readFailure = "stderr capture failed: $($_.Exception.Message)"
                    $count = 0
                }
                if ($null -ne $readFailure -or $count -eq 0) {
                    $stderrOpen = $false
                } else {
                    $observedBytesAtLeast += $count
                    $remaining = [Math]::Max([long]0, $OutputLimitBytes - $capturedBytes)
                    $writeCount = [int][Math]::Min([long]$count, $remaining)
                    if ($writeCount -gt 0) {
                        $stderrCapture.Write($stderrBuffer, 0, $writeCount)
                        $capturedBytes += $writeCount
                    }
                    if ($writeCount -lt $count) {
                        $outputLimitExceeded = $true
                    } else {
                        $stderrTask = $process.StandardError.BaseStream.ReadAsync($stderrBuffer, 0, $stderrBuffer.Length)
                    }
                }
            }

            if ($outputLimitExceeded -or $null -ne $readFailure) {
                break
            }
            if ($stopwatch.Elapsed.TotalSeconds -ge $WallClockTimeoutSeconds) {
                $timedOut = $true
                break
            }
            if ($process.HasExited -and -not $stdoutOpen -and -not $stderrOpen) {
                break
            }
            [Threading.Thread]::Sleep(5)
        }

        $stopwatch.Stop()
        if (($timedOut -or $outputLimitExceeded -or $null -ne $readFailure) -and -not $process.HasExited) {
            try {
                $process.Kill()
            } catch {
                # The caller separately removes a named Docker container when this is docker run.
            }
        }
        if (-not $process.HasExited) {
            [void]$process.WaitForExit(5000)
        }

        $stdoutBytes = $stdoutCapture.ToArray()
        $stderrBytes = $stderrCapture.ToArray()
        $utf8 = [Text.UTF8Encoding]::new($false)
        $exitCode = if ($process.HasExited) { [int]$process.ExitCode } else { 125 }
        return [pscustomobject]@{
            ExitCode = $exitCode
            Stdout = $utf8.GetString($stdoutBytes)
            Stderr = $utf8.GetString($stderrBytes)
            StdoutBytes = $stdoutBytes
            StderrBytes = $stderrBytes
            CapturedBytes = [long]($stdoutBytes.Length + $stderrBytes.Length)
            ObservedBytesAtLeast = $observedBytesAtLeast
            TimedOut = $timedOut
            OutputLimitExceeded = $outputLimitExceeded
            ReadFailure = $readFailure
            DurationMilliseconds = [long]$stopwatch.ElapsedMilliseconds
        }
    } finally {
        try { $process.StandardOutput.Close() } catch {}
        try { $process.StandardError.Close() } catch {}
        $stdoutCapture.Dispose()
        $stderrCapture.Dispose()
        $process.Dispose()
    }
}

function Assert-NativeCaptureSucceeded {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Result,

        [Parameter(Mandatory = $true)]
        [string]$Description
    )

    if ($Result.TimedOut) {
        throw "$Description timed out."
    }
    if ($Result.OutputLimitExceeded) {
        throw "$Description exceeded its output limit."
    }
    if ($null -ne $Result.ReadFailure) {
        throw "$Description output capture failed: $($Result.ReadFailure)"
    }
    if ($Result.ExitCode -ne 0) {
        throw "$Description failed: $($Result.Stderr.Trim())"
    }
}

function Invoke-GitText {
    param([string[]]$Arguments)

    $result = Invoke-NativeCapture -FilePath $gitExecutable -Arguments $Arguments
    Assert-NativeCaptureSucceeded -Result $result -Description 'git command'
    return $result.Stdout.Trim()
}

function Write-EvidenceArtifact {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Directory,

        [Parameter(Mandatory = $true)]
        [string]$Path,

        [Parameter(Mandatory = $true)]
        [object]$Metadata
    )

    New-Item -ItemType Directory -Path $Directory | Out-Null
    $json = $Metadata | ConvertTo-Json -Depth 10
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes($json + [Environment]::NewLine)
    $stream = [IO.File]::Open($Path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        $stream.Write($bytes, 0, $bytes.Length)
        $stream.Flush()
    } finally {
        $stream.Dispose()
    }
}

function Write-Utf8JsonFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,

        [Parameter(Mandatory = $true)]
        [object]$Value
    )

    $json = ConvertTo-Json -InputObject $Value -Compress -Depth 6
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes($json + [Environment]::NewLine)
    $stream = [IO.File]::Open($Path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        $stream.Write($bytes, 0, $bytes.Length)
        $stream.Flush()
    } finally {
        $stream.Dispose()
    }
}

function Convert-LsTreeBytesToManifest {
    param([byte[]]$Bytes)

    $entries = New-Object Collections.Generic.List[object]
    $totalBytes = [long]0
    $start = 0
    for ($index = 0; $index -lt $Bytes.Length; $index++) {
        if ($Bytes[$index] -ne 0) {
            continue
        }
        if ($index -eq $start) {
            $start = $index + 1
            continue
        }

        $tab = -1
        for ($cursor = $start; $cursor -lt $index; $cursor++) {
            if ($Bytes[$cursor] -eq 9) {
                $tab = $cursor
                break
            }
        }
        if ($tab -lt 0) {
            throw 'git ls-tree returned a malformed record.'
        }

        $header = [Text.Encoding]::ASCII.GetString($Bytes, $start, $tab - $start)
        if ($header -notmatch '^(?<mode>[0-9]{6}) (?<type>[^ ]+) (?<oid>[0-9a-f]{40})\s+(?<size>[0-9]+)$') {
            throw "git ls-tree returned an unsupported record header: $header"
        }
        if ($Matches['type'] -ne 'blob' -or $Matches['mode'] -notin @('100644', '100755', '120000')) {
            throw "Evidence snapshots do not support tree entry $header; submodules and special entries must be audited separately."
        }

        $pathLength = $index - $tab - 1
        if ($pathLength -le 0) {
            throw 'git ls-tree returned an empty path.'
        }
        $pathBytes = New-Object byte[] $pathLength
        [Array]::Copy($Bytes, $tab + 1, $pathBytes, 0, $pathLength)
        $pathForSafetyCheck = [Text.Encoding]::ASCII.GetString($pathBytes).ToLowerInvariant()
        if ($pathForSafetyCheck.Split('/') -contains '.git') {
            throw 'Evidence snapshots refuse a tree containing a .git path component.'
        }
        $entries.Add([ordered]@{
            mode = $Matches['mode']
            oid = $Matches['oid']
            path_base64 = [Convert]::ToBase64String($pathBytes)
        })
        try {
            $entryBytes = [Convert]::ToInt64($Matches['size'], 10)
            $candidateTotal = [decimal]$totalBytes + [decimal]$entryBytes
            if ($candidateTotal -gt [long]::MaxValue) { throw 'overflow' }
            $totalBytes = [long]$candidateTotal
        } catch {
            throw 'Evidence snapshot size exceeds the supported signed 64-bit range.'
        }
        $start = $index + 1
    }

    if ($start -ne $Bytes.Length) {
        throw 'git ls-tree output was not NUL terminated.'
    }
    return [pscustomobject]@{
        Entries = $entries.ToArray()
        TotalBytes = $totalBytes
    }
}

function Get-StatusMarkerResult {
    param(
        [Parameter(Mandatory = $true)]
        [AllowEmptyCollection()]
        [byte[]]$StderrBytes,

        [Parameter(Mandatory = $true)]
        [string]$Marker
    )

    $end = $StderrBytes.Length
    if ($end -gt 0 -and $StderrBytes[$end - 1] -eq 10) {
        $end--
        if ($end -gt 0 -and $StderrBytes[$end - 1] -eq 13) {
            $end--
        }
    }
    foreach ($value in @('true', 'false')) {
        $candidate = [Text.Encoding]::UTF8.GetBytes($Marker + $value)
        $start = $end - $candidate.Length
        if ($start -lt 0) {
            continue
        }
        $matches = $true
        for ($index = 0; $index -lt $candidate.Length; $index++) {
            if ($StderrBytes[$start + $index] -ne $candidate[$index]) {
                $matches = $false
                break
            }
        }
        if (-not $matches -or ($start -gt 0 -and $StderrBytes[$start - 1] -ne 10)) {
            continue
        }

        $cleanLength = $start
        if ($cleanLength -gt 0 -and $StderrBytes[$cleanLength - 1] -eq 10) {
            $cleanLength--
            if ($cleanLength -gt 0 -and $StderrBytes[$cleanLength - 1] -eq 13) {
                $cleanLength--
            }
        }
        $cleanBytes = New-Object byte[] $cleanLength
        if ($cleanLength -gt 0) {
            [Array]::Copy($StderrBytes, 0, $cleanBytes, 0, $cleanLength)
        }
        return [pscustomobject]@{
            Success = $true
            SubjectClean = ($value -eq 'true')
            CleanBytes = $cleanBytes
        }
    }

    return [pscustomobject]@{
        Success = $false
        SubjectClean = $false
        CleanBytes = [byte[]]@()
    }
}

function Remove-SnapshotDirectory {
    param([string]$Path)

    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path)) {
        return
    }
    $resolvedTarget = [IO.Path]::GetFullPath($Path).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar)
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar)
    $requiredPrefix = $resolvedTemp + [IO.Path]::DirectorySeparatorChar
    if (-not $resolvedTarget.StartsWith($requiredPrefix, [StringComparison]::OrdinalIgnoreCase) -or
        [IO.Path]::GetFileName($resolvedTarget) -notmatch '^allinme-evidence-snapshot-[0-9a-f]{32}$') {
        throw "Refusing to remove an unexpected snapshot path: $resolvedTarget"
    }
    Remove-Item -LiteralPath $resolvedTarget -Recurse -Force
}

function Remove-EvidenceContainer {
    param(
        [Parameter(Mandatory = $true)]
        [string]$DockerPath,

        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $remove = Invoke-NativeCapture -FilePath $DockerPath -Arguments @('rm', '--force', $Name) -WallClockTimeoutSeconds 30 -OutputLimitBytes 65536
    if (-not $remove.TimedOut -and -not $remove.OutputLimitExceeded -and $null -eq $remove.ReadFailure -and $remove.ExitCode -eq 0) {
        return [pscustomobject]@{ Succeeded = $true; Message = '' }
    }

    $probe = Invoke-NativeCapture -FilePath $DockerPath -Arguments @(
        'ps', '--all', '--filter', "name=^/$Name`$", '--format', '{{.ID}}'
    ) -WallClockTimeoutSeconds 15 -OutputLimitBytes 65536
    if (-not $probe.TimedOut -and -not $probe.OutputLimitExceeded -and $null -eq $probe.ReadFailure -and
        $probe.ExitCode -eq 0 -and [string]::IsNullOrWhiteSpace($probe.Stdout)) {
        return [pscustomobject]@{ Succeeded = $true; Message = '' }
    }

    $message = @($remove.Stderr.Trim(), $probe.Stderr.Trim()) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    return [pscustomobject]@{
        Succeeded = $false
        Message = if ($message.Count -gt 0) { $message -join ' / ' } else { 'unable to prove that the named evidence container was removed' }
    }
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$gitCommand = Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath($gitCommand.Source)
$repoPathPrefix = $repoRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
if ($gitExecutable.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-local Git binary: $gitExecutable"
}
$revisionToken = ($Revision -replace '^git:', '').Split(';')[0].Trim()
$resolvedRevision = Invoke-GitText @('-C', $repoRoot, 'rev-parse', '--verify', "$revisionToken^{commit}")
if ($resolvedRevision -notmatch '^[0-9a-f]{40}$') {
    throw "Revision does not resolve to a full commit: $Revision"
}

$resolvedTree = Invoke-GitText @('-C', $repoRoot, 'rev-parse', "$resolvedRevision^{tree}")
if ($resolvedTree -notmatch '^[0-9a-f]{40}$') {
    throw "Revision does not resolve to a full tree: $Revision"
}

$hostStatusBefore = Invoke-GitText @(
    '-C', $repoRoot, 'status', '--porcelain=v1', '--untracked-files=all', '--', '.',
    ':(exclude)docs/evidence/runs/**'
)
if (-not [string]::IsNullOrEmpty($hostStatusBefore)) {
    [Console]::Error.WriteLine('Host worktree is dirty; refusing to run or create an evidence artifact.')
    exit 125
}

$runId = $EvidenceRunId.ToLowerInvariant()
$artifactDirectory = Join-Path $repoRoot (Join-Path 'docs\evidence\runs' $runId)
$artifactPath = Join-Path $artifactDirectory 'evidence.json'
if (Test-Path -LiteralPath $artifactDirectory) {
    throw "Evidence run ID already exists and is immutable: $runId"
}

$argv = @($Command) + @($CommandArgs)
$argvJson = ConvertTo-Json -Compress -InputObject @($argv)
$argvBase64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($argvJson))
$startedAt = [DateTimeOffset]::UtcNow
$containerName = "allinme-evidence-$($runId.Replace('-', ''))"
$statusMarker = "__ALLINME_EVIDENCE_STATUS_$($runId.Replace('-', '').ToUpperInvariant())__="
$snapshotRoot = $null
$snapshotArchiveSha256 = $null
$snapshotManifestSha256 = $null
$snapshotManifestEntries = [long]0
$snapshotContentBytes = [long]0
$imageId = $null
$preflightComplete = $false

$writeFailure = {
    param(
        [string]$FailureKind,
        [string]$FailureMessage,
        [bool]$PreflightPassed = $false,
        [object]$CaptureResult = $null,
        [bool]$SnapshotCreated = $false
    )

    $completedAt = [DateTimeOffset]::UtcNow
    $stdoutBytes = if ($null -ne $CaptureResult) { [byte[]]$CaptureResult.StdoutBytes } else { [byte[]]@() }
    $stderrBytes = if ($null -ne $CaptureResult) { [byte[]]$CaptureResult.StderrBytes } else { [byte[]]@() }
    $combined = New-Object byte[] ($stdoutBytes.Length + $stderrBytes.Length)
    if ($stdoutBytes.Length -gt 0) { [Array]::Copy($stdoutBytes, 0, $combined, 0, $stdoutBytes.Length) }
    if ($stderrBytes.Length -gt 0) { [Array]::Copy($stderrBytes, 0, $combined, $stdoutBytes.Length, $stderrBytes.Length) }
    $hostStatusAfter = ''
    try {
        $hostStatusAfter = Invoke-GitText @(
            '-C', $repoRoot, 'status', '--porcelain=v1', '--untracked-files=all', '--', '.',
            ':(exclude)docs/evidence/runs/**'
        )
    } catch {
        $hostStatusAfter = '<status-check-failed>'
    }
    $truncated = ($null -ne $CaptureResult -and $CaptureResult.OutputLimitExceeded)
    $metadata = [ordered]@{
        schema = 'revision-evidence/v1'
        evidence_run_id = $runId
        evidence_revision = $resolvedRevision
        evidence_tree = $resolvedTree
        evidence_worktree = if ($SnapshotCreated) { 'detached' } else { 'not-created' }
        argv = $argv
        exit_code = 125
        isolation = [ordered]@{
            engine = 'docker'
            image = $approvedContainerImage
            image_id = $imageId
            approved_image = $true
            entrypoint = '/usr/bin/env'
            network = 'none'
            repository_mount = 'read-only'
            host_repository_mounted = $false
            snapshot_source = 'git-archive-tar+manifest/v1'
            snapshot_mount = 'read-only'
            snapshot_archive_sha256 = $snapshotArchiveSha256
            snapshot_manifest_sha256 = $snapshotManifestSha256
            snapshot_manifest_entries = $snapshotManifestEntries
            snapshot_content_bytes = $snapshotContentBytes
            max_snapshot_bytes = $MaxSnapshotBytes
            root_filesystem = 'read-only'
            capabilities = 'none'
            no_new_privileges = $true
            user = '65534:65534'
            memory_megabytes = $MemoryMegabytes
            cpus = $CpuLimit
            pids_limit = $PidsLimit
            timeout_seconds = $TimeoutSeconds
            max_output_bytes = $MaxOutputBytes
            output_capture = 'streaming-bounded'
            sanitized_environment = $true
            preflight_passed = $PreflightPassed
            failure_kind = $FailureKind
            failure_message_sha256 = Get-Sha256Hex ([Text.Encoding]::UTF8.GetBytes($FailureMessage))
        }
        output = [ordered]@{
            stdout_sha256 = Get-Sha256Hex $stdoutBytes
            stderr_sha256 = Get-Sha256Hex $stderrBytes
            combined_sha256 = Get-Sha256Hex $combined
            stdout_bytes = $stdoutBytes.Length
            stderr_bytes = $stderrBytes.Length
            captured_bytes = [long]($stdoutBytes.Length + $stderrBytes.Length)
            observed_bytes_at_least = if ($null -ne $CaptureResult) { [long]$CaptureResult.ObservedBytesAtLeast } else { [long]0 }
            truncated = [bool]$truncated
            capture_complete = [bool](-not $truncated -and ($null -eq $CaptureResult -or -not $CaptureResult.TimedOut))
        }
        clean_status = [ordered]@{
            host_tracked_clean_before_run = $true
            host_tracked_clean_after_run = [string]::IsNullOrEmpty($hostStatusAfter)
            host_tracked_state_unchanged = [string]::IsNullOrEmpty($hostStatusAfter)
            subject_tracked_clean_before_run = $false
            subject_tracked_clean_after_run = $false
            subject_workspace_discarded = $true
        }
        tracked_worktree_clean_after_run = $false
        started_at = $startedAt.ToString('o')
        completed_at = $completedAt.ToString('o')
    }
    Write-EvidenceArtifact -Directory $artifactDirectory -Path $artifactPath -Metadata $metadata
    [Console]::Error.WriteLine($FailureMessage)
    Write-Output '--- evidence metadata ---'
    Write-Output ($metadata | ConvertTo-Json -Compress -Depth 10)
    Write-Output "evidence_artifact: docs/evidence/runs/$runId/evidence.json"
    exit 125
}

try {
    $dockerCommand = @(Get-Command docker -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
    if ($null -eq $dockerCommand -or [string]::IsNullOrWhiteSpace($dockerCommand.Source)) {
        & $writeFailure 'docker-cli-unavailable' 'Docker CLI is unavailable; refusing to run unisolated evidence.'
        return
    }
    $dockerExecutable = [IO.Path]::GetFullPath($dockerCommand.Source)
    if ($dockerExecutable.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        & $writeFailure 'docker-cli-untrusted' "Refusing to execute a repository-local Docker binary: $dockerExecutable"
    }

    $dockerVersion = Invoke-NativeCapture -FilePath $dockerExecutable -Arguments @('version', '--format', '{{.Server.Version}}')
    if ($dockerVersion.TimedOut -or $dockerVersion.OutputLimitExceeded -or $null -ne $dockerVersion.ReadFailure -or
        $dockerVersion.ExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($dockerVersion.Stdout)) {
        & $writeFailure 'docker-daemon-unavailable' "Docker daemon is unavailable; refusing to run unisolated evidence. $($dockerVersion.Stderr.Trim())" $false $dockerVersion
    }

    $imageInspect = Invoke-NativeCapture -FilePath $dockerExecutable -Arguments @('image', 'inspect', $approvedContainerImage, '--format', '{{.Id}}')
    if ($imageInspect.TimedOut -or $imageInspect.OutputLimitExceeded -or $null -ne $imageInspect.ReadFailure -or
        $imageInspect.ExitCode -ne 0 -or $imageInspect.Stdout.Trim() -notmatch '^sha256:[0-9a-f]{64}$') {
        & $writeFailure 'pinned-image-unavailable' "Pinned container image is unavailable locally; refusing to pull during an evidence run: $approvedContainerImage" $false $imageInspect
    }
    $imageId = $imageInspect.Stdout.Trim()

    $snapshotRoot = Join-Path ([IO.Path]::GetTempPath()) "allinme-evidence-snapshot-$([guid]::NewGuid().ToString('N'))"
    [void][IO.Directory]::CreateDirectory($snapshotRoot)
    $snapshotArchivePath = Join-Path $snapshotRoot 'source.tar'
    $snapshotManifestPath = Join-Path $snapshotRoot 'manifest.json'

    $treeListing = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @(
        '-C', $repoRoot, 'ls-tree', '-r', '-l', '-z', '--full-tree', $resolvedRevision
    ) -WallClockTimeoutSeconds $infrastructureTimeoutSeconds -OutputLimitBytes 16777216
    Assert-NativeCaptureSucceeded -Result $treeListing -Description 'git snapshot manifest generation'
    $snapshotManifestResult = Convert-LsTreeBytesToManifest $treeListing.StdoutBytes
    $snapshotManifest = @($snapshotManifestResult.Entries)
    $snapshotManifestEntries = [long]$snapshotManifest.Count
    $snapshotContentBytes = [long]$snapshotManifestResult.TotalBytes
    $estimatedArchiveBytes = [decimal]$snapshotContentBytes + ([decimal]$snapshotManifestEntries * 4096) + 1048576
    if ($estimatedArchiveBytes -gt $MaxSnapshotBytes) {
        & $writeFailure 'snapshot-size-limit-exceeded' "Evidence snapshot estimate exceeds the $MaxSnapshotBytes byte limit." $false
    }
    Write-Utf8JsonFile -Path $snapshotManifestPath -Value $snapshotManifest

    $archive = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @(
        '-C', $repoRoot, 'archive', '--format=tar', "--output=$snapshotArchivePath", $resolvedRevision
    ) -WallClockTimeoutSeconds $infrastructureTimeoutSeconds -OutputLimitBytes $infrastructureOutputLimit
    Assert-NativeCaptureSucceeded -Result $archive -Description 'git snapshot archive generation'
    $snapshotArchiveItem = if (Test-Path -LiteralPath $snapshotArchivePath -PathType Leaf) { Get-Item -LiteralPath $snapshotArchivePath } else { $null }
    if ($null -eq $snapshotArchiveItem -or $snapshotArchiveItem.Length -le 0) {
        & $writeFailure 'snapshot-generation-failed' 'Git did not produce a non-empty evidence snapshot archive.' $false $archive
    }
    if ($snapshotArchiveItem.Length -gt $MaxSnapshotBytes) {
        & $writeFailure 'snapshot-size-limit-exceeded' "Evidence snapshot archive exceeds the $MaxSnapshotBytes byte limit." $false $archive $true
    }
    $snapshotArchiveSha256 = Get-FileSha256Hex $snapshotArchivePath
    $snapshotManifestSha256 = Get-FileSha256Hex $snapshotManifestPath
    $preflightComplete = $true

    $containerScript = @'
set -eu
mkdir -p /tmp/workspace /tmp/home /tmp/go-cache /tmp/go-path
export HOME=/tmp/home
export GOCACHE=/tmp/go-cache
export GOPATH=/tmp/go-path
export GOMODCACHE=/tmp/go-path/pkg/mod
export GOENV=off
export GOTOOLCHAIN=local
export GOPROXY=off
export GOSUMDB=off
unset SSH_AUTH_SOCK GIT_ASKPASS GIT_SSH GIT_SSH_COMMAND GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM

python3 - /evidence/source.tar "$EVIDENCE_SNAPSHOT_SHA256" /evidence/manifest.json "$EVIDENCE_MANIFEST_SHA256" <<'PY'
import hashlib
import sys

for path, expected in ((sys.argv[1], sys.argv[2]), (sys.argv[3], sys.argv[4])):
    digest = hashlib.sha256()
    with open(path, "rb") as stream:
        while True:
            chunk = stream.read(1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)
    if digest.hexdigest() != expected:
        raise SystemExit("snapshot input digest mismatch")
PY

tar --extract --file /evidence/source.tar --directory /tmp/workspace --no-same-owner

verify_snapshot() {
    python3 - "$1" /evidence/manifest.json <<'PY'
import base64
import hashlib
import json
import os
import stat
import sys

root = os.fsencode(sys.argv[1])
with open(sys.argv[2], "r", encoding="utf-8") as stream:
    manifest = json.load(stream)
if not isinstance(manifest, list):
    raise SystemExit(1)

expected = {}
for entry in manifest:
    if not isinstance(entry, dict) or set(entry) != {"mode", "oid", "path_base64"}:
        raise SystemExit(1)
    path = base64.b64decode(entry["path_base64"], validate=True)
    if not path or path.startswith(b"/") or b"\x00" in path or b"\\" in path.split(b"/"):
        raise SystemExit(1)
    if path in expected:
        raise SystemExit(1)
    expected[path] = (entry["mode"], entry["oid"])

actual = {}
for current, directories, files in os.walk(root, topdown=True, followlinks=False):
    directories[:] = [name for name in directories if not os.path.islink(os.path.join(current, name))]
    names = files + [name for name in os.listdir(current) if os.path.islink(os.path.join(current, name))]
    for name in names:
        full_path = os.path.join(current, name)
        relative = os.path.relpath(full_path, root).replace(os.fsencode(os.sep), b"/")
        actual[relative] = full_path
if set(actual) != set(expected):
    raise SystemExit(1)

for path, full_path in actual.items():
    mode, expected_oid = expected[path]
    info = os.lstat(full_path)
    if mode == "120000":
        if not stat.S_ISLNK(info.st_mode):
            raise SystemExit(1)
        content = os.fsencode(os.readlink(full_path))
    else:
        if not stat.S_ISREG(info.st_mode):
            raise SystemExit(1)
        executable = bool(info.st_mode & stat.S_IXUSR)
        if executable != (mode == "100755"):
            raise SystemExit(1)
        with open(full_path, "rb") as stream:
            content = stream.read()
    header = b"blob " + str(len(content)).encode("ascii") + b"\x00"
    if hashlib.sha1(header + content).hexdigest() != expected_oid:
        raise SystemExit(1)
PY
}

verify_snapshot /tmp/workspace
cd /tmp/workspace
set +e
python3 - "$EVIDENCE_ARGV_B64" <<'PY'
import base64
import json
import os
import subprocess
import sys

argv = json.loads(base64.b64decode(sys.argv[1]).decode("utf-8"))
if not isinstance(argv, list) or not argv or not all(isinstance(value, str) and "\x00" not in value for value in argv):
    raise SystemExit("invalid evidence argv")
raise SystemExit(subprocess.run(argv, env=os.environ).returncode)
PY
subject_exit=$?
set -e
if verify_snapshot /tmp/workspace >/dev/null 2>&1; then
    subject_clean=true
else
    subject_clean=false
fi
printf '\n%s%s\n' "$EVIDENCE_STATUS_MARKER" "$subject_clean" >&2
exit "$subject_exit"
'@

    $dockerArguments = @(
        'run', '--pull', 'never', '--name', $containerName, '--network', 'none', '--read-only',
        '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges=true',
        '--memory', "$($MemoryMegabytes)m", '--memory-swap', "$($MemoryMegabytes)m",
        '--cpus', $CpuLimit.ToString([Globalization.CultureInfo]::InvariantCulture),
        '--pids-limit', $PidsLimit.ToString([Globalization.CultureInfo]::InvariantCulture),
        '--ulimit', 'nofile=1024:1024', '--user', '65534:65534', '--workdir', '/tmp',
        '--tmpfs', '/tmp:rw,exec,nosuid,nodev,size=512m',
        '--mount', "type=bind,source=$snapshotRoot,target=/evidence,readonly",
        '--entrypoint', '/usr/bin/env',
        $approvedContainerImage, '-i',
        'PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin',
        'LANG=C.UTF-8', 'LC_ALL=C.UTF-8', 'TZ=UTC', 'TMPDIR=/tmp',
        "EVIDENCE_REVISION=$resolvedRevision", "EVIDENCE_TREE=$resolvedTree",
        "EVIDENCE_SNAPSHOT_SHA256=$snapshotArchiveSha256", "EVIDENCE_MANIFEST_SHA256=$snapshotManifestSha256",
        "EVIDENCE_ARGV_B64=$argvBase64", "EVIDENCE_STATUS_MARKER=$statusMarker",
        '/bin/sh', '-c', $containerScript
    )

    $containerResult = Invoke-NativeCapture -FilePath $dockerExecutable -Arguments $dockerArguments -WallClockTimeoutSeconds $TimeoutSeconds -OutputLimitBytes $MaxOutputBytes
    $cleanup = Remove-EvidenceContainer -DockerPath $dockerExecutable -Name $containerName
    if (-not $cleanup.Succeeded) {
        & $writeFailure 'container-cleanup-failed' "Evidence container cleanup could not be proven: $($cleanup.Message)" $true $containerResult $true
    }
    if ($containerResult.TimedOut) {
        & $writeFailure 'wall-clock-timeout' "Evidence execution exceeded the $TimeoutSeconds second wall-clock limit." $true $containerResult $true
    }
    if ($containerResult.OutputLimitExceeded) {
        & $writeFailure 'output-limit-exceeded' "Evidence execution exceeded the $MaxOutputBytes byte combined output limit." $true $containerResult $true
    }
    if ($null -ne $containerResult.ReadFailure) {
        & $writeFailure 'output-capture-failed' "Evidence output capture failed: $($containerResult.ReadFailure)" $true $containerResult $true
    }

    $statusResult = Get-StatusMarkerResult -StderrBytes $containerResult.StderrBytes -Marker $statusMarker
    if (-not $statusResult.Success) {
        & $writeFailure 'container-bootstrap-failed' "Evidence container did not return its clean-status marker; refusing to accept incomplete evidence. $($containerResult.Stderr.Trim())" $true $containerResult $true
    }
    $subjectWorkspaceClean = $statusResult.SubjectClean
    $containerStderrBytes = [byte[]]$statusResult.CleanBytes
    $containerStderr = [Text.UTF8Encoding]::new($false).GetString($containerStderrBytes)

    $completedAt = [DateTimeOffset]::UtcNow
    $hostStatusAfter = Invoke-GitText @(
        '-C', $repoRoot, 'status', '--porcelain=v1', '--untracked-files=all', '--', '.',
        ':(exclude)docs/evidence/runs/**'
    )
    $stdoutBytes = [byte[]]$containerResult.StdoutBytes
    $stderrBytes = $containerStderrBytes
    $combinedBytes = New-Object byte[] ($stdoutBytes.Length + $stderrBytes.Length)
    if ($stdoutBytes.Length -gt 0) { [Array]::Copy($stdoutBytes, 0, $combinedBytes, 0, $stdoutBytes.Length) }
    if ($stderrBytes.Length -gt 0) { [Array]::Copy($stderrBytes, 0, $combinedBytes, $stdoutBytes.Length, $stderrBytes.Length) }
    $metadata = [ordered]@{
        schema = 'revision-evidence/v1'
        evidence_run_id = $runId
        evidence_revision = $resolvedRevision
        evidence_tree = $resolvedTree
        evidence_worktree = 'detached'
        argv = $argv
        exit_code = [int]$containerResult.ExitCode
        isolation = [ordered]@{
            engine = 'docker'
            docker_server_version = $dockerVersion.Stdout.Trim()
            image = $approvedContainerImage
            image_id = $imageId
            approved_image = $true
            entrypoint = '/usr/bin/env'
            network = 'none'
            repository_mount = 'read-only'
            host_repository_mounted = $false
            snapshot_source = 'git-archive-tar+manifest/v1'
            snapshot_mount = 'read-only'
            snapshot_archive_sha256 = $snapshotArchiveSha256
            snapshot_manifest_sha256 = $snapshotManifestSha256
            snapshot_manifest_entries = $snapshotManifestEntries
            snapshot_content_bytes = $snapshotContentBytes
            max_snapshot_bytes = $MaxSnapshotBytes
            root_filesystem = 'read-only'
            capabilities = 'none'
            no_new_privileges = $true
            user = '65534:65534'
            memory_megabytes = $MemoryMegabytes
            cpus = $CpuLimit
            pids_limit = $PidsLimit
            timeout_seconds = $TimeoutSeconds
            max_output_bytes = $MaxOutputBytes
            output_capture = 'streaming-bounded'
            sanitized_environment = $true
            environment_allowlist = @(
                'PATH', 'LANG', 'LC_ALL', 'TZ', 'TMPDIR', 'HOME', 'GOCACHE', 'GOPATH', 'GOMODCACHE',
                'GOENV', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'EVIDENCE_REVISION', 'EVIDENCE_TREE',
                'EVIDENCE_SNAPSHOT_SHA256', 'EVIDENCE_MANIFEST_SHA256', 'EVIDENCE_ARGV_B64', 'EVIDENCE_STATUS_MARKER'
            )
            preflight_passed = $true
            failure_kind = 'none'
            failure_message_sha256 = Get-Sha256Hex ([byte[]]@())
            workspace = '/tmp/workspace (tmpfs)'
            workspace_mount_options = 'rw,exec,nosuid,nodev,size=512m'
        }
        output = [ordered]@{
            stdout_sha256 = Get-Sha256Hex $stdoutBytes
            stderr_sha256 = Get-Sha256Hex $stderrBytes
            combined_sha256 = Get-Sha256Hex $combinedBytes
            stdout_bytes = $stdoutBytes.Length
            stderr_bytes = $stderrBytes.Length
            captured_bytes = [long]($stdoutBytes.Length + $stderrBytes.Length)
            observed_bytes_at_least = [long]$containerResult.ObservedBytesAtLeast
            truncated = $false
            capture_complete = $true
        }
        clean_status = [ordered]@{
            host_tracked_clean_before_run = $true
            host_tracked_clean_after_run = [string]::IsNullOrEmpty($hostStatusAfter)
            host_tracked_state_unchanged = [string]::IsNullOrEmpty($hostStatusAfter)
            subject_tracked_clean_before_run = $true
            subject_tracked_clean_after_run = $subjectWorkspaceClean
            subject_workspace_discarded = $true
        }
        tracked_worktree_clean_after_run = $subjectWorkspaceClean
        started_at = $startedAt.ToString('o')
        completed_at = $completedAt.ToString('o')
    }

    Write-EvidenceArtifact -Directory $artifactDirectory -Path $artifactPath -Metadata $metadata

    Write-Output '--- evidence command stdout ---'
    if ($containerResult.Stdout.Length -gt 0) {
        [Console]::Out.Write($containerResult.Stdout)
    }
    Write-Output '--- evidence command stderr ---'
    if ($containerStderr.Length -gt 0) {
        [Console]::Out.Write($containerStderr)
    }
    Write-Output '--- evidence metadata ---'
    Write-Output ($metadata | ConvertTo-Json -Compress -Depth 10)
    Write-Output "evidence_artifact: docs/evidence/runs/$runId/evidence.json"
    exit $containerResult.ExitCode
} catch {
    if (-not (Test-Path -LiteralPath $artifactDirectory)) {
        & $writeFailure 'runner-internal-error' "Evidence runner failed closed at line $($_.InvocationInfo.ScriptLineNumber): $($_.Exception.Message)" $preflightComplete $null (-not [string]::IsNullOrWhiteSpace($snapshotArchiveSha256))
    }
    throw
} finally {
    Remove-SnapshotDirectory $snapshotRoot
}
