param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$ExpectedHead,

    [Parameter(Mandatory = $true)]
    [string[]]$Paths,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$Message,

    [string]$RepositoryRoot,

    [ValidateRange(1, 300)]
    [int]$LockTimeoutSeconds = 60,

    [switch]$AllowDeletions
)

$ErrorActionPreference = 'Stop'

function Invoke-Git([string[]]$Arguments) {
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $gitExecutable -C $repoRoot @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($exitCode -ne 0) {
        throw "git $($Arguments -join ' ') failed: $($output -join [Environment]::NewLine)"
    }
    return $output
}

function Invoke-GitProbe([string[]]$Arguments) {
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $gitExecutable -C $repoRoot @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    return [pscustomobject]@{ ExitCode = $exitCode; Output = $output }
}

function Invoke-GitWithInput([string]$ArgumentText, [string]$InputText) {
    $startInfo = New-Object Diagnostics.ProcessStartInfo
    $startInfo.FileName = $gitExecutable
    $startInfo.Arguments = $ArgumentText
    $startInfo.WorkingDirectory = $repoRoot
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $startInfo
    try {
        [void]$process.Start()
        $inputBytes = (New-Object Text.UTF8Encoding($false)).GetBytes($InputText)
        $process.StandardInput.BaseStream.Write($inputBytes, 0, $inputBytes.Length)
        $process.StandardInput.BaseStream.Flush()
        $process.StandardInput.Close()
        $standardOutput = $process.StandardOutput.ReadToEnd()
        $standardError = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        if ($process.ExitCode -ne 0) {
            throw "git $ArgumentText failed: $standardError$standardOutput"
        }
        return $standardOutput
    } finally {
        $process.Dispose()
    }
}

