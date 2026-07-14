param(
    [Parameter(Mandatory = $true)]
    [string]$Revision,

    [Parameter(Mandatory = $true)]
    [string]$Command,

    [string[]]$CommandArgs = @()
)

$ErrorActionPreference = 'Stop'

if ($Command.Count -eq 0) {
    throw 'A command and its arguments must follow -Command.'
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$revisionToken = ($Revision -replace '^git:', '').Split(';')[0].Trim()
$resolvedRevision = (& git -C $repoRoot rev-parse --verify "$revisionToken^{commit}" 2>&1 | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $resolvedRevision -notmatch '^[0-9a-f]{40}$') {
    throw "Revision does not resolve to a full commit: $Revision"
}

$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$worktreePath = Join-Path $tempRoot ("allinme-audit-evidence-{0}" -f ([guid]::NewGuid().ToString('N')))
$worktreePath = [IO.Path]::GetFullPath($worktreePath)
if (-not $worktreePath.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to create an evidence worktree outside the temp directory: $worktreePath"
}

$commandName = $Command
$commandArgs = @($CommandArgs)
$exitCode = 1
$commandOutput = @()

try {
    & git -C $repoRoot worktree add --detach --quiet $worktreePath $resolvedRevision
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to create detached evidence worktree for $resolvedRevision"
    }

    $actualRevision = (& git -C $worktreePath rev-parse HEAD 2>&1 | Out-String).Trim()
    $actualTree = (& git -C $worktreePath rev-parse 'HEAD^{tree}' 2>&1 | Out-String).Trim()
    if ($actualRevision -ne $resolvedRevision -or $actualTree -notmatch '^[0-9a-f]{40}$') {
        throw "Detached evidence worktree does not match requested revision: expected=$resolvedRevision actual=$actualRevision"
    }

    Push-Location $worktreePath
    try {
        $previousErrorPreference = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $commandOutput = @(& $commandName @commandArgs 2>&1)
        $exitCode = if ($null -ne $LASTEXITCODE) { [int]$LASTEXITCODE } else { 0 }
        $ErrorActionPreference = $previousErrorPreference
    } finally {
        $ErrorActionPreference = 'Stop'
        Pop-Location
    }

    $trackedChanges = @(& git -C $worktreePath status --short --untracked-files=no 2>&1)
    $metadata = [ordered]@{
        evidence_revision = $actualRevision
        evidence_tree = $actualTree
        evidence_worktree = 'detached'
        command = @($commandName) + $commandArgs
        exit_code = $exitCode
        tracked_worktree_clean_after_run = ($trackedChanges.Count -eq 0)
    }

    Write-Output '--- evidence command output ---'
    $commandOutput | ForEach-Object { Write-Output $_ }
    Write-Output '--- evidence metadata ---'
    Write-Output ($metadata | ConvertTo-Json -Compress -Depth 4)
} finally {
    if (Test-Path -LiteralPath $worktreePath) {
        & git -C $repoRoot worktree remove --force $worktreePath 2>$null
    }
    & git -C $repoRoot worktree prune 2>$null
}

exit $exitCode