function Get-SortedUniquePaths([object[]]$Values) {
    $pathSet = New-Object 'Collections.Generic.HashSet[string]' ([StringComparer]::Ordinal)
    foreach ($value in @($Values)) {
        if (-not [string]::IsNullOrWhiteSpace([string]$value)) {
            [void]$pathSet.Add(([string]$value).Replace('\', '/'))
        }
    }
    $result = [string[]]@($pathSet)
    [Array]::Sort($result, [StringComparer]::Ordinal)
    return $result
}

function Test-ExactPathSequence([string[]]$Left, [string[]]$Right) {
    if ($Left.Count -ne $Right.Count) { return $false }
    for ($index = 0; $index -lt $Left.Count; $index++) {
        if (-not [string]::Equals($Left[$index], $Right[$index], [StringComparison]::Ordinal)) { return $false }
    }
    return $true
}

function Get-NormalizedRepoPath([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or [IO.Path]::IsPathRooted($Path)) {
        throw "Governance transaction paths must be non-empty repo-relative paths: $Path"
    }
    $normalized = $Path.Replace('\', '/').Trim()
    while ($normalized.StartsWith('./')) { $normalized = $normalized.Substring(2) }
    if ($normalized -match '(^|/)\.\.?(?:/|$)' -or $normalized -match '(^|/)\.git(?:/|$)' -or $normalized.Contains(':')) {
        throw "Unsafe governance transaction path: $Path"
    }
    $resolved = [IO.Path]::GetFullPath((Join-Path $repoRoot $normalized))
    $rootPrefix = $repoRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
    if (-not $resolved.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Governance transaction path escapes the repository: $Path"
    }
    return $normalized
}

function Get-ChangedPaths() {
    $trackedResult = Invoke-GitProbe @('diff', '--name-only', 'HEAD', '--')
    if ($trackedResult.ExitCode -ne 0) { throw 'Unable to inspect tracked worktree changes' }
    $untrackedResult = Invoke-GitProbe @('ls-files', '--others', '--exclude-standard')
    if ($untrackedResult.ExitCode -ne 0) { throw 'Unable to inspect untracked worktree changes' }
    return @(Get-SortedUniquePaths @($trackedResult.Output + $untrackedResult.Output))
}

function Assert-TransactionPreconditions() {
    $currentHead = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim().ToLowerInvariant()
    if ($currentHead -ne $ExpectedHead.ToLowerInvariant()) {
        throw "Governance transaction HEAD CAS failed: expected=$ExpectedHead actual=$currentHead"
    }
    $indexResult = Invoke-GitProbe @('diff', '--cached', '--quiet', 'HEAD', '--')
    if ($indexResult.ExitCode -eq 1) {
        throw 'Governance transaction requires an empty Git index; do not pre-stage changes'
    }
    if ($indexResult.ExitCode -ne 0) { throw 'Unable to inspect the Git index' }
    $changedPaths = @(Get-ChangedPaths)
    if ($changedPaths.Count -eq 0) {
        throw 'Governance transaction has no changes to commit'
    }
    $unexpected = @($changedPaths | Where-Object { -not $normalizedPathSet.Contains($_) })
    if ($unexpected.Count -gt 0) {
        throw "Governance transaction refuses unrelated user or parallel changes: $($unexpected -join ', ')"
    }
    $changedPathSet = New-Object 'Collections.Generic.HashSet[string]' ([StringComparer]::Ordinal)
    foreach ($changedPath in $changedPaths) { [void]$changedPathSet.Add($changedPath) }
    $missing = @($normalizedPaths | Where-Object { -not $changedPathSet.Contains($_) })
    if ($missing.Count -gt 0) {
        throw "Governance transaction allowlist contains unchanged paths: $($missing -join ', ')"
    }
    foreach ($path in $normalizedPaths) {
        $fullPath = Join-Path $repoRoot $path
        if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
            if (-not $AllowDeletions) {
                throw "Governance transaction deletion requires -AllowDeletions: $path"
            }
            $trackedDeletionResult = Invoke-GitProbe @('cat-file', '-e', "$ExpectedHead`:$path")
            if ($trackedDeletionResult.ExitCode -ne 0) {
                throw "Governance transaction path is neither a file nor a tracked deletion: $path"
            }
        }
    }
    return $changedPaths
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}
$gitCommand = Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath($gitCommand.Source)
$repoPathPrefix = $repoRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
if ($gitExecutable.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-local Git binary: $gitExecutable"
}

$normalizedPaths = @(Get-SortedUniquePaths @($Paths | ForEach-Object { Get-NormalizedRepoPath $_ }))
$normalizedPathSet = New-Object 'Collections.Generic.HashSet[string]' ([StringComparer]::Ordinal)
foreach ($normalizedPath in $normalizedPaths) { [void]$normalizedPathSet.Add($normalizedPath) }
if ($normalizedPaths.Count -eq 0) {
    throw 'At least one exact governance transaction path is required'
}
if ($Message.Contains("`r") -or $Message.Contains("`n")) {
    throw 'Governance transaction commit messages must be a single line'
}

$commonDirectoryValue = (Invoke-Git @('rev-parse', '--git-common-dir') | Select-Object -First 1).Trim()
$commonDirectory = if ([IO.Path]::IsPathRooted($commonDirectoryValue)) {
    [IO.Path]::GetFullPath($commonDirectoryValue)
} else {
    [IO.Path]::GetFullPath((Join-Path $repoRoot $commonDirectoryValue))
}
$sha256 = [Security.Cryptography.SHA256]::Create()
try {
    $lockHashBytes = $sha256.ComputeHash([Text.Encoding]::UTF8.GetBytes($commonDirectory.ToLowerInvariant()))
} finally {
    $sha256.Dispose()
}
$lockHash = ([BitConverter]::ToString($lockHashBytes)).Replace('-', '').Substring(0, 16)
$mutex = New-Object Threading.Mutex($false, "allinme-core-api-governance-$lockHash")
$lockTaken = $false
$indexResetRequired = $false

try {
    $lockTaken = $mutex.WaitOne([TimeSpan]::FromSeconds($LockTimeoutSeconds))
    if (-not $lockTaken) { throw 'Timed out waiting for the repository governance transaction lock' }

    $changedPaths = @(Assert-TransactionPreconditions)
    $expectedBlobs = @{}
    foreach ($path in $normalizedPaths) {
        $fullPath = Join-Path $repoRoot $path
        if (Test-Path -LiteralPath $fullPath -PathType Leaf) {
            $expectedBlobs[$path] = (Invoke-Git @('hash-object', '--', $path) | Select-Object -First 1).Trim().ToLowerInvariant()
        } else {
            $expectedBlobs[$path] = '<deleted>'
        }
    }
    Invoke-Git @('read-tree', $ExpectedHead) | Out-Null
    $indexResetRequired = $true
    Invoke-Git (@('add', '-A', '--') + $normalizedPaths) | Out-Null

    $stagedPaths = @(Get-SortedUniquePaths @(Invoke-Git @('diff', '--cached', '--name-only', $ExpectedHead, '--') | ForEach-Object { $_.Trim() }))
    if (-not (Test-ExactPathSequence $stagedPaths $changedPaths)) {
        throw "Exact staging verification failed: expected=$($changedPaths -join ',') staged=$($stagedPaths -join ',')"
    }
    foreach ($path in $normalizedPaths) {
        $stagedBlobResult = Invoke-GitProbe @('rev-parse', '--verify', ":$path")
        $stagedBlob = if ($stagedBlobResult.ExitCode -eq 0 -and $stagedBlobResult.Output.Count -gt 0) {
            @($stagedBlobResult.Output | Select-Object -First 1)[0].Trim().ToLowerInvariant()
        } else { '<deleted>' }
        if ($stagedBlob -cne $expectedBlobs[$path]) {
            throw "Governance transaction path changed during staging: $path"
        }
    }

    $tree = (Invoke-Git @('write-tree') | Select-Object -First 1).Trim()
    $newRevision = (Invoke-GitWithInput "commit-tree $tree -p $ExpectedHead" "$Message`n").Trim()
    if ($newRevision -notmatch '^[0-9a-f]{40}$') { throw "git commit-tree returned an invalid revision: $newRevision" }

    $committedPaths = @(Get-SortedUniquePaths @(Invoke-Git @('diff-tree', '--no-commit-id', '--name-only', '-r', $newRevision) | ForEach-Object { $_.Trim() }))
    if (-not (Test-ExactPathSequence $committedPaths $changedPaths)) {
        throw "Governance commit path verification failed: expected=$($changedPaths -join ',') committed=$($committedPaths -join ',')"
    }
    $unstagedPaths = @(Get-SortedUniquePaths @(Invoke-Git @('diff', '--name-only', '--') | ForEach-Object { $_.Trim() }))
    $untrackedPaths = @(Get-SortedUniquePaths @(Invoke-Git @('ls-files', '--others', '--exclude-standard') | ForEach-Object { $_.Trim() }))
    if ($unstagedPaths.Count -gt 0 -or $untrackedPaths.Count -gt 0) {
        throw "Governance transaction worktree changed before ref CAS: $(@($unstagedPaths + $untrackedPaths) -join ', ')"
    }

    $headRefResult = Invoke-GitProbe @('symbolic-ref', '-q', 'HEAD')
    $headRef = @($headRefResult.Output | Select-Object -First 1)
    if ($headRef.Count -eq 0 -or [string]::IsNullOrWhiteSpace($headRef[0])) { $headRef = @('HEAD') }
    $headRef = $headRef[0].Trim()
    $governanceRef = 'refs/allinme/governance-head'
    $existingGovernanceResult = Invoke-GitProbe @('rev-parse', '--verify', $governanceRef)
    $existingGovernanceRevision = @($existingGovernanceResult.Output | Select-Object -First 1)
    if ($existingGovernanceResult.ExitCode -eq 0 -and $existingGovernanceRevision.Count -gt 0) {
        $existingGovernanceRevision = $existingGovernanceRevision[0].Trim()
        $governanceAncestorResult = Invoke-GitProbe @('merge-base', '--is-ancestor', $existingGovernanceRevision, $ExpectedHead)
        if ($governanceAncestorResult.ExitCode -ne 0) {
            throw "Governance transaction would fork the shared governance chain: shared=$existingGovernanceRevision expected=$ExpectedHead"
        }
    } else {
        $existingGovernanceRevision = $null
    }
    $refCommands = New-Object Collections.Generic.List[string]
    $refCommands.Add('start')
    $refCommands.Add("update $headRef $newRevision $ExpectedHead")
    if ($null -eq $existingGovernanceRevision) {
        $refCommands.Add("create $governanceRef $newRevision")
    } else {
        $refCommands.Add("update $governanceRef $newRevision $existingGovernanceRevision")
    }
    $refCommands.Add('prepare')
    $refCommands.Add('commit')
    try {
        Invoke-GitWithInput 'update-ref --stdin -m "governance transaction"' (($refCommands -join "`n") + "`n") | Out-Null
    } catch {
        throw "Governance transaction ref CAS failed: $($_.Exception.Message)"
    }
    $indexResetRequired = $false

    [ordered]@{
        governance_revision = $newRevision
        parent_revision = $ExpectedHead.ToLowerInvariant()
        paths = $committedPaths
        worktree = $repoRoot
        git_common_directory = $commonDirectory
        governance_ref = $governanceRef
    } | ConvertTo-Json -Compress -Depth 4
} finally {
    if ($indexResetRequired) {
        & $gitExecutable -C $repoRoot read-tree $ExpectedHead 2>$null
    }
    if ($lockTaken) { $mutex.ReleaseMutex() }
    $mutex.Dispose()
}
