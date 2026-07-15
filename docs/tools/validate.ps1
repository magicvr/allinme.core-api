param(
    [string]$DocsRoot,
    [string]$HistoryBase
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$gitCommand = @(Get-Command git -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
if ($null -eq $gitCommand -or [string]::IsNullOrWhiteSpace([string]$gitCommand.Source)) {
    throw 'Git is required to validate documentation.'
}
$gitExecutable = [IO.Path]::GetFullPath([string]$gitCommand.Source)
$repoRootPrefix = $repoRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if ([string]::Equals($gitExecutable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
    $gitExecutable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-controlled Git binary: $gitExecutable"
}
if ([string]::IsNullOrWhiteSpace($DocsRoot)) {
    $docsRoot = Join-Path $repoRoot 'docs'
} else {
    $docsRoot = (Resolve-Path $DocsRoot).Path
}
$canonicalDocsRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'docs')).TrimEnd('\', '/')
$useGitLedgerChecks = [string]::Equals(
    [IO.Path]::GetFullPath($docsRoot).TrimEnd('\', '/'),
    $canonicalDocsRoot,
    [StringComparison]::OrdinalIgnoreCase
)
$approvedEvidenceContainerImage = 'docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a'
$frontmatterExceptions = @(
    'CHANGELOG.md',
    'README.md',
    'audits/README.md',
    'audits/templates/audit-record.md',
    'audits/templates/follow-up-audit-record.md',
    'audits/templates/implementation-acceptance-audit-record.md',
    'audits/templates/implementation-audit-record.md',
    'audits/templates/plan-acceptance-audit-record.md',
    'audits/templates/plan-audit-record.md',
    'decisions/README.md',
    'evidence/README.md',
    'evidence/runtime-attestations/README.md',
    'implementations/README.md',
    'implementations/templates/implementation-record.md',
    'plans/README.md',
    'plans/archived/README.md',
    'plans/templates/checklist.md',
    'plans/templates/plan.md',
    'remediations/README.md',
    'remediations/templates/remediation-record.md',
    'scenarios/README.md',
    'tools/README.md'
)

function Get-RepoRelativePath([string]$Path) {
    return $Path.Substring($repoRoot.Length + 1).Replace('\', '/')
}

function Get-DocsRelativePath([string]$Path) {
    return $Path.Substring($docsRoot.Length + 1).Replace('\', '/')
}

function Get-FrontmatterValue([string]$Frontmatter, [string]$Field) {
    $match = [regex]::Match($Frontmatter, "(?m)^${Field}:\s*(?<value>.+?)\s*$")
    if (-not $match.Success) {
        return $null
    }
    return $match.Groups['value'].Value
}

function Get-IndexLines([string]$Content, [string]$Target) {
    $pattern = '(?m)^.*\]\(' + [regex]::Escape($Target) + '\).*$'
    return [regex]::Matches($Content, $pattern)
}

function Get-ListValues([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -eq 'none') {
        return @()
    }
    return @($Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function ConvertFrom-StrictEvidenceArgvJson([string]$Value) {
    $invalid = [pscustomobject]@{ IsValid = $false; Items = @() }
    if ([string]::IsNullOrWhiteSpace($Value)) { return $invalid }
    try {
        $parsed = ConvertFrom-Json -InputObject $Value -ErrorAction Stop
    } catch {
        return $invalid
    }
    if ($parsed -isnot [System.Array]) { return $invalid }
    $items = @($parsed)
    if ($items.Count -eq 0 -or
        @($items | Where-Object {
            $_ -isnot [string] -or [string]::IsNullOrWhiteSpace([string]$_) -or ([string]$_).IndexOf([char]0) -ge 0
        }).Count -gt 0) {
        return $invalid
    }
    return [pscustomobject]@{ IsValid = $true; Items = $items }
}

function Test-ExactStringArrays([object[]]$Left, [object[]]$Right) {
    $leftItems = @($Left)
    $rightItems = @($Right)
    if ($leftItems.Count -ne $rightItems.Count) { return $false }
    for ($index = 0; $index -lt $leftItems.Count; $index++) {
        if ([string]$leftItems[$index] -cne [string]$rightItems[$index]) { return $false }
    }
    return $true
}

function Test-MeaningfulEvidenceArgv([object[]]$Argv) {
    if ($null -eq $Argv -or $Argv.Count -eq 0) { return $false }
    $command = [IO.Path]::GetFileName(([string]$Argv[0]).Trim()).ToLowerInvariant()
    if ($command -notin @('go', 'go.exe')) {
        return $false
    }
    if ($Argv.Count -lt 2) {
        return $false
    }
    $subcommand = ([string]$Argv[1]).Trim().ToLowerInvariant()
    if ($subcommand -notin @('test', 'vet', 'build')) { return $false }
    return $true
}

function Get-AuditNumber([string]$AuditId) {
    if ($AuditId -notmatch '^AUD-(?<number>\d{4})$') {
        return -1
    }
    return [int]$Matches['number']
}

function Get-ImplementationNumber([string]$ImplementationId) {
    if ($ImplementationId -notmatch '^IMP-(?<number>\d{4})$') {
        return -1
    }
    return [int]$Matches['number']
}

function Get-RemediationNumber([string]$RemediationId) {
    if ($RemediationId -notmatch '^REM-(?<number>\d{4})$') {
        return -1
    }
    return [int]$Matches['number']
}

function Get-GitRevision([string]$Value) {
    if ($Value -match '^git:(?<sha>[0-9a-fA-F]{40})(?:;\s*worktree:clean)?$') {
        return $Matches['sha'].ToLowerInvariant()
    }
    return $null
}

function ConvertTo-DateTimeOffsetOrNull([string]$Value) {
    $parsed = [DateTimeOffset]::MinValue
    if (-not [string]::IsNullOrWhiteSpace($Value) -and [DateTimeOffset]::TryParse($Value, [ref]$parsed)) {
        return $parsed
    }
    return $null
}

function Test-DisplayRecordAvailableBefore([object]$Earlier, [object]$Later) {
    if ($null -eq $Earlier -or $null -eq $Later) { return $false }
    $earlierCompleted = ConvertTo-DateTimeOffsetOrNull $Earlier.CompletedAt
    $laterStarted = ConvertTo-DateTimeOffsetOrNull $Later.StartedAt
    return $null -ne $earlierCompleted -and $null -ne $laterStarted -and $earlierCompleted -le $laterStarted
}

function Test-Rfc3339Timestamp([string]$Value) {
    if ($Value -notmatch '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$') { return $false }
    return $null -ne (ConvertTo-DateTimeOffsetOrNull $Value)
}

function Test-GitCommitExists([string]$Revision) {
    if ([string]::IsNullOrWhiteSpace($Revision)) {
        return $false
    }
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & $gitExecutable -C $repoRoot cat-file -e "$Revision`^{commit}" 2>$null
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return $exitCode -eq 0
}

function Test-GitAncestor([string]$Ancestor, [string]$Descendant) {
    if ([string]::IsNullOrWhiteSpace($Ancestor) -or [string]::IsNullOrWhiteSpace($Descendant)) {
        return $false
    }
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & $gitExecutable -C $repoRoot merge-base --is-ancestor $Ancestor $Descendant 2>$null
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return $exitCode -eq 0
}

function Test-GitPathExistsAtRevision([string]$Revision, [string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Revision) -or [string]::IsNullOrWhiteSpace($Path)) {
        return $false
    }
    $normalizedPath = $Path.Replace('\', '/')
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & $gitExecutable -C $repoRoot cat-file -e "$Revision`:$normalizedPath" 2>$null
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return $exitCode -eq 0
}

function Get-GitBlobIdAtRevision([string]$Revision, [string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Revision) -or [string]::IsNullOrWhiteSpace($Path)) { return $null }
    $normalizedPath = $Path.Replace('\', '/')
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $gitExecutable -C $repoRoot rev-parse "$Revision`:$normalizedPath" 2>$null)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    $value = $output | Select-Object -First 1
    if ($exitCode -ne 0 -or $null -eq $value -or $value -notmatch '^[0-9a-fA-F]{40}$') { return $null }
    return ([string]$value).ToLowerInvariant()
}

function Get-WorkingTreeBlobId([string]$Path) {
    $fullPath = Join-Path $repoRoot $Path
    if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) { return $null }
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $gitExecutable -C $repoRoot hash-object -- $fullPath 2>$null)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    $value = $output | Select-Object -First 1
    if ($exitCode -ne 0 -or $null -eq $value -or $value -notmatch '^[0-9a-fA-F]{40}$') { return $null }
    return ([string]$value).ToLowerInvariant()
}

function Test-GitPathMatchesWorkingTree([string]$Revision, [string]$Path) {
    $revisionBlob = Get-GitBlobIdAtRevision $Revision $Path
    $workingTreeBlob = Get-WorkingTreeBlobId $Path
    return $null -ne $revisionBlob -and $revisionBlob -eq $workingTreeBlob
}

$script:gitPlanSnapshotCache = @{}

function Get-GitFileContentAtRevision([string]$Revision, [string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Revision) -or [string]::IsNullOrWhiteSpace($Path)) { return $null }
    $normalizedPath = $Path.Replace('\', '/')
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $gitExecutable -C $repoRoot show "$Revision`:$normalizedPath" 2>$null)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    if ($exitCode -ne 0) { return $null }
    return ($output | Out-String)
}

function Get-GitPlanSnapshotAtRevision([string]$Revision) {
    if ([string]::IsNullOrWhiteSpace($Revision) -or -not (Test-GitCommitExists $Revision)) { return @{} }
    $cacheKey = $Revision.ToLowerInvariant()
    if ($script:gitPlanSnapshotCache.ContainsKey($cacheKey)) {
        return $script:gitPlanSnapshotCache[$cacheKey]
    }

    $snapshot = @{}
    $paths = @(& $gitExecutable -C $repoRoot ls-tree -r --name-only $Revision -- docs/plans 2>$null)
    if ($LASTEXITCODE -ne 0) { return @{} }
    foreach ($path in $paths) {
        $normalizedPath = $path.Replace('\', '/')
        $pathMatch = [regex]::Match($normalizedPath, '^docs/plans/(?:(?:archived)/)?(?<stem>PLN-(?<number>\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*)\.md$')
        if (-not $pathMatch.Success -or $pathMatch.Groups['stem'].Value.EndsWith('-checklist')) {
            continue
        }
        $content = Get-GitFileContentAtRevision $Revision $normalizedPath
        if ($null -eq $content -or $content -notmatch '(?s)\A(?:\uFEFF)?---\r?\n(?<frontmatter>.*?)\r?\n---\r?\n') { continue }
        $frontmatter = $Matches['frontmatter']
        $planId = "PLN-$($pathMatch.Groups['number'].Value)"
        $checklistPath = $normalizedPath.Substring(0, $normalizedPath.Length - 3) + '-checklist.md'
        $snapshot[$planId] = @{
            Status = Get-FrontmatterValue $frontmatter 'status'
            PlanPath = $normalizedPath
            ChecklistPath = if (Test-GitPathExistsAtRevision $Revision $checklistPath) { $checklistPath } else { $null }
        }
    }
    $script:gitPlanSnapshotCache[$cacheKey] = $snapshot
    return $snapshot
}

function Get-ActivePlanIdsAtRevision([string]$Revision) {
    $snapshot = Get-GitPlanSnapshotAtRevision $Revision
    return @($snapshot.Keys | Where-Object { $snapshot[$_].Status -eq 'active' } | Sort-Object)
}

function Test-GitRecordMatchesSnapshot([string]$Revision, [hashtable]$Info) {
    if ($null -eq $Info -or [string]::IsNullOrWhiteSpace($Info.Path)) { return $false }
    if (-not $useGitLedgerChecks) { return $true }
    $snapshotBlob = Get-GitBlobIdAtRevision $Revision $Info.Path
    $terminalBlob = Get-WorkingTreeBlobId $Info.Path
    return $null -ne $snapshotBlob -and $snapshotBlob -eq $terminalBlob
}

function Test-GitEntryMatchesSnapshot([string]$Revision, [object]$Entry) {
    return $null -ne $Entry -and (Test-GitRecordMatchesSnapshot $Revision $Entry.Value)
}

function Test-GitMetadataEntryMatchesSnapshot([string]$Revision, [object]$Entry) {
    if ($null -eq $Entry) { return $false }
    if (-not $useGitLedgerChecks) { return $true }
    $snapshotBlob = Get-GitBlobIdAtRevision $Revision $Entry.Value.Path
    $workingTreeBlob = Get-WorkingTreeBlobId $Entry.Value.Path
    return $null -ne $snapshotBlob -and $snapshotBlob -eq $workingTreeBlob
}

function Test-GitRecordDescendsFrom([hashtable]$LaterInfo, [hashtable]$EarlierInfo) {
    if ($null -eq $LaterInfo -or $null -eq $EarlierInfo) { return $false }
    if (-not $useGitLedgerChecks) { return $true }
    $laterBaseline = Get-GitRevision $LaterInfo.Baseline
    if ($null -eq $laterBaseline -or -not (Test-GitCommitExists $laterBaseline)) { return $false }
    return Test-GitRecordMatchesSnapshot $laterBaseline $EarlierInfo
}

function Get-GitPathLastChangeCommitAtRevision([string]$Revision, [string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Revision) -or [string]::IsNullOrWhiteSpace($Path)) { return $null }
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $gitExecutable -C $repoRoot rev-list -1 $Revision -- $Path 2>$null)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    $commit = $output | Select-Object -First 1
    if ($exitCode -ne 0 -or $null -eq $commit -or $commit -notmatch '^[0-9a-fA-F]{40}$') { return $null }
    return ([string]$commit).ToLowerInvariant()
}

function Select-LatestGitRecord([object[]]$Entries, [string]$Revision, [System.Collections.Generic.List[string]]$Failures, [string]$Label) {
    if ($Entries.Count -eq 0) { return $null }
    if (-not $useGitLedgerChecks) {
        return $Entries | Sort-Object {
            if ($_.Key -match '^[A-Z]+-(?<number>\d{4})$') { [int]$Matches['number'] } else { -1 }
        } | Select-Object -Last 1
    }
    $candidates = New-Object System.Collections.Generic.List[object]
    foreach ($entry in $Entries) {
        $changeCommit = Get-GitPathLastChangeCommitAtRevision $Revision $entry.Value.Path
        if ($null -eq $changeCommit) {
            $Failures.Add("$Label cannot derive a Git ordering commit: $($entry.Value.Path)")
            continue
        }
        $candidates.Add([pscustomobject]@{ Entry = $entry; Commit = $changeCommit })
    }
    $maximal = New-Object System.Collections.Generic.List[object]
    foreach ($candidate in $candidates) {
        $shadowed = $false
        foreach ($other in $candidates) {
            if ($candidate.Entry.Key -eq $other.Entry.Key -or $candidate.Commit -eq $other.Commit) { continue }
            if (Test-GitAncestor $candidate.Commit $other.Commit) {
                $shadowed = $true
                break
            }
        }
        if (-not $shadowed) { $maximal.Add($candidate) }
    }
    if ($maximal.Count -ne 1) {
        $Failures.Add("$Label must have one ancestry-maximal record at $Revision; found $($maximal.Count)")
        return $null
    }
    return $maximal[0].Entry
}

function Test-RevisionEvidenceArtifact(
    [string]$AuditId,
    [hashtable]$AuditInfo,
    [bool]$RequireSuccess,
    [System.Collections.Generic.List[string]]$Failures
) {
    $runId = $AuditInfo.EvidenceRunId
    if (-not (Test-UuidV4 $runId)) {
        $Failures.Add("Terminal audit-runtime/v1 record must bind a UUIDv4 evidence artifact: $($AuditInfo.Path)")
        return
    }
    $artifactRelativePath = "docs/evidence/runs/$($runId.ToLowerInvariant())/evidence.json"
    if ($AuditInfo.EvidenceArtifact -ne $artifactRelativePath) {
        $Failures.Add("Audit evidence_artifact must exactly match its evidence_run_id: $($AuditInfo.Path) (expected=$artifactRelativePath; actual=$($AuditInfo.EvidenceArtifact))")
        return
    }
    $artifactPath = Join-Path $repoRoot $artifactRelativePath
    if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) {
        $Failures.Add("Evidence artifact is missing: $($AuditInfo.Path) ($artifactRelativePath)")
        return
    }
    $artifactItem = Get-Item -LiteralPath $artifactPath -Force
    if (($artifactItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or
        -not [string]::IsNullOrWhiteSpace($artifactItem.LinkType)) {
        $Failures.Add("Evidence artifact must be a regular non-reparse file: $($AuditInfo.Path) ($artifactRelativePath)")
        return
    }

    try {
        $artifact = Get-Content -LiteralPath $artifactPath -Raw -Encoding UTF8 | ConvertFrom-Json
    } catch {
        $Failures.Add("Evidence artifact must be valid JSON: $($AuditInfo.Path) ($artifactRelativePath)")
        return
    }
    if ($artifact -isnot [pscustomobject]) {
        $Failures.Add("Evidence artifact must be a JSON object: $($AuditInfo.Path) ($artifactRelativePath)")
        return
    }
    $declaredArgvJson = [string]$AuditInfo.EvidenceArgvJson
    $declaredArgvResult = ConvertFrom-StrictEvidenceArgvJson $declaredArgvJson
    $declarationRequired = [bool]$AuditInfo.IsNew -or -not [string]::IsNullOrWhiteSpace($declaredArgvJson)
    if ($declarationRequired -and -not $declaredArgvResult.IsValid) {
        $Failures.Add("Audit evidence_argv_json must be strict JSON containing a non-empty array of non-empty strings: $($AuditInfo.Path)")
    }
    $requiredTopLevel = @('schema', 'evidence_run_id', 'evidence_revision', 'evidence_tree', 'evidence_worktree', 'argv', 'exit_code', 'isolation', 'output', 'clean_status', 'tracked_worktree_clean_after_run', 'started_at', 'completed_at')
    foreach ($field in $requiredTopLevel) {
        if ($artifact.PSObject.Properties.Name -notcontains $field) {
            $Failures.Add("Evidence artifact is missing '$field': $($AuditInfo.Path) ($artifactRelativePath)")
        }
    }
    if ($artifact.schema -isnot [string] -or $artifact.schema -cne 'revision-evidence/v1' -or
        $artifact.evidence_run_id -isnot [string] -or $artifact.evidence_run_id -cne $runId.ToLowerInvariant()) {
        $Failures.Add("Evidence artifact schema/run ID does not match the audit: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if ($artifact.evidence_worktree -isnot [string] -or $artifact.evidence_worktree -cne 'detached') {
        $Failures.Add("Evidence artifact must prove a detached evidence worktree: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    $evidenceSha = Get-GitRevision $AuditInfo.EvidenceRevision
    if ($null -eq $evidenceSha -or $artifact.evidence_revision -isnot [string] -or $artifact.evidence_revision -cne $evidenceSha) {
        $Failures.Add("Evidence artifact revision does not match evidence_revision: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif (Test-GitCommitExists $evidenceSha) {
        $previousPreference = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $treeOutput = @(& $gitExecutable -C $repoRoot rev-parse "$evidenceSha`^{tree}" 2>$null)
        $treeExitCode = $LASTEXITCODE
        $ErrorActionPreference = $previousPreference
        $expectedTree = $treeOutput | Select-Object -First 1
        if ($treeExitCode -ne 0 -or $artifact.evidence_tree -isnot [string] -or $artifact.evidence_tree -cne $expectedTree) {
            $Failures.Add("Evidence artifact tree does not match evidence_revision: $($AuditInfo.Path) ($artifactRelativePath)")
        }
    }
    $argv = @($artifact.argv)
    if ($artifact.argv -isnot [System.Array] -or $argv.Count -eq 0 -or @($argv | Where-Object { $_ -isnot [string] -or [string]::IsNullOrWhiteSpace($_) }).Count -gt 0) {
        $Failures.Add("Evidence artifact argv must be a non-empty string array: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif ($declaredArgvResult.IsValid -and -not (Test-ExactStringArrays $declaredArgvResult.Items $argv)) {
        $Failures.Add("Audit evidence_argv_json must exactly match the ordered evidence artifact argv: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif (-not (Test-MeaningfulEvidenceArgv $argv)) {
        $Failures.Add("Evidence artifact must contain a subject-specific command (meaningful argv required): $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif (($argv -join ' ') -match '(?i)(?:^|[\\/])validate(?:-[a-z0-9-]+)?(?:\.tests)?\.ps1(?:\s|$)') {
        $Failures.Add("Evidence artifact must contain a subject-specific command: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    $parsedExitCode = [long]0
    $exitCodeIsInteger = $artifact.exit_code -is [int] -or $artifact.exit_code -is [long]
    if (-not $exitCodeIsInteger -or [long]$artifact.exit_code -lt [int]::MinValue -or [long]$artifact.exit_code -gt [int]::MaxValue) {
        $Failures.Add("Evidence artifact exit_code must be a signed 32-bit integer: $($AuditInfo.Path) ($artifactRelativePath)")
    } else {
        $parsedExitCode = [long]$artifact.exit_code
    }
    if ($RequireSuccess -and $exitCodeIsInteger -and $parsedExitCode -ne 0) {
        $Failures.Add("Successful acceptance requires evidence exit_code=0: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if (-not $RequireSuccess -and $exitCodeIsInteger -and $parsedExitCode -ne 0 -and
        $AuditInfo.Content -notmatch '(?m)^-\s*Disposition:\s*(?:open|partially-resolved)\s*$') {
        $Failures.Add("Nonzero primary evidence must be reflected by an actionable audit finding: $($AuditInfo.Path) ($artifactRelativePath)")
    }

    $isolation = $artifact.isolation
    $memoryIsInteger = $isolation.memory_megabytes -is [int] -or $isolation.memory_megabytes -is [long]
    $memoryMegabytes = if ($memoryIsInteger) { [long]$isolation.memory_megabytes } else { [long]0 }
    $cpuIsNumber = $isolation.cpus -is [int] -or $isolation.cpus -is [long] -or
        $isolation.cpus -is [decimal] -or $isolation.cpus -is [double]
    $cpuLimit = if ($cpuIsNumber) { [double]$isolation.cpus } else { 0.0 }
    $pidsIsInteger = $isolation.pids_limit -is [int] -or $isolation.pids_limit -is [long]
    $pidsLimit = if ($pidsIsInteger) { [long]$isolation.pids_limit } else { [long]0 }
    $snapshotEntriesIsInteger = $isolation.snapshot_manifest_entries -is [int] -or $isolation.snapshot_manifest_entries -is [long]
    $timeoutIsInteger = $isolation.timeout_seconds -is [int] -or $isolation.timeout_seconds -is [long]
    $outputLimitIsInteger = $isolation.max_output_bytes -is [int] -or $isolation.max_output_bytes -is [long]
    $failureKind = [string]$isolation.failure_kind
    $allowedFailureKinds = @('none', 'wall-clock-timeout', 'output-limit-exceeded')
    if ($isolation -isnot [pscustomobject] -or
        $isolation.engine -isnot [string] -or $isolation.engine -cne 'docker' -or
        $isolation.image -isnot [string] -or $isolation.image -cne $approvedEvidenceContainerImage -or
        $isolation.approved_image -isnot [bool] -or -not $isolation.approved_image -or
        $isolation.image_id -isnot [string] -or $isolation.image_id -notmatch '^sha256:[0-9a-f]{64}$' -or
        $isolation.entrypoint -isnot [string] -or $isolation.entrypoint -cne '/usr/bin/env' -or
        $isolation.network -isnot [string] -or $isolation.network -cne 'none' -or
        $isolation.repository_mount -isnot [string] -or $isolation.repository_mount -cne 'read-only' -or
        $isolation.host_repository_mounted -isnot [bool] -or $isolation.host_repository_mounted -or
        $isolation.snapshot_source -isnot [string] -or $isolation.snapshot_source -cne 'git-archive-tar+manifest/v1' -or
        $isolation.snapshot_mount -isnot [string] -or $isolation.snapshot_mount -cne 'read-only' -or
        $isolation.snapshot_archive_sha256 -isnot [string] -or $isolation.snapshot_archive_sha256 -cnotmatch '^[0-9a-f]{64}$' -or
        $isolation.snapshot_manifest_sha256 -isnot [string] -or $isolation.snapshot_manifest_sha256 -cnotmatch '^[0-9a-f]{64}$' -or
        -not $snapshotEntriesIsInteger -or [long]$isolation.snapshot_manifest_entries -lt 1 -or
        $isolation.root_filesystem -isnot [string] -or $isolation.root_filesystem -cne 'read-only' -or
        $isolation.capabilities -isnot [string] -or $isolation.capabilities -cne 'none' -or
        $isolation.no_new_privileges -isnot [bool] -or -not $isolation.no_new_privileges -or
        $isolation.user -isnot [string] -or $isolation.user -cne '65534:65534' -or
        -not $memoryIsInteger -or $memoryMegabytes -lt 64 -or $memoryMegabytes -gt 4096 -or
        -not $cpuIsNumber -or $cpuLimit -lt 0.1 -or $cpuLimit -gt 4.0 -or
        -not $pidsIsInteger -or $pidsLimit -lt 16 -or $pidsLimit -gt 1024 -or
        -not $timeoutIsInteger -or [long]$isolation.timeout_seconds -lt 1 -or [long]$isolation.timeout_seconds -gt 3600 -or
        -not $outputLimitIsInteger -or [long]$isolation.max_output_bytes -lt 1024 -or [long]$isolation.max_output_bytes -gt 16777216 -or
        $isolation.output_capture -isnot [string] -or $isolation.output_capture -cne 'streaming-bounded' -or
        $isolation.sanitized_environment -isnot [bool] -or -not $isolation.sanitized_environment -or
        $isolation.preflight_passed -isnot [bool] -or -not $isolation.preflight_passed -or
        $failureKind -notin $allowedFailureKinds -or
        $isolation.failure_message_sha256 -isnot [string] -or $isolation.failure_message_sha256 -cnotmatch '^[0-9a-f]{64}$' -or
        $isolation.workspace -isnot [string] -or $isolation.workspace -cne '/tmp/workspace (tmpfs)' -or
        $isolation.workspace_mount_options -isnot [string] -or $isolation.workspace_mount_options -cne 'rw,exec,nosuid,nodev,size=512m') {
        $Failures.Add("Evidence artifact does not prove the required isolated execution: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if ($RequireSuccess -and $failureKind -ne 'none') {
        $Failures.Add("Successful evidence cannot use a runner failure_kind other than none: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if ($failureKind -eq 'none' -and $isolation.failure_message_sha256 -cne 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855') {
        $Failures.Add("Successful runner metadata must bind an empty failure message: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if (-not $RequireSuccess -and $failureKind -notin @('none', 'wall-clock-timeout', 'output-limit-exceeded')) {
        $Failures.Add("Evidence runner failure_kind is not admissible as primary audit evidence: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    $artifactOutput = $artifact.output
    foreach ($digestField in @('stdout_sha256', 'stderr_sha256', 'combined_sha256')) {
        if ($artifactOutput -isnot [pscustomobject] -or $artifactOutput.$digestField -isnot [string] -or
            $artifactOutput.$digestField -cnotmatch '^[0-9a-f]{64}$') {
            $Failures.Add("Evidence artifact output.$digestField must be SHA-256: $($AuditInfo.Path) ($artifactRelativePath)")
        }
    }
    foreach ($byteField in @('stdout_bytes', 'stderr_bytes')) {
        if ($artifactOutput -isnot [pscustomobject] -or
            ($artifactOutput.$byteField -isnot [int] -and $artifactOutput.$byteField -isnot [long]) -or
            [long]$artifactOutput.$byteField -lt 0) {
            $Failures.Add("Evidence artifact output.$byteField must be non-negative: $($AuditInfo.Path) ($artifactRelativePath)")
        }
    }
    $capturedBytesIsInteger = $artifactOutput -is [pscustomobject] -and ($artifactOutput.captured_bytes -is [int] -or $artifactOutput.captured_bytes -is [long])
    $observedBytesIsInteger = $artifactOutput -is [pscustomobject] -and ($artifactOutput.observed_bytes_at_least -is [int] -or $artifactOutput.observed_bytes_at_least -is [long])
    if (-not $capturedBytesIsInteger -or -not $observedBytesIsInteger -or
        [long]$artifactOutput.captured_bytes -ne ([long]$artifactOutput.stdout_bytes + [long]$artifactOutput.stderr_bytes) -or
        [long]$artifactOutput.captured_bytes -gt [long]$isolation.max_output_bytes -or
        [long]$artifactOutput.observed_bytes_at_least -lt [long]$artifactOutput.captured_bytes -or
        $artifactOutput.truncated -isnot [bool] -or $artifactOutput.capture_complete -isnot [bool]) {
        $Failures.Add("Evidence artifact output capture bounds are invalid: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif ($failureKind -eq 'none' -and
        ($artifactOutput.truncated -or -not $artifactOutput.capture_complete -or
         [long]$artifactOutput.observed_bytes_at_least -ne [long]$artifactOutput.captured_bytes)) {
        $Failures.Add("Successful evidence must prove complete, non-truncated bounded output: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif ($failureKind -eq 'output-limit-exceeded' -and -not $artifactOutput.truncated) {
        $Failures.Add("Output-limit evidence must record truncation: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    $cleanStatus = $artifact.clean_status
    if ($cleanStatus -isnot [pscustomobject] -or
        $cleanStatus.host_tracked_clean_before_run -isnot [bool] -or
        $cleanStatus.host_tracked_clean_after_run -isnot [bool] -or
        $cleanStatus.host_tracked_state_unchanged -isnot [bool] -or -not $cleanStatus.host_tracked_state_unchanged -or
        $cleanStatus.subject_tracked_clean_before_run -isnot [bool] -or
        $cleanStatus.subject_tracked_clean_after_run -isnot [bool] -or
        $cleanStatus.subject_workspace_discarded -isnot [bool] -or -not $cleanStatus.subject_workspace_discarded -or
        $artifact.tracked_worktree_clean_after_run -isnot [bool]) {
        $Failures.Add("Evidence artifact must prove typed clean status, unchanged host state, and discarded subject workspace: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if ($RequireSuccess -and
        ($cleanStatus -isnot [pscustomobject] -or
         $cleanStatus.subject_tracked_clean_before_run -isnot [bool] -or -not $cleanStatus.subject_tracked_clean_before_run -or
         $cleanStatus.subject_tracked_clean_after_run -isnot [bool] -or -not $cleanStatus.subject_tracked_clean_after_run -or
         $artifact.tracked_worktree_clean_after_run -isnot [bool] -or -not $artifact.tracked_worktree_clean_after_run)) {
        $Failures.Add("Successful evidence must start and finish with the detached subject tracked-clean: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if ($artifact.started_at -isnot [string] -or $artifact.completed_at -isnot [string] -or
        -not (Test-Rfc3339Timestamp $artifact.started_at) -or -not (Test-Rfc3339Timestamp $artifact.completed_at)) {
        $Failures.Add("Evidence artifact timestamps must be RFC3339: $($AuditInfo.Path) ($artifactRelativePath)")
    } elseif ((ConvertTo-DateTimeOffsetOrNull $artifact.completed_at) -lt (ConvertTo-DateTimeOffsetOrNull $artifact.started_at)) {
        $Failures.Add("Evidence artifact completed_at cannot precede started_at: $($AuditInfo.Path) ($artifactRelativePath)")
    }
    if (Test-GitPathMatchesWorkingTree 'HEAD' $AuditInfo.Path) {
        $recordCommit = Get-GitPathLastChangeCommitAtRevision 'HEAD' $AuditInfo.Path
        $artifactBlobAtRecordCommit = Get-GitBlobIdAtRevision $recordCommit $artifactRelativePath
        $artifactBlob = Get-WorkingTreeBlobId $artifactRelativePath
        if ($null -eq $artifactBlobAtRecordCommit -or $artifactBlobAtRecordCommit -ne $artifactBlob) {
            $Failures.Add("Committed terminal audit must bind the immutable evidence artifact blob: $($AuditInfo.Path) ($artifactRelativePath)")
        }
    }
}

function Test-UuidV4([string]$Value) {
    return $Value -match '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$'
}

function Test-EvidencePlaceholder([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value)) { return $true }
    return $Value -match '^(?:\.\.\.|TODO|TBD|具体证据|具体文件、frontmatter、链接和索引证据|相关计划 AUD/REM/follow-up 链和验收基线证据|计划/实施 AUD、REM、follow-up 链和验收基线证据|<required:.*>)$'
}

function Test-SubjectSpecificValidation([object]$AuditInfo, [string]$AuditId, [System.Collections.Generic.List[string]]$Failures, [string]$Label) {
    $validationSection = [regex]::Match($AuditInfo.Content, '(?s)##\s+验证结果\s*(?<body>.*?)(?=\r?\n##\s+|\z)')
    if (-not $validationSection.Success) {
        $Failures.Add("Successful $Label must record subject-specific validation: $AuditId")
        return
    }
    $commands = @([regex]::Matches($validationSection.Groups['body'].Value, '`(?<command>[^`\r\n]+)`') | ForEach-Object { $_.Groups['command'].Value.Trim() })
    $subjectCommands = @($commands | Where-Object {
        $_ -notmatch '(?i)(?:^|[\\/])validate(?:\.tests)?\.ps1(?:\s|$)' -and
        $_ -notmatch '(?i)^git\s+diff(?:\s+HEAD)?\s+--check$'
    })
    if ($subjectCommands.Count -eq 0) {
        $Failures.Add("Successful $Label must include a non-governance subject-specific command: $AuditId")
    }
    $declaredArgv = ConvertFrom-StrictEvidenceArgvJson ([string]$AuditInfo.EvidenceArgvJson)
    if (-not $declaredArgv.IsValid) { return }
    $declaredCommand = ($declaredArgv.Items -join ' ')
    if ($commands -cnotcontains $declaredCommand) {
        $Failures.Add("Successful $Label validation results must quote the exact evidence_argv_json command: $AuditId ($declaredCommand)")
    }
}

function Test-FindingDetails([string]$Content, [string]$AuditId, [System.Collections.Generic.List[string]]$Failures, [string]$Label) {
    $findingMatches = [regex]::Matches($Content, "(?m)^###\s+(?<id>$([regex]::Escape($AuditId))-F\d{3})\s+-.*$")
    foreach ($findingMatch in $findingMatches) {
        $start = $findingMatch.Index
        $next = [regex]::Match($Content.Substring($start + $findingMatch.Length), '(?m)^###\s+')
        $end = if ($next.Success) { $start + $findingMatch.Length + $next.Index } else { $Content.Length }
        $section = $Content.Substring($start, $end - $start)
        foreach ($field in @('Severity', 'Evidence', 'Impact', 'Recommendation', 'Owner', 'Disposition')) {
            if ($section -notmatch "(?m)^-\s*$field`:\s*\S.+$") {
                $Failures.Add("$Label finding must record ${field}: $($findingMatch.Groups['id'].Value)")
            }
        }
        $dispositionMatch = [regex]::Match($section, '(?m)^-\s*Disposition:\s*(?<value>\S+)\s*$')
        if ($dispositionMatch.Success -and $dispositionMatch.Groups['value'].Value -notin @('open', 'partially-resolved', 'resolved', 'accepted-risk', 'not-reproduced', 'superseded')) {
            $Failures.Add("$Label finding has an invalid Disposition: $($findingMatch.Groups['id'].Value)")
        }
    }
}

function Test-PlanReadinessChainAudit([string]$AuditId, [System.Collections.Generic.HashSet[string]]$Visited) {
    if (-not $auditMetadata.ContainsKey($AuditId) -or $Visited.Contains($AuditId)) { return $false }
    [void]$Visited.Add($AuditId)
    $info = $auditMetadata[$AuditId]
    if ($info.Schema -in @('plan-audit/v2', 'plan-acceptance/v2')) { return $true }
    if ($info.AuditType -ne 'follow-up') { return $false }
    foreach ($sourceAuditId in @($info.RelatedAudits)) {
        if (Test-PlanReadinessChainAudit $sourceAuditId $Visited) { return $true }
    }
    return $false
}

function Test-PhaseFiveEdge([hashtable]$Dag, [string]$Prerequisite, [string]$Dependent) {
    return $Dag.ContainsKey($Dependent) -and @($Dag[$Dependent]) -contains $Prerequisite
}

function Get-PhaseFiveStatementClauses([string]$Text) {
    $separator = '(?<=[.;\u3002\uFF1B!?\uFF01\uFF1F])\s*|[,\uFF0C]?\s*(?=(?i:(?:but|however|yet)\b|\u4F46\u662F|\u7136\u800C|\u4E0D\u8FC7|\u4F46))'
    return @([regex]::Split($Text, $separator) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Get-PhaseFiveDependencyClauses([string]$Text) {
    $clauses = New-Object System.Collections.Generic.List[string]
    foreach ($sentence in [regex]::Split($Text, '(?<=[.;\u3002\uFF1B!?\uFF01\uFF1F])\s*')) {
        if ([string]::IsNullOrWhiteSpace($sentence)) {
            continue
        }
        if ($sentence -match '(?i)(reject|must\s+reject|\u62D2\u7EDD)') {
            foreach ($clause in Get-PhaseFiveStatementClauses $sentence) {
                $clauses.Add($clause)
            }
        } else {
            $clauses.Add($sentence)
        }
    }
    return @($clauses)
}

function Get-PhaseFiveDag([string]$Content, [System.Collections.Generic.List[string]]$Failures) {
    $dag = @{}
    $rows = [regex]::Matches($Content, '(?m)^\|\s*(?<package>WP-[A-Za-z0-9-]+)\s*\|(?<items>[^|]*)\|(?<owner>[^|]*)\|(?<inputs>[^|]*)\|(?<timebox>[^|]*)\|(?<evidence>[^|]*)\|\s*$')
    foreach ($row in $rows) {
        $package = $row.Groups['package'].Value
        if ($dag.ContainsKey($package)) {
            $Failures.Add("PLN-0005 dependency DAG contains duplicate work package: $package")
            continue
        }
        $dependencies = @([regex]::Matches($row.Groups['inputs'].Value, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value } | Select-Object -Unique)
        $dag[$package] = $dependencies
    }

    $expectedPackages = @('WP-Facts', 'WP-Schema-Recovery', 'WP-HTTP-Order', 'WP-Lock', 'WP-Baseline-Evidence', 'WP-Files', 'WP-Runtime', 'WP-Release')
    foreach ($package in $expectedPackages) {
        if (-not $dag.ContainsKey($package)) {
            $Failures.Add("PLN-0005 dependency DAG is missing work package: $package")
        }
    }
    foreach ($package in @($dag.Keys)) {
        if ($package -notin $expectedPackages) {
            $Failures.Add("PLN-0005 dependency DAG contains unknown work package: $package")
        }
        foreach ($dependency in @($dag[$package])) {
            if ($dependency -eq $package) {
                $Failures.Add("PLN-0005 dependency DAG contains a self dependency: $package")
            } elseif (-not $dag.ContainsKey($dependency)) {
                $Failures.Add("PLN-0005 dependency DAG references unknown dependency: $package -> $dependency")
            }
        }
    }

    $requiredDependencies = @{
        'WP-Facts' = @()
        'WP-Schema-Recovery' = @('WP-Facts')
        'WP-HTTP-Order' = @('WP-Facts')
        'WP-Lock' = @('WP-Facts')
        'WP-Baseline-Evidence' = @('WP-Facts')
        'WP-Files' = @('WP-Lock')
        'WP-Runtime' = @('WP-Lock')
        'WP-Release' = @('WP-Facts', 'WP-Schema-Recovery', 'WP-HTTP-Order', 'WP-Lock', 'WP-Baseline-Evidence', 'WP-Files', 'WP-Runtime')
    }
    foreach ($package in $requiredDependencies.Keys) {
        if (-not $dag.ContainsKey($package)) {
            continue
        }
        $actual = @($dag[$package] | Sort-Object)
        $expected = @($requiredDependencies[$package] | Sort-Object)
        if (($actual -join ',') -ne ($expected -join ',')) {
            $Failures.Add("PLN-0005 dependency DAG inputs do not match the tracked contract for ${package}: expected [$($expected -join ', ')], found [$($actual -join ', ')]")
        }
    }

    $remaining = @{}
    foreach ($package in $dag.Keys) {
        $remaining[$package] = @($dag[$package]).Count
    }
    $ready = New-Object System.Collections.Generic.Queue[string]
    foreach ($package in $remaining.Keys) {
        if ($remaining[$package] -eq 0) {
            $ready.Enqueue($package)
        }
    }
    $processed = 0
    while ($ready.Count -gt 0) {
        $completed = $ready.Dequeue()
        $processed++
        foreach ($dependent in @($dag.Keys)) {
            if (@($dag[$dependent]) -contains $completed) {
                $remaining[$dependent]--
                if ($remaining[$dependent] -eq 0) {
                    $ready.Enqueue($dependent)
                }
            }
        }
    }
    if ($dag.Count -gt 0 -and $processed -ne $dag.Count) {
        $Failures.Add('PLN-0005 dependency DAG contains a cycle')
    }
    return $dag
}

function Test-PhaseFiveFactsOutput([string]$Content, [System.Collections.Generic.List[string]]$Failures) {
    $row = [regex]::Match($Content, '(?m)^\|\s*WP-Facts\s*\|(?<items>[^|]*)\|(?<owner>[^|]*)\|(?<inputs>[^|]*)\|(?<timebox>[^|]*)\|(?<evidence>[^|]*)\|\s*$')
    if (-not $row.Success) {
        $Failures.Add('PLN-0005 tracked work-package table is missing the WP-Facts output contract')
        return
    }

    $requiredOutputs = @(
        'docs/01-architecture.md',
        'docs/05-domain-model.md',
        'docs/03-http-api-target.md',
        'docs/06-implementation-roadmap.md',
        'docs/04-validation.md',
        'plan/checklist'
    )
    $evidence = $row.Groups['evidence'].Value
    $outputTokens = @(
        [regex]::Split($evidence, '[,\uFF0C\u3001;\uFF1B\s]+') |
            ForEach-Object { $_.Trim('`', '.', ':', ',', ';') } |
            Where-Object { $_ -match '^(?:docs|plan)[/\\]' }
    )
    foreach ($outputToken in $outputTokens) {
        if ($outputToken -notin $requiredOutputs) {
            $Failures.Add("PLN-0005 WP-Facts output contains unknown or non-canonical token: $outputToken")
        }
    }
    foreach ($duplicate in @($outputTokens | Group-Object | Where-Object { $_.Count -gt 1 })) {
        $Failures.Add("PLN-0005 WP-Facts output contains duplicate token: $($duplicate.Name)")
    }
    foreach ($requiredOutput in $requiredOutputs) {
        if ($outputTokens -notcontains $requiredOutput) {
            $Failures.Add("PLN-0005 WP-Facts output is missing required fact source: $requiredOutput")
        }
    }
}

function Test-PhaseFiveDependencyStatements([string]$Content, [hashtable]$Dag, [System.Collections.Generic.List[string]]$Failures) {
    $contentWithoutDagRows = [regex]::Replace($Content, '(?m)^\|\s*WP-[A-Za-z0-9-]+\s*\|.*$', '')
    foreach ($line in [regex]::Split($contentWithoutDagRows, '\r?\n')) {
        foreach ($clause in Get-PhaseFiveDependencyClauses $line) {
            if ($clause -notmatch 'WP-[A-Za-z0-9-]+') {
                continue
            }
            $isRejectionClause = $clause -match '(?i)(reject|must\s+reject|\u62D2\u7EDD)'
            if ($isRejectionClause) {
                continue
            }

            $consumedPackages = New-Object System.Collections.Generic.HashSet[string]
            $relationshipFound = $false
            foreach ($match in [regex]::Matches($clause, '(?<prerequisite>WP-[A-Za-z0-9-]+)\s*(?:->|\u2192)\s*(?<dependent>WP-[A-Za-z0-9-]+)')) {
                $relationshipFound = $true
                [void]$consumedPackages.Add($match.Groups['prerequisite'].Value)
                [void]$consumedPackages.Add($match.Groups['dependent'].Value)
                if (-not (Test-PhaseFiveEdge $Dag $match.Groups['prerequisite'].Value $match.Groups['dependent'].Value)) {
                    $Failures.Add("PLN-0005 dependency statement is not present in the tracked DAG: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)(?<middle>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?)(?<prerequisite>WP-[A-Za-z0-9-]+)(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?)(?:does\s+not|doesn''t|need\s+not)\s+depend\s+(?:on|upon)\s+it\b')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                $prerequisite = $match.Groups['prerequisite'].Value
                [void]$consumedPackages.Add($dependent)
                [void]$consumedPackages.Add($prerequisite)
                if (Test-PhaseFiveEdge $Dag $prerequisite $dependent) {
                    $Failures.Add("PLN-0005 dependency statement denies a tracked edge: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?<negation>does\s+not|doesn''t|need\s+not|without|\u4E0D\u4F9D\u8D56|\u65E0\u9700\u7B49\u5F85)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:depend(?:ing)?\s+(?:on|upon)\s+)?(?<prerequisite>WP-[A-Za-z0-9-]+)')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                $prerequisite = $match.Groups['prerequisite'].Value
                [void]$consumedPackages.Add($dependent)
                [void]$consumedPackages.Add($prerequisite)
                if (Test-PhaseFiveEdge $Dag $prerequisite $dependent) {
                    $Failures.Add("PLN-0005 dependency statement denies a tracked edge: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:depends?\s+(?:on|upon)|(?<!\u5148\u4E8E)\u4F9D\u8D56)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                if ($match.Value -match '(?i)(?:does\s+not|doesn''t|need\s+not)\s+depend|\u4E0D\u4F9D\u8D56|\u65E0\u9700\u7B49\u5F85') {
                    continue
                }
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                [void]$consumedPackages.Add($dependent)
                $prerequisites = @([regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value })
                if ($prerequisites.Count -eq 0) {
                    $Failures.Add("PLN-0005 dependency statement could not be fully parsed: $($match.Value)")
                }
                foreach ($prerequisite in $prerequisites) {
                    [void]$consumedPackages.Add($prerequisite)
                    if (-not (Test-PhaseFiveEdge $Dag $prerequisite $dependent)) {
                        $Failures.Add("PLN-0005 dependency statement contradicts the tracked DAG: $dependent depends on $prerequisite")
                    }
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<first>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:before|precedes?|\u5148\u4E8E|\u65E9\u4E8E)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                $relationshipFound = $true
                $first = $match.Groups['first'].Value
                [void]$consumedPackages.Add($first)
                foreach ($secondMatch in [regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+')) {
                    $second = $secondMatch.Value
                    [void]$consumedPackages.Add($second)
                    if (-not (Test-PhaseFiveEdge $Dag $first $second)) {
                        $Failures.Add("PLN-0005 dependency ordering contradicts or is absent from the tracked DAG: $first before $second")
                    }
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:runs?\s+after|after|follows?|\u665A\u4E8E|\u4E4B\u540E)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                [void]$consumedPackages.Add($dependent)
                foreach ($prerequisiteMatch in [regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+')) {
                    $prerequisite = $prerequisiteMatch.Value
                    [void]$consumedPackages.Add($prerequisite)
                    if (-not (Test-PhaseFiveEdge $Dag $prerequisite $dependent)) {
                        $Failures.Add("PLN-0005 dependency ordering contradicts or is absent from the tracked DAG: $dependent after $prerequisite")
                    }
                }
            }

            $allPackages = @([regex]::Matches($clause, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value } | Select-Object -Unique)
            if ($allPackages.Count -gt 1) {
                if (-not $relationshipFound) {
                    $Failures.Add("PLN-0005 dependency statement could not be fully parsed: $($clause.Trim())")
                } else {
                    foreach ($package in $allPackages) {
                        if (-not $consumedPackages.Contains($package)) {
                            $Failures.Add("PLN-0005 dependency statement contains an unconsumed work package: $package in $($clause.Trim())")
                        }
                    }
                }
            }
        }
    }
}

function Get-PhaseFiveLiveEvidencePattern([string[]]$Categories, [System.Collections.Generic.List[string]]$Failures) {
    $patterns = @{
        'release-binary' = '(?:release[-\s]+binary|live\s+binary|real\s+binary|\u9636\u6BB5\u4E94\s*binary|\u771F\u5B9E(?:\u9636\u6BB5\u4E94\s*)?binary)'
        'supervisor-run' = '(?:live\s+supervisor|real\s+supervisor|supervisor.{0,24}(?:run|evidence)|\u771F\u5B9E(?:\u9636\u6BB5\u4E94\s*)?\u76D1\u7763\u5668|\u76D1\u7763\u5668.{0,20}(?:run|Evidence|\u8BC1\u636E))'
        'cleanup-schedule-run' = '(?:cleanup\s+schedule|cleanup\s*\u8C03\u5EA6|\u6E05\u7406\u8C03\u5EA6)'
        'watchdog-recovery-run' = '(?:watchdog(?:/|\s+and\s+|\s*)recovery|watchdog.{0,24}(?:run|evidence))'
        'enospc-run' = '(?:ENOSPC.{0,40}(?:run|evidence|\u8BC1\u636E|\u6F14\u7EC3))'
        'live-profile-run' = '(?:live\s+(?:deployment\s+)?profile|real\s+(?:deployment\s+)?profile|\u90E8\u7F72\s*profile\s*run|\u5B9E\u6D4B.{0,20}(?:\u90E8\u7F72|profile))'
    }
    $selected = New-Object System.Collections.Generic.List[string]
    foreach ($category in $Categories) {
        if (-not $patterns.ContainsKey($category)) {
            $Failures.Add("PLN-0005 deployment evidence contract contains an unknown forbidden category: $category")
            continue
        }
        $selected.Add($patterns[$category])
    }
    if ($selected.Count -eq 0) {
        return $null
    }
    return '(?i)(' + ($selected -join '|') + ')'
}

function Test-PhaseFiveDeploymentClauses([string]$Content, [string[]]$ForbiddenCategories, [bool]$Checklist, [System.Collections.Generic.List[string]]$Failures) {
    $liveEvidence = Get-PhaseFiveLiveEvidencePattern $ForbiddenCategories $Failures
    if ([string]::IsNullOrWhiteSpace($liveEvidence)) {
        return
    }
    $obligation = '(?i)(must|required?|shall|before\s+P0|P0.{0,20}(?:complete|gate)|\u5FC5\u987B|\u8981\u6C42|\u5B8C\u6210\u524D|\u963B\u585E|\u4E0D\u5F97\u8FDB\u5165)'
    $deferral = '(?i)(not\s+require|does\s+not\s+require|must\s+not\s+require|not.{0,30}P0.{0,20}(?:prerequisite|gate)|defer|belongs?\s+to|only.{0,40}(?:5A-D|5B)|\u4E0D\u5F97\u8981\u6C42|\u4E0D\u8981\u6C42|\u7559\u7ED9|\u5C5E\u4E8E|\u53EA\u80FD\u7531|\u540E\u79FB|\u5E76\u975E|\u4E0D\u662F.{0,30}P0.{0,20}(?:\u524D\u7F6E|\u95E8\u7981)|\u4E0D\u5F97\u628A|\u7531.{0,40}(?:M1A|5A-D|5B).{0,20}(?:\u63D0\u4F9B|\u4EA7\u751F|\u5B8C\u6210))'
    $scopes = if ($Checklist) {
        @([regex]::Matches($Content, '(?ms)^- \[[ xX]\] P0-(?<number>\d+)\.(?<body>.*?)(?=^- \[[ xX]\] (?:P0-\d+|[A-Za-z0-9]+-\d+)\.|^## |\z)') | ForEach-Object {
            @{ Label = "P0-$($_.Groups['number'].Value)"; Text = $_.Groups['body'].Value }
        })
    } else {
        @(@{ Label = 'plan'; Text = $Content })
    }
    foreach ($scope in $scopes) {
        $clauses = Get-PhaseFiveStatementClauses $scope.Text
        foreach ($clause in $clauses) {
            if (-not $Checklist -and $clause -notmatch '(?i)\bP0(?:-\d+)?\b') {
                continue
            }
            $liveMatches = @([regex]::Matches($clause, $liveEvidence))
            if ($liveMatches.Count -eq 0) {
                continue
            }
            $deferralMatches = @([regex]::Matches($clause, $deferral))
            $obligationMatches = New-Object System.Collections.Generic.List[object]
            foreach ($obligationMatch in [regex]::Matches($clause, $obligation)) {
                $overlapsDeferral = $false
                foreach ($deferralMatch in $deferralMatches) {
                    if ($obligationMatch.Index -lt ($deferralMatch.Index + $deferralMatch.Length) -and
                        $deferralMatch.Index -lt ($obligationMatch.Index + $obligationMatch.Length)) {
                        $overlapsDeferral = $true
                        break
                    }
                }
                if (-not $overlapsDeferral) {
                    $obligationMatches.Add($obligationMatch)
                }
            }

            $violation = $false
            foreach ($liveMatch in $liveMatches) {
                $nearestType = $null
                $nearestDistance = [int]::MaxValue
                foreach ($signal in @($deferralMatches | ForEach-Object { @{ Type = 'deferral'; Match = $_ } }) + @($obligationMatches | ForEach-Object { @{ Type = 'obligation'; Match = $_ } })) {
                    $signalMatch = $signal.Match
                    $liveEnd = $liveMatch.Index + $liveMatch.Length
                    $signalEnd = $signalMatch.Index + $signalMatch.Length
                    $distance = if ($signalEnd -le $liveMatch.Index) {
                        $liveMatch.Index - $signalEnd
                    } elseif ($liveEnd -le $signalMatch.Index) {
                        $signalMatch.Index - $liveEnd
                    } else {
                        0
                    }
                    if ($distance -lt $nearestDistance -or
                        ($distance -eq $nearestDistance -and $signal.Type -eq 'obligation')) {
                        $nearestDistance = $distance
                        $nearestType = $signal.Type
                    }
                }
                if ($nearestType -eq 'obligation') {
                    $violation = $true
                    break
                }
            }
            if ($violation) {
                $Failures.Add("PLN-0005 P0 deployment evidence clause must stop at contract fixtures and defer live runs ($($scope.Label)): $($clause.Trim())")
            }
        }
    }
}

function Test-AuditMatrix([string]$Content, [string]$Marker, [string[]]$Controls, [string]$AuditId, [System.Collections.Generic.List[string]]$Failures, [string]$Label) {
    if ($Content -match '(?m)^status:\s*superseded\s*$') { return }
    $markers = [regex]::Matches($Content, [regex]::Escape($Marker))
    if ($markers.Count -ne 1) {
        $Failures.Add("$Label must contain exactly one matrix marker: $Marker")
        return
    }
    $separatorIndex = $Marker.LastIndexOf(':')
    $markerPrefix = if ($separatorIndex -ge 0) { $Marker.Substring(0, $separatorIndex + 1) } else { $Marker }
    $matrixContent = Get-MatrixContent $Content $Marker $markerPrefix
    foreach ($control in $Controls) {
        $rows = [regex]::Matches($matrixContent, "(?m)^\|\s*$([regex]::Escape($control))\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail)\s*\|(?<finding>[^|]*)\|\s*$")
        if ($rows.Count -ne 1) {
            $Failures.Add("$Label must contain one valid $control row: $Marker")
            continue
        }
        $evidence = $rows[0].Groups['evidence'].Value.Trim()
        $verdict = $rows[0].Groups['verdict'].Value
        $finding = $rows[0].Groups['finding'].Value.Trim()
        if (Test-EvidencePlaceholder $evidence) {
            $Failures.Add("$Label evidence is empty for ${control}: $Marker")
        }
        if ($verdict -eq 'fail') {
            if ($finding -notmatch "^$([regex]::Escape($AuditId))-F\d{3}$" -or
                $Content -notmatch "(?m)^###\s+$([regex]::Escape($finding))\s+-") {
                $Failures.Add("Failed $Label control must reference an existing finding: $control in $Marker")
            }
        } elseif ($finding -ne 'none') {
            $Failures.Add("Passing $Label control must use finding=none: $control in $Marker")
        }
    }
}

function Get-MatrixContent([string]$Content, [string]$Marker, [string]$MarkerPrefix) {
    $markerIndex = $Content.IndexOf($Marker, [StringComparison]::Ordinal)
    if ($markerIndex -lt 0) {
        return $null
    }
    $nextMarker = $Content.IndexOf($MarkerPrefix, $markerIndex + $Marker.Length, [StringComparison]::Ordinal)
    $matrixEnd = if ($nextMarker -ge 0) { $nextMarker } else { $Content.Length }
    return $Content.Substring($markerIndex, $matrixEnd - $markerIndex)
}

function Test-AcceptanceVerdict([string]$Content, [string]$Marker, [string]$MarkerPrefix, [string[]]$Controls, [string]$Verdict, [string]$Status, [string]$AuditId, [string]$Label, [System.Collections.Generic.List[string]]$Failures) {
    if ($Status -eq 'open' -and $Verdict -ne 'pending') {
        $Failures.Add("Closed acceptance result must use status=closed: $AuditId")
    }
    if ($Status -eq 'closed' -and $Verdict -eq 'pending') {
        $Failures.Add("Closed $Label cannot keep acceptance_verdict=pending: $AuditId")
    }
    if ($Status -eq 'superseded' -and $Verdict -ne 'superseded') {
        $Failures.Add("Superseded $Label must use acceptance_verdict=superseded: $AuditId")
    }
    if ($Verdict -eq 'superseded') {
        if ($Status -ne 'superseded') {
            $Failures.Add("acceptance_verdict=superseded requires status=superseded: $AuditId")
        }
        return
    }
    if ($Verdict -eq 'pending') {
        return
    }
    $matrixContent = if ([string]::IsNullOrWhiteSpace($Marker)) { $Content } else { Get-MatrixContent $Content $Marker $MarkerPrefix }
    if ($null -eq $matrixContent) { return }
    $failedCount = 0
    foreach ($control in $Controls) {
        $rows = [regex]::Matches($matrixContent, "(?m)^\|\s*$([regex]::Escape($control))\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail)\s*\|(?<finding>[^|]*)\|\s*$")
        foreach ($row in $rows) {
            if ($row.Groups['verdict'].Value -eq 'fail') {
                $failedCount++
            }
        }
    }
    if ($Verdict -in @('ready', 'complete') -and $failedCount -gt 0) {
        $Failures.Add("Acceptance verdict $Verdict requires every Control to pass: $AuditId")
    }
    if ($Verdict -in @('not-ready', 'blocked', 'incomplete') -and $failedCount -eq 0) {
        $Failures.Add("Acceptance verdict $Verdict requires at least one failed Control with a finding: $AuditId")
    }
}

$failures = New-Object System.Collections.Generic.List[string]
$effectiveHistoryBase = $HistoryBase
if ([string]::IsNullOrWhiteSpace($effectiveHistoryBase)) {
    $effectiveHistoryBase = $env:AUDIT_HISTORY_BASE
}
if ($useGitLedgerChecks -and [string]::IsNullOrWhiteSpace($effectiveHistoryBase)) {
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $upstreamRevision = (& $gitExecutable -C $repoRoot rev-parse --verify '@{upstream}' 2>$null | Select-Object -First 1)
    $upstreamExitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    if ($upstreamExitCode -eq 0 -and $upstreamRevision -match '^[0-9a-fA-F]{40}$') {
        $effectiveHistoryBase = (& $gitExecutable -C $repoRoot merge-base HEAD $upstreamRevision 2>$null | Select-Object -First 1)
    }
    if ([string]::IsNullOrWhiteSpace($effectiveHistoryBase)) {
        $previousErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $effectiveHistoryBase = (& $gitExecutable -C $repoRoot rev-parse --verify 'HEAD^' 2>$null | Select-Object -First 1)
        $ErrorActionPreference = $previousErrorAction
    }
    if ([string]::IsNullOrWhiteSpace($effectiveHistoryBase)) {
        $failures.Add('Documentation validation requires an explicit or resolvable HistoryBase.')
    }
}
$validationShell = @(Get-Command pwsh -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
if ($null -eq $validationShell) {
    $validationShell = @(Get-Command powershell.exe -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
}
if ($null -eq $validationShell) {
    $validationShell = @(Get-Command powershell -CommandType Application -ErrorAction Stop) | Select-Object -First 1
}
$validationShellExecutable = [IO.Path]::GetFullPath([string]$validationShell.Source)
if ([string]::Equals($validationShellExecutable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
    $validationShellExecutable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-controlled validation shell: $validationShellExecutable"
}
$auditWorkflowValidator = Join-Path $PSScriptRoot 'validate-audit-workflows.ps1'
if (-not (Test-Path -LiteralPath $auditWorkflowValidator)) {
    $failures.Add('Audit workflow contract validator is missing: docs/tools/validate-audit-workflows.ps1')
} else {
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $workflowOutput = & $validationShellExecutable -NoProfile -ExecutionPolicy Bypass -File $auditWorkflowValidator -RepositoryRoot $repoRoot 2>&1
    $workflowExitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    if ($workflowExitCode -ne 0) {
        $failures.Add("audit workflow contracts validator exited with code $workflowExitCode.")
        foreach ($line in $workflowOutput) {
            $failures.Add("audit workflow contracts: $line")
        }
    }
}
if ($useGitLedgerChecks) {
    $runtimeAttestationValidator = Join-Path $PSScriptRoot 'validate-runtime-attestations.ps1'
    if (-not (Test-Path -LiteralPath $runtimeAttestationValidator)) {
        $failures.Add('Runtime attestation validator is missing: docs/tools/validate-runtime-attestations.ps1')
    } else {
        $attestationArguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $runtimeAttestationValidator, '-RepositoryRoot', $repoRoot)
        if (-not [string]::IsNullOrWhiteSpace($effectiveHistoryBase)) { $attestationArguments += @('-HistoryBase', $effectiveHistoryBase) }
        $previousErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $attestationOutput = & $validationShellExecutable @attestationArguments 2>&1
        $attestationExitCode = $LASTEXITCODE
        $ErrorActionPreference = $previousErrorAction
        if ($attestationExitCode -ne 0) {
            $failures.Add("runtime attestation validator exited with code $attestationExitCode.")
            foreach ($line in $attestationOutput) {
                $failures.Add("runtime attestations: $line")
            }
        }
    }

    $evidenceAttestationValidator = Join-Path $PSScriptRoot 'validate-evidence-attestations.ps1'
    if (-not (Test-Path -LiteralPath $evidenceAttestationValidator)) {
        $failures.Add('Evidence attestation validator is missing: docs/tools/validate-evidence-attestations.ps1')
    } else {
        $evidenceHistoryBase = $effectiveHistoryBase
        $evidenceAttestationArguments = @(
            '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $evidenceAttestationValidator,
            '-RepositoryRoot', $repoRoot, '-HistoryBase', $evidenceHistoryBase
        )
        $previousErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        $evidenceAttestationOutput = & $validationShellExecutable @evidenceAttestationArguments 2>&1
        $evidenceAttestationExitCode = $LASTEXITCODE
        $ErrorActionPreference = $previousErrorAction
        if ($evidenceAttestationExitCode -ne 0) {
            $failures.Add("evidence attestation validator exited with code $evidenceAttestationExitCode.")
            foreach ($line in $evidenceAttestationOutput) {
                $failures.Add("evidence attestations: $line")
            }
        }
    }
}
$markdownFiles = Get-ChildItem $docsRoot -Recurse -File -Filter '*.md'
$planIds = @{}
$planStems = @{}
$planMetadata = @{}
$auditIds = @{}
$auditRecords = @{}
$auditMetadata = @{}
$auditRemediationStates = @{}
$acceptanceEvidenceRunIds = @{}
$remediationIds = @{}
$remediationRecords = @{}
$remediationMetadata = @{}
$remediationVerificationStates = @{}
$implementationIds = @{}
$implementationRecords = @{}
$implementationMetadata = @{}
$workingTreeChangedPaths = @()
if ($docsRoot -eq (Join-Path $repoRoot 'docs')) {
    $trackedChanges = @(& $gitExecutable -c core.safecrlf=false -C $repoRoot diff --name-only HEAD 2>$null)
    $untrackedChanges = @(& $gitExecutable -c core.safecrlf=false -C $repoRoot ls-files --others --exclude-standard 2>$null)
    $workingTreeChangedPaths = @($trackedChanges + $untrackedChanges | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
}

$contractHistoryRevision = $null
if ($useGitLedgerChecks) {
    $contractHistorySpec = $effectiveHistoryBase
    if ([string]::IsNullOrWhiteSpace($contractHistorySpec)) {
        $contractHistoryRevision = 'HEAD'
    } else {
        $previousErrorAction = $ErrorActionPreference
        try {
            $ErrorActionPreference = 'Continue'
            $contractHistoryOutput = @(& $gitExecutable -C $repoRoot rev-parse --verify "$contractHistorySpec`^{commit}" 2>$null)
            $contractHistoryExitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousErrorAction
        }
        $contractHistoryCandidate = $contractHistoryOutput | Select-Object -First 1
        if ($contractHistoryExitCode -eq 0 -and $contractHistoryCandidate -match '^[0-9a-fA-F]{40}$') {
            $contractHistoryRevision = $contractHistoryCandidate.ToLowerInvariant()
        } else {
            $failures.Add("Documentation validation requires a valid HistoryBase: $contractHistorySpec")
            $contractHistoryRevision = 'HEAD'
        }
    }
}

$auditsRoot = Join-Path $docsRoot 'audits'
if (Test-Path -LiteralPath $auditsRoot) {
    foreach ($auditFile in Get-ChildItem $auditsRoot -Recurse -File) {
        if ($auditFile.Extension -ne '.md') {
            $failures.Add("Non-Markdown file is not allowed under audits/: $(Get-RepoRelativePath $auditFile.FullName)")
        }
    }
}

$remediationsRoot = Join-Path $docsRoot 'remediations'
if (Test-Path -LiteralPath $remediationsRoot) {
    foreach ($remediationFile in Get-ChildItem $remediationsRoot -Recurse -File) {
        if ($remediationFile.Extension -ne '.md') {
            $failures.Add("Non-Markdown file is not allowed under remediations/: $(Get-RepoRelativePath $remediationFile.FullName)")
        }
    }
}

foreach ($file in $markdownFiles) {
    $relativePath = Get-RepoRelativePath $file.FullName
    $docsRelativePath = Get-DocsRelativePath $file.FullName
    $content = Get-Content $file.FullName -Raw -Encoding UTF8
    $frontmatter = $null
    $isGovernanceRecord = $docsRelativePath -match '^(?:audits|remediations|implementations)/records/'
    $isNewGovernanceRecord = $useGitLedgerChecks -and $isGovernanceRecord -and
        -not (Test-GitPathExistsAtRevision $contractHistoryRevision $relativePath)

    if ($frontmatterExceptions -notcontains $docsRelativePath) {
        if ($content -notmatch '(?s)\A(?:\uFEFF)?---\r?\n(?<frontmatter>.*?)\r?\n---\r?\n') {
            $failures.Add("Missing frontmatter: $relativePath")
        } else {
            $frontmatter = $Matches['frontmatter']
            $requiredFields = @('status', 'owner', 'last_updated', 'applies_to')
            if ($docsRelativePath.StartsWith('decisions/')) {
                $requiredFields = @('status', 'date')
            }
            if ($docsRelativePath -match '^plans/(?!templates/)(?:archived/)?PLN-') {
                $requiredFields = @('status', 'plan_id', 'owner', 'created', 'last_updated', 'applies_to')
            }
            if ($docsRelativePath.StartsWith('audits/records/')) {
                $requiredFields = @('status', 'audit_id', 'auditor', 'audit_type', 'scope', 'subject', 'baseline', 'started_at', 'last_updated')
                if ((Get-FrontmatterValue $frontmatter 'governance_contract') -eq 'audit-loop/v3') {
                    $requiredFields += @('execution_context_id')
                }
                if ((Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -eq 'audit-runtime/v1') {
                    $requiredFields += @(
                        'runtime_context_ref',
                        'evidence_revision', 'evidence_worktree_revision', 'evidence_runner',
                        'evidence_run_id', 'evidence_artifact'
                    )
                    if ($isNewGovernanceRecord) {
                        $requiredFields += @('runtime_context_attestation', 'evidence_attestation', 'evidence_argv_json')
                    }
                }
                if ((Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -eq 'audit-runtime/v1' -and
                    (Get-FrontmatterValue $frontmatter 'audit_schema') -eq 'plan-audit/v2') {
                    $requiredFields += @('audited_peer_plans', 'audited_subject_paths')
                }
                if ((Get-FrontmatterValue $frontmatter 'status') -eq 'superseded') {
                    $requiredFields += @('completed_at', 'superseded_by', 'supersession_reason')
                }
                if ((Get-FrontmatterValue $frontmatter 'audit_schema') -eq 'implementation-acceptance/v2') {
                    $requiredFields += @('acceptance_next_action')
                }
            }
            if ($docsRelativePath.StartsWith('remediations/records/')) {
                $requiredFields = @('status', 'remediation_id', 'implementer', 'scope', 'source_audits', 'source_findings', 'baseline', 'started_at', 'last_updated')
                if ((Get-FrontmatterValue $frontmatter 'remediation_schema') -eq 'remediation/v2') {
                    $requiredFields += @('result_revision', 'affects_implementation', 'related_implementations')
                }
                if ((Get-FrontmatterValue $frontmatter 'governance_contract') -eq 'audit-loop/v3') {
                    $requiredFields += @('execution_context_id', 'parent_result_revision')
                }
                if ((Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -eq 'audit-runtime/v1') {
                    $requiredFields += @('runtime_context_ref')
                    if ($isNewGovernanceRecord) { $requiredFields += @('runtime_context_attestation') }
                }
                if ((Get-FrontmatterValue $frontmatter 'status') -eq 'superseded') {
                    $requiredFields += @('completed_at', 'superseded_by', 'supersession_reason')
                }
            }
            if ($docsRelativePath.StartsWith('implementations/records/')) {
                $requiredFields = @('status', 'implementation_schema', 'implementation_id', 'implementer', 'scope', 'related_plans', 'plan_acceptance_audits', 'trigger_audits', 'plan_evidence_revision', 'baseline', 'result_revision', 'started_at', 'last_updated')
                if ((Get-FrontmatterValue $frontmatter 'governance_contract') -eq 'audit-loop/v3') {
                    $requiredFields += @('execution_context_id')
                }
                if ((Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -eq 'audit-runtime/v1') {
                    $requiredFields += @('runtime_context_ref')
                    if ($isNewGovernanceRecord) { $requiredFields += @('runtime_context_attestation') }
                }
                if ((Get-FrontmatterValue $frontmatter 'status') -eq 'superseded') {
                    $requiredFields += @('completed_at', 'superseded_by', 'supersession_reason')
                }
            }
            if ($isNewGovernanceRecord) {
                if ((Get-FrontmatterValue $frontmatter 'governance_contract') -ne 'audit-loop/v3') {
                    $failures.Add("New governance record must use governance_contract audit-loop/v3: $relativePath")
                }
                if ((Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -ne 'audit-runtime/v1') {
                    $failures.Add("New governance record must use workflow_contract_revision audit-runtime/v1: $relativePath")
                }
            }
            foreach ($field in $requiredFields) {
                if ($frontmatter -notmatch "(?m)^${field}:\s*\S.*$") {
                    $failures.Add("Missing frontmatter field '$field': $relativePath")
                }
            }
            if ($isNewGovernanceRecord -and
                $docsRelativePath.StartsWith('audits/records/') -and
                (Get-FrontmatterValue $frontmatter 'workflow_contract_revision') -eq 'audit-runtime/v1') {
                $declaredArgvJson = Get-FrontmatterValue $frontmatter 'evidence_argv_json'
                if (-not [string]::IsNullOrWhiteSpace($declaredArgvJson) -and
                    -not (ConvertFrom-StrictEvidenceArgvJson $declaredArgvJson).IsValid) {
                    $failures.Add("Audit evidence_argv_json must be strict JSON containing a non-empty array of non-empty strings: $relativePath")
                }
            }
        }
    }

    if ($docsRelativePath -match '^plans/(?<archived>archived/)?(?<stem>PLN-(?<number>\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*?)(?<checklist>-checklist)?\.md$') {
        if ($null -eq $frontmatter) {
            continue
        }
        $planId = "PLN-$($Matches['number'])"
        $declaredPlanId = Get-FrontmatterValue $frontmatter 'plan_id'
        if ($declaredPlanId -ne $planId) {
            $failures.Add("Plan ID does not match filename: $relativePath ($declaredPlanId != $planId)")
        }
        $declaredStatus = Get-FrontmatterValue $frontmatter 'status'
        if ($Matches['archived']) {
            if ($declaredStatus -ne 'archived') {
                $failures.Add("Legacy archived plan directory only allows status=archived: $relativePath ($declaredStatus)")
            }
        } elseif ($declaredStatus -notin @('active', 'archived')) {
            $failures.Add("Stable plan path requires status active or archived: $relativePath ($declaredStatus)")
        }
        $stem = $Matches['stem']
        if (-not $planMetadata.ContainsKey($planId)) {
            $planMetadata[$planId] = @{ Status = $declaredStatus; Stem = $stem; PlanPath = $null; ChecklistPath = $null; PlanDocsPath = $null; ChecklistDocsPath = $null }
        }
        if ($planStems.ContainsKey($planId) -and $planStems[$planId] -ne $stem) {
            $failures.Add("Plan ID is reused by multiple subjects: $planId")
        } else {
            $planStems[$planId] = $stem
        }
        $key = "$planId|$stem"
        if (-not $planIds.ContainsKey($key)) {
            $planIds[$key] = @{ Plan = 0; Checklist = 0 }
        }
        if ($Matches['checklist']) {
            $planIds[$key].Checklist++
            $planMetadata[$planId].ChecklistPath = $relativePath
            $planMetadata[$planId].ChecklistDocsPath = ("docs/$docsRelativePath" -replace '\\', '/')
        } else {
            $planIds[$key].Plan++
            $planMetadata[$planId].PlanPath = $relativePath
            $planMetadata[$planId].PlanDocsPath = ("docs/$docsRelativePath" -replace '\\', '/')
        }
    } elseif ($docsRelativePath.StartsWith('plans/') -and
        $docsRelativePath -notin @('plans/README.md', 'plans/archived/README.md') -and
        -not $docsRelativePath.StartsWith('plans/templates/')) {
        $failures.Add("Invalid plan filename or location: $relativePath")
    }

    if ($docsRelativePath.StartsWith('audits/records/')) {
        if ($docsRelativePath -notmatch '^audits/records/(?<auditId>AUD-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-(?<scopeKind>repository|plan|implementation|feature|control|follow-up)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid audit filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $auditId = $Matches['auditId']
            $auditDate = $Matches['date']
            $scopeKind = $Matches['scopeKind']
            $governanceContract = Get-FrontmatterValue $frontmatter 'governance_contract'
            $executionContextId = Get-FrontmatterValue $frontmatter 'execution_context_id'
            $sourceContextIds = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_context_ids'))
            $runtimeContextRef = Get-FrontmatterValue $frontmatter 'runtime_context_ref'
            $sourceContextRefs = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_context_refs'))
            $workflowContractRevision = Get-FrontmatterValue $frontmatter 'workflow_contract_revision'
            if (-not [string]::IsNullOrWhiteSpace($governanceContract) -and $governanceContract -ne 'audit-loop/v3') {
                $failures.Add("Invalid governance_contract: $relativePath ($governanceContract)")
            }
            if ($governanceContract -eq 'audit-loop/v3' -and -not (Test-UuidV4 $executionContextId)) {
                $failures.Add("audit-loop/v3 record must use a UUIDv4 execution_context_id: $relativePath")
            }
            if (-not [string]::IsNullOrWhiteSpace($workflowContractRevision) -and $workflowContractRevision -ne 'audit-runtime/v1') {
                $failures.Add("Invalid workflow_contract_revision: $relativePath ($workflowContractRevision)")
            }
            if ($workflowContractRevision -eq 'audit-runtime/v1' -and [string]::IsNullOrWhiteSpace($runtimeContextRef)) {
                $failures.Add("audit-loop/v3 record must declare runtime_context_ref: $relativePath")
            }
            if ($governanceContract -eq 'audit-loop/v3' -and
                (Get-FrontmatterValue $frontmatter 'baseline') -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                $failures.Add("audit-loop/v3 audit baseline must be a full git SHA on a clean worktree: $relativePath")
            }
            $declaredAuditId = Get-FrontmatterValue $frontmatter 'audit_id'
            if ($declaredAuditId -ne $auditId) {
                $failures.Add("Audit ID does not match filename: $relativePath ($declaredAuditId != $auditId)")
            }
            if ($auditIds.ContainsKey($auditId)) {
                $failures.Add("Duplicate audit ID: $auditId")
            } else {
                $auditIds[$auditId] = $relativePath
            }
            $auditStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($auditStatus -notin @('open', 'closed', 'superseded')) {
                $failures.Add("Invalid audit status: $relativePath ($auditStatus)")
            }
            if ($auditStatus -in @('closed', 'superseded')) {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Terminal audit must record completed_at: $relativePath")
                }
            }
            if ($auditStatus -eq 'superseded') {
                if ((Get-FrontmatterValue $frontmatter 'superseded_by') -notmatch '^AUD-\d{4}$' -or
                    (Get-FrontmatterValue $frontmatter 'supersession_reason') -notin @('baseline-drift', 'context-loss')) {
                    $failures.Add("Superseded audit must identify its replacement and a valid supersession reason: $relativePath")
                }
            }
            $startedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $startedAtValue = ConvertTo-DateTimeOffsetOrNull $startedAt
            if ($null -eq $startedAtValue) {
                $failures.Add("Audit must record a parseable started_at: $relativePath")
            }
            $expectedDate = "$($auditDate.Substring(0, 4))-$($auditDate.Substring(4, 2))-$($auditDate.Substring(6, 2))"
            if ($startedAt -notlike "$expectedDate*") {
                $failures.Add("Audit filename date does not match started_at: $relativePath")
            }
            if ($auditStatus -in @('closed', 'superseded')) {
                $completedAtValue = ConvertTo-DateTimeOffsetOrNull (Get-FrontmatterValue $frontmatter 'completed_at')
                if ($null -eq $completedAtValue -or ($null -ne $startedAtValue -and $completedAtValue -lt $startedAtValue)) {
                    $failures.Add("Terminal audit completed_at must be parseable and not earlier than started_at: $relativePath")
                }
            }
            $declaredScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($scopeKind -ne 'follow-up' -and $declaredScope -notlike "${scopeKind}:*") {
                $failures.Add("Audit scope does not match filename scope kind: $relativePath")
            }

            if ($scopeKind -eq 'implementation') {
                $auditSchema = Get-FrontmatterValue $frontmatter 'audit_schema'
                $expectedImplementationAuditSchema = if ($governanceContract -eq 'audit-loop/v3') { 'implementation-audit/v2' } else { 'implementation-audit/v1' }
                if ($auditSchema -ne $expectedImplementationAuditSchema) {
                    $failures.Add("Implementation audit must use audit_schema ${expectedImplementationAuditSchema}: $relativePath")
                }
                $relatedImplementations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations'))
                if ($relatedImplementations.Count -eq 0) {
                    $failures.Add("Implementation audit must list related_implementations: $relativePath")
                }
                if ($governanceContract -eq 'audit-loop/v3' -and $relatedImplementations.Count -ne 1) {
                    $failures.Add("audit-loop/v3 implementation audit must identify exactly one IMP: $relativePath")
                } elseif ($governanceContract -eq 'audit-loop/v3' -and $declaredScope -ne "implementation:$($relatedImplementations[0])") {
                    $failures.Add("audit-loop/v3 implementation audit scope must match its IMP: $relativePath")
                }
                foreach ($relatedImplementation in $relatedImplementations) {
                    if ($relatedImplementation -notmatch '^IMP-\d{4}$') {
                        $failures.Add("Invalid related implementation ID in implementation audit: $relativePath ($relatedImplementation)")
                        continue
                    }
                    Test-AuditMatrix $content "<!-- implementation-audit: $relatedImplementation -->" @('IMP_TRACEABILITY', 'CHECKLIST_EVIDENCE', 'CODE_CONTRACT', 'TEST_FAILURE', 'SECURITY_DATA', 'MIGRATION_RECOVERY', 'DOCS_CI_RELEASE') $auditId $failures 'Implementation audit matrix'
                }
                if ($auditSchema -eq 'implementation-audit/v2') {
                    $implementationAuditEvidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                    $implementationAuditEvidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
                    if ((Get-FrontmatterValue $frontmatter 'independence_basis') -ne 'separate-context') {
                        $failures.Add("implementation-audit/v2 must use independence_basis=separate-context: $relativePath")
                    }
                    if ($sourceContextIds.Count -eq 0) {
                        $failures.Add("implementation-audit/v2 must record source_context_ids: $relativePath")
                    }
                    if ($workflowContractRevision -eq 'audit-runtime/v1' -and ($runtimeContextRef -eq 'runtime-unavailable' -or $sourceContextRefs.Count -eq 0)) {
                        $failures.Add("implementation-audit/v2 must record a real runtime_context_ref and source_context_refs: $relativePath")
                    }
                    foreach ($sourceContextRef in $sourceContextRefs) {
                        if ($sourceContextRef -ne 'legacy-unavailable' -and $sourceContextRef -eq $runtimeContextRef) {
                            $failures.Add("Implementation audit runtime context must differ from every source context ref: $relativePath")
                        }
                    }
                    foreach ($sourceContextId in $sourceContextIds) {
                        if ($sourceContextId -ne 'legacy-unavailable' -and -not (Test-UuidV4 $sourceContextId)) {
                            $failures.Add("Invalid source_context_id in implementation audit: $relativePath ($sourceContextId)")
                        }
                        if ($sourceContextId -eq $executionContextId) {
                            $failures.Add("Implementation audit execution context must differ from every source context: $relativePath")
                        }
                    }
                    if ($implementationAuditEvidenceRevision -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                        $failures.Add("implementation-audit/v2 evidence_revision must be a full git SHA on a clean worktree: $relativePath")
                    }
                    if ($workflowContractRevision -eq 'audit-runtime/v1' -and ((Get-FrontmatterValue $frontmatter 'evidence_worktree_revision') -ne "git:$(Get-GitRevision $implementationAuditEvidenceRevision)" -or
                        (Get-FrontmatterValue $frontmatter 'evidence_runner') -ne 'docs/tools/invoke-revision-evidence.ps1')) {
                        $failures.Add("implementation-audit/v2 must bind detached evidence execution to evidence_revision: $relativePath")
                    }
                    if (-not (Test-UuidV4 $implementationAuditEvidenceRunId)) {
                        $failures.Add("implementation-audit/v2 must record a valid UUIDv4 evidence_run_id: $relativePath")
                    } elseif ($acceptanceEvidenceRunIds.ContainsKey($implementationAuditEvidenceRunId.ToLowerInvariant())) {
                        $failures.Add("Independent evidence_run_id must be globally unique: $relativePath ($implementationAuditEvidenceRunId)")
                    } else {
                        $acceptanceEvidenceRunIds[$implementationAuditEvidenceRunId.ToLowerInvariant()] = $auditId
                    }
                }
            } elseif ($scopeKind -eq 'plan') {
                $auditSchema = Get-FrontmatterValue $frontmatter 'audit_schema'
                $legacyPlanAudit = $auditId -in @('AUD-0002', 'AUD-0003')
                if ($legacyPlanAudit) {
                    if ($auditSchema -eq 'plan-audit/v2') {
                        $failures.Add("Legacy plan audit must not claim v2 checklist evidence: $relativePath")
                    }
                } else {
                    $relatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
                    if ($auditSchema -in @('plan-acceptance/v2', 'implementation-acceptance/v2')) {
                        if ($relatedPlans.Count -ne 1) {
                            $failures.Add("Acceptance audit must identify exactly one related plan: $relativePath")
                        }
                        $acceptanceType = Get-FrontmatterValue $frontmatter 'acceptance_type'
                        $expectedAcceptanceType = if ($auditSchema -eq 'plan-acceptance/v2') { 'plan-readiness' } else { 'implementation-completion' }
                        if ($acceptanceType -ne $expectedAcceptanceType) {
                            $failures.Add("Acceptance audit has incorrect acceptance_type: $relativePath")
                        }
                        $acceptanceVerdict = Get-FrontmatterValue $frontmatter 'acceptance_verdict'
                        $acceptanceNextAction = Get-FrontmatterValue $frontmatter 'acceptance_next_action'
                        $validAcceptanceVerdicts = if ($auditSchema -eq 'plan-acceptance/v2') {
                            @('pending', 'ready', 'not-ready', 'blocked', 'superseded')
                        } else {
                            @('pending', 'complete', 'incomplete', 'blocked', 'superseded')
                        }
                        if ($acceptanceVerdict -notin $validAcceptanceVerdicts) {
                            $failures.Add("Acceptance audit has invalid acceptance_verdict: $relativePath ($acceptanceVerdict)")
                        }
                        if ($auditSchema -eq 'implementation-acceptance/v2') {
                            $validNextActions = @('pending', 'none', 'implement', 'implementation-audit', 'remediate', 'decision', 'superseded')
                            if ($acceptanceNextAction -notin $validNextActions) {
                                $failures.Add("Implementation acceptance has invalid acceptance_next_action: $relativePath ($acceptanceNextAction)")
                            }
                            $validVerdictAction =
                                ($acceptanceVerdict -eq 'pending' -and $acceptanceNextAction -eq 'pending') -or
                                ($acceptanceVerdict -eq 'complete' -and $acceptanceNextAction -eq 'none') -or
                                ($acceptanceVerdict -eq 'incomplete' -and $acceptanceNextAction -in @('implement', 'implementation-audit', 'remediate')) -or
                                ($acceptanceVerdict -eq 'blocked' -and $acceptanceNextAction -eq 'decision') -or
                                ($acceptanceVerdict -eq 'superseded' -and $acceptanceNextAction -eq 'superseded')
                            if (-not $validVerdictAction) {
                                $failures.Add("Implementation acceptance verdict and next action are inconsistent: $relativePath ($acceptanceVerdict/$acceptanceNextAction)")
                            }
                        }
                        $planStatusAtAcceptance = Get-FrontmatterValue $frontmatter 'plan_status_at_acceptance'
                        if ($planStatusAtAcceptance -ne 'active') {
                            $failures.Add("Acceptance audit must record plan_status_at_acceptance=active: $relativePath")
                        }
                        $independenceBasis = Get-FrontmatterValue $frontmatter 'independence_basis'
                        if ($governanceContract -eq 'audit-loop/v3' -and $independenceBasis -ne 'separate-context') {
                            $failures.Add("audit-loop/v3 acceptance must use independence_basis=separate-context: $relativePath")
                        } elseif ($governanceContract -ne 'audit-loop/v3' -and $independenceBasis -notin @('separate-auditor', 'fresh-context-independent-rerun')) {
                            $failures.Add("Acceptance audit must record a valid independence_basis: $relativePath")
                        }
                        if ($governanceContract -eq 'audit-loop/v3') {
                            if ($sourceContextIds.Count -eq 0) {
                                $failures.Add("audit-loop/v3 acceptance must record source_context_ids: $relativePath")
                            }
                            if ($workflowContractRevision -eq 'audit-runtime/v1' -and ($runtimeContextRef -eq 'runtime-unavailable' -or $sourceContextRefs.Count -eq 0)) {
                                $failures.Add("audit-loop/v3 acceptance must record a real runtime_context_ref and source_context_refs: $relativePath")
                            }
                            foreach ($sourceContextRef in $sourceContextRefs) {
                                if ($sourceContextRef -ne 'legacy-unavailable' -and $sourceContextRef -eq $runtimeContextRef) {
                                    $failures.Add("Acceptance runtime context must differ from every source context ref: $relativePath")
                                }
                            }
                            foreach ($sourceContextId in $sourceContextIds) {
                                if ($sourceContextId -ne 'legacy-unavailable' -and -not (Test-UuidV4 $sourceContextId)) {
                                    $failures.Add("Invalid source_context_id in acceptance: $relativePath ($sourceContextId)")
                                }
                                if ($sourceContextId -eq $executionContextId) {
                                    $failures.Add("Acceptance execution context must differ from every source context: $relativePath")
                                }
                            }
                        }
                        $acceptanceBaseline = Get-FrontmatterValue $frontmatter 'baseline'
                        $evidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                        $evidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
                        if ($acceptanceBaseline -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                            $failures.Add("Acceptance audit baseline must be a full git SHA on a clean worktree: $relativePath")
                        }
                        if ($evidenceRevision -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                            $failures.Add("Acceptance audit evidence_revision must be a full git SHA on a clean worktree: $relativePath")
                        }
                        if ($workflowContractRevision -eq 'audit-runtime/v1' -and ((Get-FrontmatterValue $frontmatter 'evidence_worktree_revision') -ne "git:$(Get-GitRevision $evidenceRevision)" -or
                            (Get-FrontmatterValue $frontmatter 'evidence_runner') -ne 'docs/tools/invoke-revision-evidence.ps1')) {
                            $failures.Add("Acceptance audit must bind detached evidence execution to evidence_revision: $relativePath")
                        }
                        $acceptanceRevisionSha = Get-GitRevision $acceptanceBaseline
                        $acceptanceEvidenceSha = Get-GitRevision $evidenceRevision
                        if ($docsRoot -eq (Join-Path $repoRoot 'docs') -and
                            $null -ne $acceptanceRevisionSha -and
                            -not (Test-GitCommitExists $acceptanceRevisionSha)) {
                            $failures.Add("Acceptance audit governance baseline must reference an existing commit: $relativePath ($acceptanceRevisionSha)")
                        }
                        if ($docsRoot -eq (Join-Path $repoRoot 'docs') -and
                            $null -ne $acceptanceEvidenceSha -and
                            -not (Test-GitCommitExists $acceptanceEvidenceSha)) {
                            $failures.Add("Acceptance audit evidence revision must reference an existing commit: $relativePath ($acceptanceEvidenceSha)")
                        }
                        if ($evidenceRunId -notmatch '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$') {
                            $failures.Add("Acceptance audit must record a valid UUIDv4 evidence_run_id: $relativePath")
                        } elseif ($acceptanceEvidenceRunIds.ContainsKey($evidenceRunId.ToLowerInvariant())) {
                            $failures.Add("Acceptance evidence_run_id must be globally unique: $relativePath ($evidenceRunId)")
                        } else {
                            $acceptanceEvidenceRunIds[$evidenceRunId.ToLowerInvariant()] = $auditId
                        }
                        $markerPrefix = if ($auditSchema -eq 'plan-acceptance/v2') { 'plan-acceptance-audit' } else { 'implementation-acceptance-audit' }
                        $acceptanceMarkers = [regex]::Matches($content, "<!--\s*$markerPrefix`:\s*PLN-\d{4}\s*-->")
                        if ($acceptanceMarkers.Count -ne 1) {
                            $failures.Add("Acceptance audit v2 must contain exactly one plan matrix: $relativePath")
                        }
                        $controls = if ($auditSchema -eq 'plan-acceptance/v2') {
                            @('READY_IDENTITY', 'READY_SCOPE', 'READY_FACTS', 'READY_DEPENDENCIES', 'READY_DESIGN', 'READY_EVIDENCE', 'READY_GATES', 'PLAN_AUDIT_CHAIN_CLEAN')
                        } else {
                            @('IMP_PRESENT', 'SCOPE_COMPLETE', 'CHECKLIST_COMPLETE', 'VALIDATION_GATES', 'AUDIT_CHAIN_CLEAN', 'RESIDUAL_RISK', 'ARCHIVE_READY')
                        }
                        foreach ($relatedPlan in $relatedPlans) {
                            if ($relatedPlan -notmatch '^PLN-\d{4}$') {
                                $failures.Add("Invalid related plan ID in acceptance audit: $relativePath ($relatedPlan)")
                                continue
                            }
                            Test-AuditMatrix $content "<!-- $markerPrefix`: $relatedPlan -->" $controls $auditId $failures 'Acceptance audit matrix'
                        }
                        if ($relatedPlans.Count -eq 1 -and $declaredScope -ne "plan:$($relatedPlans[0])") {
                            $failures.Add("Acceptance audit scope must match its single related plan: $relativePath")
                        }
                        Test-AcceptanceVerdict $content $null $null $controls $acceptanceVerdict $auditStatus $auditId 'acceptance audit' $failures
                        if ($auditSchema -eq 'implementation-acceptance/v2') {
                            $relatedImplementations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations'))
                            if ($relatedImplementations.Count -gt 1) {
                                $failures.Add("Implementation acceptance audit may identify at most one related implementation: $relativePath")
                            }
                            if ($acceptanceVerdict -eq 'complete' -and $relatedImplementations.Count -ne 1) {
                                $failures.Add("Complete implementation acceptance must identify exactly one related implementation: $relativePath")
                            }
                            $effectiveResultRevision = Get-FrontmatterValue $frontmatter 'effective_result_revision'
                            if ($relatedImplementations.Count -eq 0 -and $effectiveResultRevision -ne 'none') {
                                $failures.Add("Implementation acceptance without an IMP must use effective_result_revision=none: $relativePath")
                            } elseif ($relatedImplementations.Count -eq 0 -and $acceptanceVerdict -eq 'incomplete' -and $acceptanceNextAction -ne 'implement') {
                                $failures.Add("Implementation acceptance without an IMP must use acceptance_next_action=implement: $relativePath")
                            } elseif ($acceptanceVerdict -eq 'complete' -and $effectiveResultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                                $failures.Add("Complete implementation acceptance must record a full effective_result_revision: $relativePath")
                            } elseif ($effectiveResultRevision -match '^git:[0-9a-fA-F]{40}$' -and (Get-GitRevision $effectiveResultRevision) -ne (Get-GitRevision $evidenceRevision)) {
                                $failures.Add("Implementation acceptance effective_result_revision must match evidence_revision: $relativePath")
                            }
                        }
                    } elseif ($auditSchema -ne 'plan-audit/v2') {
                        $failures.Add("Plan audit must use audit_schema plan-audit/v2: $relativePath")
                    } else {
                        if ($relatedPlans.Count -eq 0) {
                            $failures.Add("Plan audit must list related_plans: $relativePath")
                        }
                        if ($governanceContract -eq 'audit-loop/v3' -and $relatedPlans.Count -ne 1) {
                            $failures.Add("audit-loop/v3 plan audit must identify exactly one plan: $relativePath")
                        } elseif ($governanceContract -eq 'audit-loop/v3' -and $declaredScope -ne "plan:$($relatedPlans[0])") {
                            $failures.Add("audit-loop/v3 plan audit scope must match its plan: $relativePath")
                        }
                        foreach ($relatedPlan in $relatedPlans) {
                        if ($relatedPlan -notmatch '^PLN-\d{4}$') {
                            $failures.Add("Invalid related plan ID in plan audit: $relativePath ($relatedPlan)")
                            continue
                        }
                        if ($auditStatus -eq 'superseded') { continue }
                        $matrixMarker = "<!-- plan-checklist-audit: $relatedPlan -->"
                        $matrixMatches = [regex]::Matches($content, [regex]::Escape($matrixMarker))
                        if ($matrixMatches.Count -ne 1) {
                            $failures.Add("Plan audit must contain exactly one checklist matrix for ${relatedPlan}: $relativePath")
                            continue
                        }
                        $markerIndex = $matrixMatches[0].Index
                        $nextMarker = $content.IndexOf('<!-- plan-checklist-audit:', $markerIndex + $matrixMarker.Length, [StringComparison]::Ordinal)
                        $matrixEnd = if ($nextMarker -ge 0) { $nextMarker } else { $content.Length }
                        $matrixContent = $content.Substring($markerIndex, $matrixEnd - $markerIndex)

                        $planLinkPattern = '\]\((?:\.\./)+plans/(?:archived/)?' + [regex]::Escape($relatedPlan) + '-[a-z0-9]+(?:-[a-z0-9]+)*\.md\)'
                        $checklistLinkPattern = '\]\((?:\.\./)+plans/(?:archived/)?' + [regex]::Escape($relatedPlan) + '-[a-z0-9]+(?:-[a-z0-9]+)*-checklist\.md\)'
                        if ($matrixContent -notmatch $planLinkPattern) {
                            $failures.Add("Checklist matrix must link the plan file for ${relatedPlan}: $relativePath")
                        }
                        if ($matrixContent -notmatch $checklistLinkPattern) {
                            $failures.Add("Checklist matrix must link the checklist file for ${relatedPlan}: $relativePath")
                        }

                        foreach ($control in @('PAIRING', 'PLAN_TO_CHECKLIST', 'CHECKLIST_TO_PLAN', 'CHECKED_EVIDENCE', 'GATE_COMPLETENESS', 'ARCHIVE_CLOSURE')) {
                            $controlRows = [regex]::Matches($matrixContent, "(?m)^\|\s*$control\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail|not-applicable)\s*\|(?<finding>[^|]*)\|\s*$")
                            if ($controlRows.Count -ne 1) {
                                $failures.Add("Checklist matrix must contain one valid $control row for ${relatedPlan}: $relativePath")
                                continue
                            }
                            $evidence = $controlRows[0].Groups['evidence'].Value.Trim()
                            $verdict = $controlRows[0].Groups['verdict'].Value
                            $finding = $controlRows[0].Groups['finding'].Value.Trim()
                            if (Test-EvidencePlaceholder $evidence) {
                                $failures.Add("Checklist matrix evidence is empty for $control/${relatedPlan}: $relativePath")
                            }
                            if ($control -ne 'CHECKED_EVIDENCE' -and $verdict -eq 'not-applicable') {
                                $failures.Add("Only CHECKED_EVIDENCE may be not-applicable: $control/${relatedPlan} in $relativePath")
                            }
                            if ($verdict -eq 'fail') {
                                if ($finding -notmatch "^$([regex]::Escape($auditId))-F\d{3}$" -or
                                    $content -notmatch "(?m)^###\s+$([regex]::Escape($finding))\s+-") {
                                    $failures.Add("Failed checklist control must reference an existing finding: $control/${relatedPlan} in $relativePath")
                                }
                            } elseif ($finding -ne 'none') {
                                $failures.Add("Passing checklist control must use finding=none: $control/${relatedPlan} in $relativePath")
                            }
                        }
                        }
                        if ($governanceContract -eq 'audit-loop/v3') {
                            $planAuditEvidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                            $planAuditEvidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
                            $auditedPeerPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'audited_peer_plans'))
                            $auditedSubjectPaths = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'audited_subject_paths'))
                            if ($planAuditEvidenceRevision -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                                $failures.Add("audit-loop/v3 plan audit evidence_revision must be a full git SHA on a clean worktree: $relativePath")
                            }
                            if ($auditedSubjectPaths.Count -lt 2) {
                                $failures.Add("audit-loop/v3 plan audit must record plan/checklist audited_subject_paths: $relativePath")
                            }
                            $directFactPaths = @($auditedSubjectPaths | Where-Object {
                                $_ -notmatch '^docs/plans/PLN-\d{4}-.+\.md$' -and
                                $_ -notmatch '^docs/plans/PLN-\d{4}-.+-checklist\.md$'
                            })
                            if ($directFactPaths.Count -eq 0) {
                                $failures.Add("audit-loop/v3 plan audit must record at least one direct fact-source/code/config path: $relativePath")
                            }
                            if ($workflowContractRevision -eq 'audit-runtime/v1' -and ($auditedPeerPlans.Count -eq 0 -or $auditedPeerPlans -notcontains $relatedPlans[0])) {
                                $failures.Add("audit-loop/v3 plan audit must record an audited_peer_plans set containing its subject: $relativePath")
                            }
                            if ($workflowContractRevision -eq 'audit-runtime/v1') {
                                if (-not (Test-UuidV4 $planAuditEvidenceRunId)) {
                                    $failures.Add("audit-loop/v3 plan audit must record a valid UUIDv4 evidence_run_id: $relativePath")
                                } elseif ($acceptanceEvidenceRunIds.ContainsKey($planAuditEvidenceRunId.ToLowerInvariant())) {
                                    $failures.Add("Independent evidence_run_id must be globally unique: $relativePath ($planAuditEvidenceRunId)")
                                } else {
                                    $acceptanceEvidenceRunIds[$planAuditEvidenceRunId.ToLowerInvariant()] = $auditId
                                }
                            }
                            if ($workflowContractRevision -eq 'audit-runtime/v1' -and $docsRoot -eq (Join-Path $repoRoot 'docs')) {
                                $planAuditEvidenceSha = Get-GitRevision $planAuditEvidenceRevision
                                $expectedActivePeers = @(Get-ActivePlanIdsAtRevision $planAuditEvidenceSha)
                                $normalizedAuditedPeers = @($auditedPeerPlans | Sort-Object -Unique)
                                if (($expectedActivePeers -join ',') -ne ($normalizedAuditedPeers -join ',')) {
                                    $failures.Add("Plan audit must snapshot the active peer set at evidence_revision: $relativePath (expected=$($expectedActivePeers -join ','); actual=$($normalizedAuditedPeers -join ','))")
                                }
                            }
                            foreach ($auditedPeerPlan in $auditedPeerPlans) {
                                $normalizedAuditedSubjectPaths = @($auditedSubjectPaths | ForEach-Object { $_ -replace '\\', '/' })
                                $peerPlanPathCount = @($normalizedAuditedSubjectPaths | Where-Object { $_ -match "^docs/plans/$([regex]::Escape($auditedPeerPlan))-.+\.md$" -and $_ -notmatch '-checklist\.md$' }).Count
                                $peerChecklistPathCount = @($normalizedAuditedSubjectPaths | Where-Object { $_ -match "^docs/plans/$([regex]::Escape($auditedPeerPlan))-.+-checklist\.md$" }).Count
                                if ($peerPlanPathCount -eq 0 -or $peerChecklistPathCount -eq 0) {
                                    $failures.Add("Plan audit peer snapshot must include each peer plan/checklist path: $relativePath ($auditedPeerPlan; actual=$($normalizedAuditedSubjectPaths -join ','))")
                                }
                            }
                            if ($workflowContractRevision -eq 'audit-runtime/v1' -and ((Get-FrontmatterValue $frontmatter 'evidence_worktree_revision') -ne "git:$(Get-GitRevision $planAuditEvidenceRevision)" -or
                                (Get-FrontmatterValue $frontmatter 'evidence_runner') -ne 'docs/tools/invoke-revision-evidence.ps1')) {
                                $failures.Add("Plan audit must bind detached evidence execution to evidence_revision: $relativePath")
                            }
                            foreach ($auditedPath in $auditedSubjectPaths) {
                                if ($auditedPath -match '(^|/)(?:\.\.?)(?:/|$)' -or $auditedPath -match '[*?\[\]]' -or $auditedPath.StartsWith('/') -or $auditedPath.EndsWith('/')) {
                                    $failures.Add("Invalid audited_subject_path in plan audit: $relativePath ($auditedPath)")
                                }
                            }
                            if ($docsRoot -eq (Join-Path $repoRoot 'docs')) {
                                $planAuditEvidenceSha = Get-GitRevision $planAuditEvidenceRevision
                                $planAuditBaselineSha = Get-GitRevision (Get-FrontmatterValue $frontmatter 'baseline')
                                if ($null -ne $planAuditEvidenceSha -and -not (Test-GitCommitExists $planAuditEvidenceSha)) {
                                    $failures.Add("Plan audit evidence revision must reference an existing commit: $relativePath ($planAuditEvidenceSha)")
                                }
                                if ($null -ne $planAuditEvidenceSha -and $null -ne $planAuditBaselineSha -and
                                    (Test-GitCommitExists $planAuditEvidenceSha) -and (Test-GitCommitExists $planAuditBaselineSha)) {
                                    if (-not (Test-GitAncestor $planAuditEvidenceSha $planAuditBaselineSha)) {
                                        $failures.Add("Plan audit baseline must descend from evidence_revision: $relativePath")
                                    } else {
                                        & $gitExecutable -C $repoRoot diff --quiet $planAuditEvidenceSha $planAuditBaselineSha -- $auditedSubjectPaths
                                        if ($LASTEXITCODE -eq 1) {
                                            $failures.Add("Plan audit audited_subject_paths drifted between evidence_revision and baseline: $relativePath")
                                        } elseif ($LASTEXITCODE -ne 0) {
                                            $failures.Add("Plan audit audited_subject_paths comparison failed: $relativePath")
                                        }
                                    }
                                    foreach ($auditedPath in $auditedSubjectPaths) {
                                        if (-not (Test-GitPathExistsAtRevision $planAuditEvidenceSha $auditedPath)) {
                                            $failures.Add("Plan audit audited_subject_path is missing at evidence_revision: $relativePath ($auditedPath)")
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            if ($governanceContract -eq 'audit-loop/v3' -and (Get-FrontmatterValue $frontmatter 'audit_type') -eq 'follow-up') {
                $independenceBasis = Get-FrontmatterValue $frontmatter 'independence_basis'
                $evidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
                $followUpEvidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                if ($independenceBasis -ne 'separate-context') {
                    $failures.Add("audit-loop/v3 follow-up must use independence_basis=separate-context: $relativePath")
                }
                if ($sourceContextIds.Count -eq 0) {
                    $failures.Add("audit-loop/v3 follow-up must record source_context_ids: $relativePath")
                }
                if ($workflowContractRevision -eq 'audit-runtime/v1' -and ($runtimeContextRef -eq 'runtime-unavailable' -or $sourceContextRefs.Count -eq 0)) {
                    $failures.Add("audit-loop/v3 follow-up must record a real runtime_context_ref and source_context_refs: $relativePath")
                }
                foreach ($sourceContextRef in $sourceContextRefs) {
                    if ($sourceContextRef -ne 'legacy-unavailable' -and $sourceContextRef -eq $runtimeContextRef) {
                        $failures.Add("Follow-up runtime context must differ from every source context ref: $relativePath")
                    }
                }
                $followUpRemediations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_remediations'))
                if ($followUpRemediations.Count -ne 1 -or $declaredScope -ne "follow-up:$($followUpRemediations[0])") {
                    $failures.Add("audit-loop/v3 follow-up must identify exactly one matching REM: $relativePath")
                }
                foreach ($sourceContextId in $sourceContextIds) {
                    if ($sourceContextId -ne 'legacy-unavailable' -and -not (Test-UuidV4 $sourceContextId)) {
                        $failures.Add("Invalid source_context_id in follow-up: $relativePath ($sourceContextId)")
                    }
                    if ($sourceContextId -eq $executionContextId) {
                        $failures.Add("Follow-up execution context must differ from every source context: $relativePath")
                    }
                }
                if (-not (Test-UuidV4 $evidenceRunId)) {
                    $failures.Add("audit-loop/v3 follow-up must record a valid UUIDv4 evidence_run_id: $relativePath")
                } elseif ($acceptanceEvidenceRunIds.ContainsKey($evidenceRunId.ToLowerInvariant())) {
                    $failures.Add("Independent evidence_run_id must be globally unique: $relativePath ($evidenceRunId)")
                } else {
                    $acceptanceEvidenceRunIds[$evidenceRunId.ToLowerInvariant()] = $auditId
                }
                if ($followUpEvidenceRevision -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                    $failures.Add("audit-loop/v3 follow-up must record a full evidence_revision: $relativePath")
                }
                if ($workflowContractRevision -eq 'audit-runtime/v1' -and ((Get-FrontmatterValue $frontmatter 'evidence_worktree_revision') -ne "git:$(Get-GitRevision $followUpEvidenceRevision)" -or
                    (Get-FrontmatterValue $frontmatter 'evidence_runner') -ne 'docs/tools/invoke-revision-evidence.ps1')) {
                    $failures.Add("Follow-up must bind detached evidence execution to evidence_revision: $relativePath")
                }
            }
            if ($governanceContract -eq 'audit-loop/v3') {
                Test-FindingDetails $content $auditId $failures 'audit-loop/v3'
            }
            $auditMetadata[$auditId] = @{
                Path = $relativePath
                IsNew = $isNewGovernanceRecord
                Status = $auditStatus
                Auditor = Get-FrontmatterValue $frontmatter 'auditor'
                AuditType = Get-FrontmatterValue $frontmatter 'audit_type'
                Schema = Get-FrontmatterValue $frontmatter 'audit_schema'
                Verdict = Get-FrontmatterValue $frontmatter 'acceptance_verdict'
                NextAction = Get-FrontmatterValue $frontmatter 'acceptance_next_action'
                PlanStatusAtAcceptance = Get-FrontmatterValue $frontmatter 'plan_status_at_acceptance'
                IndependenceBasis = Get-FrontmatterValue $frontmatter 'independence_basis'
                Baseline = Get-FrontmatterValue $frontmatter 'baseline'
                EvidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                AuditedSubjectPaths = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'audited_subject_paths'))
                AuditedPeerPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'audited_peer_plans'))
                EvidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
                EvidenceArtifact = Get-FrontmatterValue $frontmatter 'evidence_artifact'
                EvidenceAttestation = Get-FrontmatterValue $frontmatter 'evidence_attestation'
                EvidenceArgvJson = Get-FrontmatterValue $frontmatter 'evidence_argv_json'
                EffectiveResultRevision = Get-FrontmatterValue $frontmatter 'effective_result_revision'
                GovernanceContract = $governanceContract
                WorkflowContractRevision = $workflowContractRevision
                ExecutionContextId = $executionContextId
                SourceContextIds = $sourceContextIds
                RuntimeContextRef = $runtimeContextRef
                SourceContextRefs = $sourceContextRefs
                StartedAt = Get-FrontmatterValue $frontmatter 'started_at'
                CompletedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                SupersededBy = Get-FrontmatterValue $frontmatter 'superseded_by'
                SupersessionReason = Get-FrontmatterValue $frontmatter 'supersession_reason'
                RelatedAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_audits'))
                RelatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
                RelatedImplementations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations'))
                RelatedRemediations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_remediations'))
                Supersedes = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'supersedes'))
                Scope = $declaredScope
                Content = $content
            }
            $auditRecords[$file.Name] = $auditStatus
        }
    }

    if ($docsRelativePath.StartsWith('remediations/records/')) {
        if ($docsRelativePath -notmatch '^remediations/records/(?<remediationId>REM-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-(?<scopeKind>audit|repository|plan|feature)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid remediation filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $remediationId = $Matches['remediationId']
            $remediationDate = $Matches['date']
            $remediationScopeKind = $Matches['scopeKind']
            $declaredRemediationId = Get-FrontmatterValue $frontmatter 'remediation_id'
            if ($declaredRemediationId -ne $remediationId) {
                $failures.Add("Remediation ID does not match filename: $relativePath ($declaredRemediationId != $remediationId)")
            }
            if ($remediationIds.ContainsKey($remediationId)) {
                $failures.Add("Duplicate remediation ID: $remediationId")
            } else {
                $remediationIds[$remediationId] = $relativePath
            }
            $remediationStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($remediationStatus -notin @('in-progress', 'completed', 'partial', 'blocked', 'superseded')) {
                $failures.Add("Invalid remediation status: $relativePath ($remediationStatus)")
            }
            if ($remediationStatus -ne 'in-progress') {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Closed remediation must record completed_at: $relativePath")
                }
            }
            if ($remediationStatus -eq 'superseded' -and
                ((Get-FrontmatterValue $frontmatter 'superseded_by') -notmatch '^REM-\d{4}$' -or
                 (Get-FrontmatterValue $frontmatter 'supersession_reason') -ne 'context-loss')) {
                $failures.Add("Superseded remediation must identify a replacement and context-loss reason: $relativePath")
            }
            $remediationStartedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $remediationStartedAtValue = ConvertTo-DateTimeOffsetOrNull $remediationStartedAt
            if ($null -eq $remediationStartedAtValue) {
                $failures.Add("Remediation must record a parseable started_at: $relativePath")
            }
            if ($remediationStatus -ne 'in-progress') {
                $remediationCompletedAtValue = ConvertTo-DateTimeOffsetOrNull (Get-FrontmatterValue $frontmatter 'completed_at')
                if ($null -eq $remediationCompletedAtValue -or ($null -ne $remediationStartedAtValue -and $remediationCompletedAtValue -lt $remediationStartedAtValue)) {
                    $failures.Add("Closed remediation completed_at must be parseable and not earlier than started_at: $relativePath")
                }
            }
            $expectedRemediationDate = "$($remediationDate.Substring(0, 4))-$($remediationDate.Substring(4, 2))-$($remediationDate.Substring(6, 2))"
            if ($remediationStartedAt -notlike "$expectedRemediationDate*") {
                $failures.Add("Remediation filename date does not match started_at: $relativePath")
            }
            $declaredRemediationScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($declaredRemediationScope -notlike "${remediationScopeKind}:*") {
                $failures.Add("Remediation scope does not match filename scope kind: $relativePath")
            }
            $remediationSchema = Get-FrontmatterValue $frontmatter 'remediation_schema'
            $remediationContract = Get-FrontmatterValue $frontmatter 'governance_contract'
            $remediationExecutionContextId = Get-FrontmatterValue $frontmatter 'execution_context_id'
            $remediationRuntimeContextRef = Get-FrontmatterValue $frontmatter 'runtime_context_ref'
            $remediationResultRevision = Get-FrontmatterValue $frontmatter 'result_revision'
            $parentResultRevision = Get-FrontmatterValue $frontmatter 'parent_result_revision'
            $affectsImplementation = Get-FrontmatterValue $frontmatter 'affects_implementation'
            $remediationRelatedImplementations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations'))
            if (-not [string]::IsNullOrWhiteSpace($remediationSchema)) {
                if ($remediationSchema -ne 'remediation/v2') {
                    $failures.Add("Invalid remediation_schema: $relativePath ($remediationSchema)")
                }
                if ($affectsImplementation -notin @('true', 'false')) {
                    $failures.Add("remediation/v2 must record affects_implementation as true or false: $relativePath")
                }
                if ($remediationStatus -in @('completed', 'partial') -and $remediationResultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                    $failures.Add("Completed or partial remediation/v2 must record a full result_revision: $relativePath")
                }
                $remediationRevisionSha = Get-GitRevision $remediationResultRevision
                if ($remediationStatus -in @('completed', 'partial') -and
                    $docsRoot -eq (Join-Path $repoRoot 'docs') -and
                    $null -ne $remediationRevisionSha -and
                    -not (Test-GitCommitExists $remediationRevisionSha)) {
                    $failures.Add("remediation/v2 result_revision must reference an existing commit: $relativePath ($remediationRevisionSha)")
                }
                if ($affectsImplementation -eq 'true' -and $remediationRelatedImplementations.Count -eq 0) {
                    $failures.Add("Implementation-affecting remediation must list related_implementations: $relativePath")
                }
            }
            if (-not [string]::IsNullOrWhiteSpace($remediationContract) -and $remediationContract -ne 'audit-loop/v3') {
                $failures.Add("Invalid remediation governance_contract: $relativePath ($remediationContract)")
            }
            if ($remediationContract -eq 'audit-loop/v3') {
                if (-not (Test-UuidV4 $remediationExecutionContextId)) {
                    $failures.Add("audit-loop/v3 remediation must use a UUIDv4 execution_context_id: $relativePath")
                }
                if ((Get-FrontmatterValue $frontmatter 'baseline') -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                    $failures.Add("audit-loop/v3 remediation baseline must be a full git SHA on a clean worktree: $relativePath")
                }
                $remediationRelatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
                if ($remediationRelatedPlans.Count -gt 1 -or $remediationRelatedImplementations.Count -gt 1) {
                    $failures.Add("audit-loop/v3 remediation cannot span multiple plans or IMPs: $relativePath")
                }
                if ($affectsImplementation -eq 'true') {
                    if ($parentResultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                        $failures.Add("Implementation remediation must record a full parent_result_revision: $relativePath")
                    } elseif ($remediationStatus -in @('completed', 'partial') -and
                        $remediationResultRevision -match '^git:[0-9a-fA-F]{40}$' -and
                        $docsRoot -eq (Join-Path $repoRoot 'docs') -and
                        -not (Test-GitAncestor (Get-GitRevision $parentResultRevision) (Get-GitRevision $remediationResultRevision))) {
                        $failures.Add("Implementation remediation result_revision must descend from parent_result_revision: $relativePath")
                    }
                } elseif ($parentResultRevision -ne 'none') {
                    $failures.Add("Non-implementation remediation must use parent_result_revision=none: $relativePath")
                }
            }
            $remediationMetadata[$remediationId] = @{
                Path = $relativePath
                Status = $remediationStatus
                Schema = $remediationSchema
                Implementer = Get-FrontmatterValue $frontmatter 'implementer'
                ResultRevision = $remediationResultRevision
                ParentResultRevision = $parentResultRevision
                GovernanceContract = $remediationContract
                ExecutionContextId = $remediationExecutionContextId
                RuntimeContextRef = $remediationRuntimeContextRef
                AffectsImplementation = $affectsImplementation -eq 'true'
                SourceAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_audits'))
                SourceFindings = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_findings'))
                Scope = $declaredRemediationScope
                Baseline = Get-FrontmatterValue $frontmatter 'baseline'
                RelatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
                RelatedImplementations = $remediationRelatedImplementations
                Supersedes = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'supersedes'))
                SupersededBy = Get-FrontmatterValue $frontmatter 'superseded_by'
                SupersessionReason = Get-FrontmatterValue $frontmatter 'supersession_reason'
            }
            $remediationRecords[$file.Name] = $remediationStatus
        }
    }

    if ($docsRelativePath.StartsWith('implementations/records/')) {
        if ($docsRelativePath -notmatch '^implementations/records/(?<implementationId>IMP-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-plan-(?<planId>pln-\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid implementation filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $implementationId = $Matches['implementationId']
            $implementationDate = $Matches['date']
            $filenamePlanId = "PLN-$($Matches['planId'].Substring(4))"
            $declaredImplementationId = Get-FrontmatterValue $frontmatter 'implementation_id'
            if ($declaredImplementationId -ne $implementationId) {
                $failures.Add("Implementation ID does not match filename: $relativePath ($declaredImplementationId != $implementationId)")
            }
            if ($implementationIds.ContainsKey($implementationId)) {
                $failures.Add("Duplicate implementation ID: $implementationId")
            } else {
                $implementationIds[$implementationId] = $relativePath
            }
            $implementationStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($implementationStatus -notin @('in-progress', 'completed', 'partial', 'blocked', 'superseded')) {
                $failures.Add("Invalid implementation status: $relativePath ($implementationStatus)")
            }
            if ($implementationStatus -ne 'in-progress') {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Closed implementation must record completed_at: $relativePath")
                }
            }
            if ($implementationStatus -eq 'superseded' -and
                ((Get-FrontmatterValue $frontmatter 'superseded_by') -notmatch '^IMP-\d{4}$' -or
                 (Get-FrontmatterValue $frontmatter 'supersession_reason') -ne 'context-loss')) {
                $failures.Add("Superseded implementation must identify a replacement and context-loss reason: $relativePath")
            }
            $implementationStartedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $implementationStartedAtValue = ConvertTo-DateTimeOffsetOrNull $implementationStartedAt
            if ($null -eq $implementationStartedAtValue) {
                $failures.Add("Implementation must record a parseable started_at: $relativePath")
            }
            if ($implementationStatus -ne 'in-progress') {
                $implementationCompletedAtValue = ConvertTo-DateTimeOffsetOrNull (Get-FrontmatterValue $frontmatter 'completed_at')
                if ($null -eq $implementationCompletedAtValue -or ($null -ne $implementationStartedAtValue -and $implementationCompletedAtValue -lt $implementationStartedAtValue)) {
                    $failures.Add("Closed implementation completed_at must be parseable and not earlier than started_at: $relativePath")
                }
            }
            $expectedImplementationDate = "$($implementationDate.Substring(0, 4))-$($implementationDate.Substring(4, 2))-$($implementationDate.Substring(6, 2))"
            if ($implementationStartedAt -notlike "$expectedImplementationDate*") {
                $failures.Add("Implementation filename date does not match started_at: $relativePath")
            }
            $declaredImplementationScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($declaredImplementationScope -notlike 'plan:*') {
                $failures.Add("Implementation scope must start with plan:: $relativePath")
            }
            $relatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
            if ($relatedPlans.Count -ne 1 -or $relatedPlans[0] -notmatch '^PLN-\d{4}$') {
                $failures.Add("Implementation must identify exactly one matching related plan: $relativePath")
            } elseif ($relatedPlans[0] -ne $filenamePlanId) {
                $failures.Add("Implementation related plan does not match filename: $relativePath ($($relatedPlans[0]) != $filenamePlanId)")
            }
            $planAcceptanceAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'plan_acceptance_audits'))
            $triggerAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'trigger_audits'))
            if ($planAcceptanceAudits.Count -ne 1) {
                $failures.Add("implementation/v2 must reference exactly one plan acceptance audit: $relativePath")
            }
            foreach ($planAcceptanceAudit in $planAcceptanceAudits) {
                if ($planAcceptanceAudit -notmatch '^AUD-\d{4}$') {
                    $failures.Add("Invalid plan acceptance audit ID in implementation: $relativePath ($planAcceptanceAudit)")
                }
            }
            foreach ($triggerAudit in $triggerAudits) {
                if ($triggerAudit -notmatch '^AUD-\d{4}$') {
                    $failures.Add("Invalid trigger audit ID in implementation: $relativePath ($triggerAudit)")
                }
            }
            $resultRevision = Get-FrontmatterValue $frontmatter 'result_revision'
            if ($implementationStatus -in @('completed', 'partial') -and $resultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                $failures.Add("Completed or partial implementation must record a full git result_revision: $relativePath")
            }
            $implementationSchema = Get-FrontmatterValue $frontmatter 'implementation_schema'
            $implementationContract = Get-FrontmatterValue $frontmatter 'governance_contract'
            $implementationExecutionContextId = Get-FrontmatterValue $frontmatter 'execution_context_id'
            $implementationRuntimeContextRef = Get-FrontmatterValue $frontmatter 'runtime_context_ref'
            if ($implementationSchema -ne 'implementation/v2') {
                $failures.Add("Implementation must use implementation_schema implementation/v2: $relativePath")
            }
            $planEvidenceRevision = Get-FrontmatterValue $frontmatter 'plan_evidence_revision'
            if ($planEvidenceRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                $failures.Add("Implementation must record a full plan_evidence_revision: $relativePath")
            }
            $implementationBaseline = Get-FrontmatterValue $frontmatter 'baseline'
            if ($implementationBaseline -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                $failures.Add("implementation/v2 baseline must be a full git SHA on a clean worktree: $relativePath")
            }
            if (-not [string]::IsNullOrWhiteSpace($implementationContract) -and $implementationContract -ne 'audit-loop/v3') {
                $failures.Add("Invalid implementation governance_contract: $relativePath ($implementationContract)")
            }
            if ($implementationContract -eq 'audit-loop/v3' -and -not (Test-UuidV4 $implementationExecutionContextId)) {
                $failures.Add("audit-loop/v3 implementation must use a UUIDv4 execution_context_id: $relativePath")
            }
            if ($docsRoot -eq (Join-Path $repoRoot 'docs')) {
                foreach ($revisionField in @($planEvidenceRevision, $implementationBaseline, $resultRevision)) {
                    $revisionSha = Get-GitRevision $revisionField
                    if ($null -ne $revisionSha -and -not (Test-GitCommitExists $revisionSha)) {
                        $failures.Add("Implementation revision must reference an existing commit: $relativePath ($revisionSha)")
                    }
                }
            }
            $implementationMetadata[$implementationId] = @{
                Path = $relativePath
                Schema = $implementationSchema
                Status = $implementationStatus
                Implementer = Get-FrontmatterValue $frontmatter 'implementer'
                PlanId = if ($relatedPlans.Count -eq 1) { $relatedPlans[0] } else { $filenamePlanId }
                PlanAcceptanceAudits = $planAcceptanceAudits
                TriggerAudits = $triggerAudits
                PlanEvidenceRevision = $planEvidenceRevision
                Baseline = $implementationBaseline
                StartedAt = $implementationStartedAt
                ResultRevision = $resultRevision
                GovernanceContract = $implementationContract
                ExecutionContextId = $implementationExecutionContextId
                RuntimeContextRef = $implementationRuntimeContextRef
                Supersedes = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'supersedes'))
                SupersededBy = Get-FrontmatterValue $frontmatter 'superseded_by'
                SupersessionReason = Get-FrontmatterValue $frontmatter 'supersession_reason'
            }
            $implementationRecords[$file.Name] = $implementationStatus
        }
    }

    foreach ($match in [regex]::Matches($content, '\]\((?<target>[^)]+)\)')) {
        $target = $match.Groups['target'].Value.Trim()
        if ($target.StartsWith('<') -and $target.EndsWith('>')) {
            $target = $target.Substring(1, $target.Length - 2)
        }
        if ($target -match '^[A-Za-z][A-Za-z0-9+.-]*:' -or $target.StartsWith('#')) {
            continue
        }
        $targetPath = $target.Split('#')[0].Split('?')[0]
        if ([string]::IsNullOrWhiteSpace($targetPath)) {
            continue
        }
        $targetPath = [Uri]::UnescapeDataString($targetPath)
        $resolvedTarget = [System.IO.Path]::GetFullPath((Join-Path $file.DirectoryName $targetPath))
        if (-not (Test-Path -LiteralPath $resolvedTarget)) {
            $failures.Add("Missing relative link target: $relativePath -> $target")
        }
    }
}

$openAuditKeys = @{}
foreach ($auditEntry in $auditMetadata.GetEnumerator()) {
    $auditInfo = $auditEntry.Value
    if ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -or $auditInfo.Status -ne 'open') { continue }
    $key = "$($auditInfo.Schema)|$($auditInfo.AuditType)|$($auditInfo.Scope)|$($auditInfo.Baseline)"
    if ($openAuditKeys.ContainsKey($key)) {
        $failures.Add("Duplicate open audit-loop/v3 work for the same subject and baseline: $($openAuditKeys[$key]), $($auditEntry.Key)")
    } else {
        $openAuditKeys[$key] = $auditEntry.Key
    }
}

foreach ($auditEntry in $auditMetadata.GetEnumerator()) {
    $auditInfo = $auditEntry.Value
    if ($auditInfo.Status -ne 'superseded') { continue }
    $replacementId = $auditInfo.SupersededBy
    if (-not $auditMetadata.ContainsKey($replacementId)) {
        $failures.Add("Superseded audit references a missing replacement: $($auditInfo.Path) ($replacementId)")
        continue
    }
    if ((Get-AuditNumber $replacementId) -le (Get-AuditNumber $auditEntry.Key)) {
        $failures.Add("Superseded audit replacement must be newer: $($auditInfo.Path) ($replacementId)")
    }
    if ($auditMetadata[$replacementId].Supersedes -notcontains $auditEntry.Key) {
        $failures.Add("Replacement audit must list the superseded audit: $($auditInfo.Path) ($replacementId)")
    }
    $replacement = $auditMetadata[$replacementId]
    if ($replacement.Schema -ne $auditInfo.Schema -or $replacement.AuditType -ne $auditInfo.AuditType -or $replacement.Scope -ne $auditInfo.Scope) {
        $failures.Add("Replacement audit must preserve audit type, schema, and scope: $($auditInfo.Path) ($replacementId)")
    }
    if ($auditInfo.SupersessionReason -eq 'baseline-drift' -and
        $replacement.Baseline -eq $auditInfo.Baseline -and
        $replacement.EvidenceRevision -eq $auditInfo.EvidenceRevision) {
        $failures.Add("baseline-drift replacement must change baseline or evidence revision: $($auditInfo.Path) ($replacementId)")
    }
    if ($auditInfo.SupersessionReason -eq 'context-loss') {
        $isIndependentAudit = $auditInfo.WorkflowContractRevision -eq 'audit-runtime/v1' -and
            ($auditInfo.Schema -in @('plan-acceptance/v2', 'implementation-audit/v2', 'implementation-acceptance/v2') -or $auditInfo.AuditType -eq 'follow-up')
        if (-not $isIndependentAudit) {
            $failures.Add("context-loss supersession is only valid for independent audit-runtime/v1 audits: $($auditInfo.Path)")
        }
        if ([string]::IsNullOrWhiteSpace($auditInfo.RuntimeContextRef) -or $auditInfo.RuntimeContextRef -eq 'runtime-unavailable') {
            $failures.Add("context-loss supersession requires a real lost runtime_context_ref: $($auditInfo.Path)")
        }
        if ($replacement.Baseline -ne $auditInfo.Baseline -or $replacement.EvidenceRevision -ne $auditInfo.EvidenceRevision) {
            $failures.Add("context-loss replacement must preserve baseline and evidence revision: $($auditInfo.Path) ($replacementId)")
        }
        if ([string]::IsNullOrWhiteSpace($replacement.RuntimeContextRef) -or
            $replacement.RuntimeContextRef -eq 'runtime-unavailable' -or
            $replacement.RuntimeContextRef -eq $auditInfo.RuntimeContextRef) {
            $failures.Add("context-loss replacement must use a different real runtime_context_ref: $($auditInfo.Path) ($replacementId)")
        }
    }
}

foreach ($remediationEntry in $remediationMetadata.GetEnumerator()) {
    $remediationInfo = $remediationEntry.Value
    if ($remediationInfo.Status -ne 'superseded') { continue }
    $replacementId = $remediationInfo.SupersededBy
    if (-not $remediationMetadata.ContainsKey($replacementId)) {
        $failures.Add("Superseded remediation references a missing replacement: $($remediationInfo.Path) ($replacementId)")
        continue
    }
    $replacement = $remediationMetadata[$replacementId]
    if ((Get-RemediationNumber $replacementId) -le (Get-RemediationNumber $remediationEntry.Key) -or
        $replacement.Supersedes -notcontains $remediationEntry.Key) {
        $failures.Add("Replacement remediation must be newer and list the superseded REM: $($remediationInfo.Path) ($replacementId)")
    }
    if ($remediationInfo.SupersessionReason -ne 'context-loss' -or
        $replacement.Scope -ne $remediationInfo.Scope -or
        $replacement.Baseline -ne $remediationInfo.Baseline -or
        $replacement.RuntimeContextRef -eq $remediationInfo.RuntimeContextRef -or
        $replacement.ExecutionContextId -eq $remediationInfo.ExecutionContextId) {
        $failures.Add("context-loss remediation replacement must preserve scope/baseline and use a different runtime context: $($remediationInfo.Path) ($replacementId)")
    }
}

foreach ($implementationEntry in $implementationMetadata.GetEnumerator()) {
    $implementationInfo = $implementationEntry.Value
    if ($implementationInfo.Status -ne 'superseded') { continue }
    $replacementId = $implementationInfo.SupersededBy
    if (-not $implementationMetadata.ContainsKey($replacementId)) {
        $failures.Add("Superseded implementation references a missing replacement: $($implementationInfo.Path) ($replacementId)")
        continue
    }
    $replacement = $implementationMetadata[$replacementId]
    if ((Get-ImplementationNumber $replacementId) -le (Get-ImplementationNumber $implementationEntry.Key) -or
        $replacement.Supersedes -notcontains $implementationEntry.Key) {
        $failures.Add("Replacement implementation must be newer and list the superseded IMP: $($implementationInfo.Path) ($replacementId)")
    }
    if ($implementationInfo.SupersessionReason -ne 'context-loss' -or
        $replacement.PlanId -ne $implementationInfo.PlanId -or
        $replacement.Baseline -ne $implementationInfo.Baseline -or
        $replacement.RuntimeContextRef -eq $implementationInfo.RuntimeContextRef -or
        $replacement.ExecutionContextId -eq $implementationInfo.ExecutionContextId) {
        $failures.Add("context-loss implementation replacement must preserve plan/baseline and use a different runtime context: $($implementationInfo.Path) ($replacementId)")
    }
}

$inProgressRemediationKeys = @{}
foreach ($remediationEntry in $remediationMetadata.GetEnumerator()) {
    $remediationInfo = $remediationEntry.Value
    if ($remediationInfo.GovernanceContract -ne 'audit-loop/v3' -or $remediationInfo.Status -ne 'in-progress') { continue }
    $key = "$($remediationInfo.Scope)|$($remediationInfo.Baseline)|$($remediationInfo.SourceFindings -join ',')"
    if ($inProgressRemediationKeys.ContainsKey($key)) {
        $failures.Add("Duplicate in-progress audit-loop/v3 remediation: $($inProgressRemediationKeys[$key]), $($remediationEntry.Key)")
    } else {
        $inProgressRemediationKeys[$key] = $remediationEntry.Key
    }
}

foreach ($entry in $planIds.GetEnumerator()) {
    if ($entry.Value.Plan -ne 1 -or $entry.Value.Checklist -ne 1) {
        $failures.Add("Plan/checklist pair is incomplete or duplicated: $($entry.Key)")
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -or $auditInfo.Schema -ne 'plan-audit/v2') { continue }
    if ($auditInfo.RelatedPlans.Count -ne 1) { continue }
    $evidenceSha = Get-GitRevision $auditInfo.EvidenceRevision
    $evidencePlanSnapshot = Get-GitPlanSnapshotAtRevision $evidenceSha
    if (-not $evidencePlanSnapshot.ContainsKey($auditInfo.RelatedPlans[0])) { continue }
    foreach ($requiredSubjectPath in @($evidencePlanSnapshot[$auditInfo.RelatedPlans[0]].PlanPath, $evidencePlanSnapshot[$auditInfo.RelatedPlans[0]].ChecklistPath)) {
        if ([string]::IsNullOrWhiteSpace($requiredSubjectPath) -or $auditInfo.AuditedSubjectPaths -notcontains $requiredSubjectPath) {
            $failures.Add("Plan audit audited_subject_paths must include the resolved plan/checklist: $($auditInfo.Path) ($requiredSubjectPath)")
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    foreach ($relatedPlan in @($auditInfo.RelatedPlans)) {
        if (-not $planMetadata.ContainsKey($relatedPlan)) {
            $failures.Add("Audit references a missing plan: $($auditInfo.Path) ($relatedPlan)")
        }
    }
    foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
        if (-not $implementationMetadata.ContainsKey($relatedImplementation)) {
            $failures.Add("Audit references a missing implementation: $($auditInfo.Path) ($relatedImplementation)")
        }
    }
    foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($relatedAudit)) {
            $failures.Add("Audit references a missing related audit: $($auditInfo.Path) ($relatedAudit)")
        }
        if ($auditInfo.Schema -in @('plan-acceptance/v2', 'implementation-acceptance/v2')) {
            $currentAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
            if ($auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
                $auditMetadata.ContainsKey($relatedAudit) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$relatedAudit])) {
                $failures.Add("Acceptance audit baseline must contain the referenced terminal audit blob: $($auditInfo.Path) ($relatedAudit)")
            } elseif ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $relatedAudit) -ge (Get-AuditNumber $currentAuditId)) {
                $failures.Add("Acceptance audit cannot reference a future or same-number audit: $($auditInfo.Path) ($relatedAudit)")
            }
        }
    }
    foreach ($relatedRemediation in @($auditInfo.RelatedRemediations)) {
        if (-not $remediationMetadata.ContainsKey($relatedRemediation)) {
            $failures.Add("Audit references a missing related remediation: $($auditInfo.Path) ($relatedRemediation)")
        }
    }
    if ($auditInfo.AuditType -eq 'follow-up') {
        if ($auditInfo.RelatedAudits.Count -eq 0) {
            $failures.Add("Follow-up audit must list source related_audits: $($auditInfo.Path)")
        }
        if ($auditInfo.RelatedRemediations.Count -eq 0) {
            $failures.Add("Follow-up audit must list related_remediations: $($auditInfo.Path)")
        }
    }
    if ($auditInfo.Schema -eq 'plan-acceptance/v2') {
        foreach ($relatedPlan in @($auditInfo.RelatedPlans)) {
            $currentAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
            $currentAuditNumber = Get-AuditNumber $currentAuditId
            $matchingPlanAudits = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.Schema -eq 'plan-audit/v2' -and
                $_.Value.GovernanceContract -eq 'audit-loop/v3' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.AuditedSubjectPaths.Count -ge 2 -and
                (Get-GitRevision $_.Value.EvidenceRevision) -ne $null -and
                $_.Value.RelatedPlans -contains $relatedPlan -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-DisplayRecordAvailableBefore $_.Value $auditInfo)) -or
                 ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $_.Key) -lt $currentAuditNumber))
            })
            if ($matchingPlanAudits.Count -eq 0) {
                $failures.Add("Plan acceptance must reference a matching closed plan-audit/v2 record: $($auditInfo.Path) ($relatedPlan)")
            } else {
                $latestPlanAuditEntry = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    Select-LatestGitRecord $matchingPlanAudits (Get-GitRevision $auditInfo.Baseline) $failures "Plan acceptance source plan audit"
                } else { $matchingPlanAudits | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                if ($null -eq $latestPlanAuditEntry) { continue }
                $latestPlanAuditId = $latestPlanAuditEntry.Key
                if ($auditInfo.RelatedAudits -notcontains $latestPlanAuditId) {
                    $failures.Add("Plan acceptance must reference the latest matching plan audit: $($auditInfo.Path) ($relatedPlan=$latestPlanAuditId)")
                }
                $latestPlanAudit = $latestPlanAuditEntry.Value
                $acceptanceEvidenceSnapshot = Get-GitPlanSnapshotAtRevision (Get-GitRevision $auditInfo.EvidenceRevision)
                if ($acceptanceEvidenceSnapshot.ContainsKey($relatedPlan)) {
                    foreach ($requiredSubjectPath in @($acceptanceEvidenceSnapshot[$relatedPlan].PlanPath, $acceptanceEvidenceSnapshot[$relatedPlan].ChecklistPath)) {
                        if ($latestPlanAudit.AuditedSubjectPaths -notcontains $requiredSubjectPath) {
                            $failures.Add("Plan acceptance source audit omits required audited subject path: $($auditInfo.Path) ($latestPlanAuditId/$requiredSubjectPath)")
                        }
                    }
                }
                if ($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    $expectedCurrentPeers = @(Get-ActivePlanIdsAtRevision (Get-GitRevision $auditInfo.EvidenceRevision))
                    $auditedPeers = @($latestPlanAudit.AuditedPeerPlans | Sort-Object -Unique)
                    if (($expectedCurrentPeers -join ',') -ne ($auditedPeers -join ',')) {
                        $failures.Add("Plan readiness source audit peer snapshot drifted at acceptance evidence_revision: $($auditInfo.Path) (expected=$($expectedCurrentPeers -join ','); actual=$($auditedPeers -join ','))")
                    }
                }
                if ($docsRoot -eq (Join-Path $repoRoot 'docs')) {
                    $planAuditEvidenceSha = Get-GitRevision $latestPlanAudit.EvidenceRevision
                    $acceptanceEvidenceSha = Get-GitRevision $auditInfo.EvidenceRevision
                    if ($null -ne $planAuditEvidenceSha -and $null -ne $acceptanceEvidenceSha -and
                        (Test-GitCommitExists $planAuditEvidenceSha) -and (Test-GitCommitExists $acceptanceEvidenceSha)) {
                        foreach ($auditedSubjectPath in @($latestPlanAudit.AuditedSubjectPaths)) {
                            & $gitExecutable -C $repoRoot diff --quiet $planAuditEvidenceSha $acceptanceEvidenceSha -- $auditedSubjectPath
                            if ($LASTEXITCODE -eq 1) {
                                $failures.Add("Plan acceptance cannot use a stale plan audit after subject drift: $($auditInfo.Path) ($latestPlanAuditId/$auditedSubjectPath)")
                            } elseif ($LASTEXITCODE -ne 0) {
                                $failures.Add("Plan acceptance subject drift comparison failed: $($auditInfo.Path) ($latestPlanAuditId/$auditedSubjectPath)")
                            }
                        }
                    }
                }
                if ($auditInfo.IndependenceBasis -eq 'separate-auditor' -and
                    $auditInfo.Auditor -eq $latestPlanAudit.Auditor) {
                    $failures.Add("Separate-auditor plan acceptance must use a different auditor: $($auditInfo.Path) ($latestPlanAuditId)")
                }
            }
            $matchingFollowUps = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.AuditType -eq 'follow-up' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedPlans -contains $relatedPlan -and
                (Test-PlanReadinessChainAudit $_.Key (New-Object 'System.Collections.Generic.HashSet[string]')) -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-DisplayRecordAvailableBefore $_.Value $auditInfo)) -or
                 ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $_.Key) -lt $currentAuditNumber))
            })
            if ($matchingFollowUps.Count -gt 0) {
                $latestFollowUp = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    Select-LatestGitRecord $matchingFollowUps (Get-GitRevision $auditInfo.Baseline) $failures "Plan acceptance follow-up chain"
                } else { $matchingFollowUps | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                if ($null -ne $latestFollowUp -and $auditInfo.RelatedAudits -notcontains $latestFollowUp.Key) {
                    $failures.Add("Plan acceptance must reference the latest related follow-up audit: $($auditInfo.Path) ($relatedPlan=$($latestFollowUp.Key))")
                }
            }
            foreach ($relatedRemediation in @($auditInfo.RelatedRemediations)) {
                if ($remediationMetadata.ContainsKey($relatedRemediation) -and
                    $auditInfo.IndependenceBasis -eq 'separate-auditor' -and
                    $auditInfo.Auditor -eq $remediationMetadata[$relatedRemediation].Implementer) {
                    $failures.Add("Separate-auditor plan acceptance must differ from related remediation implementers: $($auditInfo.Path) ($relatedRemediation)")
                }
            }
        }
    }
    if ($auditInfo.Schema -eq 'implementation-acceptance/v2') {
        foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
            $currentAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
            $currentAuditNumber = Get-AuditNumber $currentAuditId
            $requiredImplementationAuditSchema = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') { 'implementation-audit/v2' } else { 'implementation-audit/v1' }
            $matchingImplementationAudits = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.Schema -eq $requiredImplementationAuditSchema -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedImplementations -contains $relatedImplementation -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-DisplayRecordAvailableBefore $_.Value $auditInfo)) -or
                 ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $_.Key) -lt $currentAuditNumber))
            })
            if ($auditInfo.Verdict -eq 'complete' -and $matchingImplementationAudits.Count -eq 0) {
                $failures.Add("Implementation acceptance must reference a matching closed $requiredImplementationAuditSchema record: $($auditInfo.Path) ($relatedImplementation)")
            } elseif ($matchingImplementationAudits.Count -gt 0) {
                $latestImplementationAudit = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    Select-LatestGitRecord $matchingImplementationAudits (Get-GitRevision $auditInfo.Baseline) $failures "Implementation acceptance source implementation audit"
                } else { $matchingImplementationAudits | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                if ($null -eq $latestImplementationAudit) { continue }
                $latestImplementationAuditId = $latestImplementationAudit.Key
                if ($auditInfo.RelatedAudits -notcontains $latestImplementationAuditId) {
                    $failures.Add("Implementation acceptance must reference the latest implementation audit: $($auditInfo.Path) ($relatedImplementation=$latestImplementationAuditId)")
                }
                if ($auditInfo.IndependenceBasis -eq 'separate-auditor' -and
                    $auditInfo.Auditor -eq $latestImplementationAudit.Value.Auditor) {
                    $failures.Add("Separate-auditor implementation acceptance must use a different implementation auditor: $($auditInfo.Path) ($latestImplementationAuditId)")
                }
            }
            if ($auditInfo.NextAction -eq 'implementation-audit' -and $matchingImplementationAudits.Count -gt 0) {
                $failures.Add("acceptance_next_action=implementation-audit requires the implementation audit to be missing: $($auditInfo.Path) ($relatedImplementation)")
            }
            if ($auditInfo.NextAction -eq 'remediate' -and $matchingImplementationAudits.Count -eq 0) {
                $failures.Add("acceptance_next_action=remediate requires a prior implementation audit: $($auditInfo.Path) ($relatedImplementation)")
            }
            if ($implementationMetadata.ContainsKey($relatedImplementation)) {
                $implementationInfo = $implementationMetadata[$relatedImplementation]
                if ($auditInfo.Verdict -eq 'complete' -and $implementationInfo.Status -ne 'completed') {
                    $failures.Add("Implementation acceptance requires a completed IMP: $($auditInfo.Path) ($relatedImplementation=$($implementationInfo.Status))")
                }
                if ($auditInfo.NextAction -eq 'implementation-audit' -and $implementationInfo.Status -ne 'completed') {
                    $failures.Add("acceptance_next_action=implementation-audit requires a completed IMP: $($auditInfo.Path) ($relatedImplementation)")
                }
                if ($auditInfo.IndependenceBasis -eq 'separate-auditor' -and
                    $auditInfo.Auditor -eq $implementationInfo.Implementer) {
                    $failures.Add("Separate-auditor implementation acceptance must differ from the implementer: $($auditInfo.Path) ($relatedImplementation)")
                }
            }
        }
        if ($auditInfo.Verdict -eq 'complete' -and $auditInfo.RelatedImplementations.Count -ne 1) {
            $failures.Add("Complete implementation acceptance requires exactly one IMP: $($auditInfo.Path)")
        }
        if ($auditInfo.RelatedPlans.Count -eq 1) {
            $relatedPlan = $auditInfo.RelatedPlans[0]
            $planImplementations = @($implementationMetadata.GetEnumerator() | Where-Object {
                $_.Value.PlanId -eq $relatedPlan -and $_.Value.Status -ne 'superseded' -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
                  (ConvertTo-DateTimeOffsetOrNull $_.Value.StartedAt) -le (ConvertTo-DateTimeOffsetOrNull $auditInfo.StartedAt)) -or
                 $auditInfo.GovernanceContract -ne 'audit-loop/v3')
            })
            if ($planImplementations.Count -eq 0 -and $auditInfo.RelatedImplementations.Count -ne 0) {
                $failures.Add("Implementation acceptance must use related_implementations=none when the plan has no IMP: $($auditInfo.Path) ($relatedPlan)")
            }
            if ($planImplementations.Count -gt 0) {
                $latestPlanImplementation = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    Select-LatestGitRecord $planImplementations (Get-GitRevision $auditInfo.Baseline) $failures "Implementation acceptance plan implementation"
                } else { $planImplementations | Sort-Object { Get-ImplementationNumber $_.Key } | Select-Object -Last 1 }
                if ($null -eq $latestPlanImplementation) { continue }
                if ($auditInfo.RelatedImplementations.Count -ne 1) {
                    $failures.Add("Implementation acceptance must identify the latest IMP when one exists: $($auditInfo.Path) ($relatedPlan=$($latestPlanImplementation.Key))")
                } else {
                    $relatedImplementation = $auditInfo.RelatedImplementations[0]
                    if (-not $implementationMetadata.ContainsKey($relatedImplementation) -or
                        $implementationMetadata[$relatedImplementation].PlanId -ne $relatedPlan) {
                        $failures.Add("Implementation acceptance must map its IMP to its single related plan: $($auditInfo.Path) ($relatedPlan/$relatedImplementation)")
                    }
                    if ($latestPlanImplementation.Key -ne $relatedImplementation) {
                        $failures.Add("Implementation acceptance must target the latest IMP for the plan: $($auditInfo.Path) ($relatedPlan=$($latestPlanImplementation.Key), selected=$relatedImplementation)")
                    }
                }
            }
        }
        foreach ($relatedRemediation in @($auditInfo.RelatedRemediations)) {
            if ($remediationMetadata.ContainsKey($relatedRemediation) -and
                $auditInfo.IndependenceBasis -eq 'separate-auditor' -and
                $auditInfo.Auditor -eq $remediationMetadata[$relatedRemediation].Implementer) {
                $failures.Add("Separate-auditor implementation acceptance must differ from related remediation implementers: $($auditInfo.Path) ($relatedRemediation)")
            }
        }
        foreach ($relatedPlan in @($auditInfo.RelatedPlans)) {
            $currentAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
            $currentAuditNumber = Get-AuditNumber $currentAuditId
            $matchingPlanAcceptances = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.Schema -eq 'plan-acceptance/v2' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedPlans -contains $relatedPlan -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-DisplayRecordAvailableBefore $_.Value $auditInfo)) -or
                 ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $_.Key) -lt $currentAuditNumber))
            })
            if ($matchingPlanAcceptances.Count -eq 0) {
                $failures.Add("Implementation acceptance requires a prior plan acceptance: $($auditInfo.Path) ($relatedPlan)")
                continue
            }
            $latestPlanAcceptance = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                Select-LatestGitRecord $matchingPlanAcceptances (Get-GitRevision $auditInfo.Baseline) $failures "Implementation acceptance source plan acceptance"
            } else { $matchingPlanAcceptances | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
            if ($null -eq $latestPlanAcceptance) { continue }
            if ($latestPlanAcceptance.Value.Verdict -ne 'ready') {
                $failures.Add("Implementation acceptance requires the latest plan acceptance to be ready: $($auditInfo.Path) ($relatedPlan=$($latestPlanAcceptance.Key))")
            }
            if ($auditInfo.RelatedAudits -notcontains $latestPlanAcceptance.Key) {
                $failures.Add("Implementation acceptance must reference the latest plan acceptance: $($auditInfo.Path) ($relatedPlan=$($latestPlanAcceptance.Key))")
            }
            $matchingFollowUps = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.AuditType -eq 'follow-up' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedPlans -contains $relatedPlan -and
                (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
                 (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and (Test-DisplayRecordAvailableBefore $_.Value $auditInfo)) -or
                 ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $_.Key) -lt $currentAuditNumber))
            })
            if ($matchingFollowUps.Count -gt 0) {
                $latestFollowUp = if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    Select-LatestGitRecord $matchingFollowUps (Get-GitRevision $auditInfo.Baseline) $failures "Implementation acceptance follow-up chain"
                } else { $matchingFollowUps | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                if ($null -ne $latestFollowUp -and $auditInfo.RelatedAudits -notcontains $latestFollowUp.Key) {
                    $failures.Add("Implementation acceptance must reference the latest related follow-up audit: $($auditInfo.Path) ($relatedPlan=$($latestFollowUp.Key))")
                }
            }
        }
    }
    if ($auditInfo.Schema -in @('implementation-audit/v1', 'implementation-audit/v2')) {
        foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
            if ($implementationMetadata.ContainsKey($relatedImplementation) -and
                $implementationMetadata[$relatedImplementation].Status -ne 'completed') {
                $failures.Add("Implementation audit may only reference a completed IMP: $($auditInfo.Path) ($relatedImplementation=$($implementationMetadata[$relatedImplementation].Status))")
            }
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -or
        ($auditInfo.Schema -notin @('plan-acceptance/v2', 'implementation-acceptance/v2', 'implementation-audit/v2') -and $auditInfo.AuditType -ne 'follow-up')) {
        continue
    }
    $expectedSourceContexts = New-Object System.Collections.Generic.List[string]
    $expectedSourceContextRefs = New-Object System.Collections.Generic.List[string]
    $hasLegacySource = $false
    $hasLegacySourceRef = $false
    foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($relatedAudit)) { continue }
        $context = $auditMetadata[$relatedAudit].ExecutionContextId
        if (Test-UuidV4 $context) { $expectedSourceContexts.Add($context) } else { $hasLegacySource = $true }
        $contextRef = $auditMetadata[$relatedAudit].RuntimeContextRef
        if ([string]::IsNullOrWhiteSpace($contextRef) -or $contextRef -eq 'runtime-unavailable') { $hasLegacySourceRef = $true } else { $expectedSourceContextRefs.Add($contextRef) }
    }
    foreach ($supersededAudit in @($auditInfo.Supersedes)) {
        if (-not $auditMetadata.ContainsKey($supersededAudit)) { continue }
        $context = $auditMetadata[$supersededAudit].ExecutionContextId
        if (Test-UuidV4 $context) { $expectedSourceContexts.Add($context) } else { $hasLegacySource = $true }
        $contextRef = $auditMetadata[$supersededAudit].RuntimeContextRef
        if ([string]::IsNullOrWhiteSpace($contextRef) -or $contextRef -eq 'runtime-unavailable') { $hasLegacySourceRef = $true } else { $expectedSourceContextRefs.Add($contextRef) }
    }
    foreach ($relatedRemediation in @($auditInfo.RelatedRemediations)) {
        if (-not $remediationMetadata.ContainsKey($relatedRemediation)) { continue }
        $context = $remediationMetadata[$relatedRemediation].ExecutionContextId
        if (Test-UuidV4 $context) { $expectedSourceContexts.Add($context) } else { $hasLegacySource = $true }
        $contextRef = $remediationMetadata[$relatedRemediation].RuntimeContextRef
        if ([string]::IsNullOrWhiteSpace($contextRef) -or $contextRef -eq 'runtime-unavailable') { $hasLegacySourceRef = $true } else { $expectedSourceContextRefs.Add($contextRef) }
    }
    foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
        if (-not $implementationMetadata.ContainsKey($relatedImplementation)) { continue }
        $context = $implementationMetadata[$relatedImplementation].ExecutionContextId
        if (Test-UuidV4 $context) { $expectedSourceContexts.Add($context) } else { $hasLegacySource = $true }
        $contextRef = $implementationMetadata[$relatedImplementation].RuntimeContextRef
        if ([string]::IsNullOrWhiteSpace($contextRef) -or $contextRef -eq 'runtime-unavailable') { $hasLegacySourceRef = $true } else { $expectedSourceContextRefs.Add($contextRef) }
    }
    $expectedSourceContexts = @($expectedSourceContexts | Select-Object -Unique)
    $expectedSourceContextRefs = @($expectedSourceContextRefs | Select-Object -Unique)
    foreach ($expectedContext in $expectedSourceContexts) {
        if ($auditInfo.SourceContextIds -notcontains $expectedContext) {
            $failures.Add("Independent audit must include every available source execution context: $($auditInfo.Path) ($expectedContext)")
        }
    }
    if ($hasLegacySource -and $auditInfo.SourceContextIds -notcontains 'legacy-unavailable') {
        $failures.Add("Independent audit with legacy sources must record legacy-unavailable: $($auditInfo.Path)")
    }
    if (-not $hasLegacySource -and $auditInfo.SourceContextIds -contains 'legacy-unavailable') {
        $failures.Add("Independent audit must not claim legacy-unavailable when all source contexts exist: $($auditInfo.Path)")
    }
    foreach ($sourceContext in @($auditInfo.SourceContextIds | Where-Object { $_ -ne 'legacy-unavailable' })) {
        if ($expectedSourceContexts -notcontains $sourceContext) {
            $failures.Add("Independent audit lists an unrelated source_context_id: $($auditInfo.Path) ($sourceContext)")
        }
    }
    if ($auditInfo.WorkflowContractRevision -eq 'audit-runtime/v1') {
        foreach ($expectedContextRef in $expectedSourceContextRefs) {
            if ($auditInfo.SourceContextRefs -notcontains $expectedContextRef) {
                $failures.Add("Independent audit must include every available source runtime context ref: $($auditInfo.Path) ($expectedContextRef)")
            }
        }
        if ($hasLegacySourceRef -and $auditInfo.SourceContextRefs -notcontains 'legacy-unavailable') {
            $failures.Add("Independent audit with legacy runtime sources must record legacy-unavailable: $($auditInfo.Path)")
        }
        if (-not $hasLegacySourceRef -and $auditInfo.SourceContextRefs -contains 'legacy-unavailable') {
            $failures.Add("Independent audit must not claim legacy-unavailable when all source runtime refs exist: $($auditInfo.Path)")
        }
        foreach ($sourceContextRef in @($auditInfo.SourceContextRefs | Where-Object { $_ -ne 'legacy-unavailable' })) {
            if ($expectedSourceContextRefs -notcontains $sourceContextRef) {
                $failures.Add("Independent audit lists an unrelated source_context_ref: $($auditInfo.Path) ($sourceContextRef)")
            }
        }
    }

    if ($docsRoot -eq (Join-Path $repoRoot 'docs')) {
        $governanceBaselineSha = Get-GitRevision $auditInfo.Baseline
        $subjectEvidenceSha = Get-GitRevision $auditInfo.EvidenceRevision
        if ($null -ne $governanceBaselineSha -and (Test-GitCommitExists $governanceBaselineSha)) {
            foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
                if ($auditMetadata.ContainsKey($relatedAudit) -and -not (Test-GitPathExistsAtRevision $governanceBaselineSha $auditMetadata[$relatedAudit].Path)) {
                    $failures.Add("Independent audit governance baseline must contain its source audit: $($auditInfo.Path) ($relatedAudit)")
                } elseif ($auditMetadata.ContainsKey($relatedAudit) -and
                    $auditMetadata[$relatedAudit].Status -in @('closed', 'superseded') -and
                    -not (Test-GitPathMatchesWorkingTree $governanceBaselineSha $auditMetadata[$relatedAudit].Path)) {
                    $failures.Add("Independent audit governance baseline must contain the terminal source audit content: $($auditInfo.Path) ($relatedAudit)")
                }
            }
            foreach ($supersededAudit in @($auditInfo.Supersedes)) {
                if ($auditMetadata.ContainsKey($supersededAudit) -and -not (Test-GitPathExistsAtRevision $governanceBaselineSha $auditMetadata[$supersededAudit].Path)) {
                    $failures.Add("Independent audit governance baseline must contain its superseded predecessor: $($auditInfo.Path) ($supersededAudit)")
                }
            }
            foreach ($relatedRemediation in @($auditInfo.RelatedRemediations)) {
                if ($remediationMetadata.ContainsKey($relatedRemediation) -and -not (Test-GitPathExistsAtRevision $governanceBaselineSha $remediationMetadata[$relatedRemediation].Path)) {
                    $failures.Add("Independent audit governance baseline must contain its source remediation: $($auditInfo.Path) ($relatedRemediation)")
                } elseif ($remediationMetadata.ContainsKey($relatedRemediation) -and
                    $remediationMetadata[$relatedRemediation].Status -in @('completed', 'partial', 'blocked') -and
                    -not (Test-GitPathMatchesWorkingTree $governanceBaselineSha $remediationMetadata[$relatedRemediation].Path)) {
                    $failures.Add("Independent audit governance baseline must contain the terminal source remediation content: $($auditInfo.Path) ($relatedRemediation)")
                }
            }
            foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
                if ($implementationMetadata.ContainsKey($relatedImplementation) -and -not (Test-GitPathExistsAtRevision $governanceBaselineSha $implementationMetadata[$relatedImplementation].Path)) {
                    $failures.Add("Independent audit governance baseline must contain its source implementation: $($auditInfo.Path) ($relatedImplementation)")
                } elseif ($implementationMetadata.ContainsKey($relatedImplementation) -and
                    $implementationMetadata[$relatedImplementation].Status -in @('completed', 'partial', 'blocked') -and
                    -not (Test-GitPathMatchesWorkingTree $governanceBaselineSha $implementationMetadata[$relatedImplementation].Path)) {
                    $failures.Add("Independent audit governance baseline must contain the terminal source implementation content: $($auditInfo.Path) ($relatedImplementation)")
                }
            }
        }
        if ($null -ne $governanceBaselineSha -and $null -ne $subjectEvidenceSha -and
            (Test-GitCommitExists $governanceBaselineSha) -and (Test-GitCommitExists $subjectEvidenceSha)) {
            if (-not (Test-GitAncestor $subjectEvidenceSha $governanceBaselineSha)) {
                $failures.Add("Independent audit governance baseline must descend from its subject evidence revision: $($auditInfo.Path)")
            }
            $postEvidenceChanges = @(& $gitExecutable -C $repoRoot diff --name-only $subjectEvidenceSha $governanceBaselineSha 2>$null)
            $expectedEvidenceArtifactPath = if (Test-UuidV4 $auditInfo.EvidenceRunId) {
                "docs/evidence/runs/$($auditInfo.EvidenceRunId.ToLowerInvariant())/evidence.json"
            } else { $null }
            $nonGovernanceChanges = @($postEvidenceChanges | Where-Object {
                $_ -notmatch '^docs/(?:audits|remediations|implementations)/(?:README\.md|records/.+\.md)$' -and
                $_ -ne $expectedEvidenceArtifactPath
            })
            if ($nonGovernanceChanges.Count -gt 0) {
                $failures.Add("Independent audit subject drifted between evidence revision and governance baseline: $($auditInfo.Path) ($($nonGovernanceChanges -join ', '))")
            }
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.Schema -eq 'implementation-audit/v2' -and $auditInfo.RelatedImplementations.Count -eq 1) {
        $implementationId = $auditInfo.RelatedImplementations[0]
        if ($implementationMetadata.ContainsKey($implementationId)) {
            $implementationInfo = $implementationMetadata[$implementationId]
            if ((Get-GitRevision $auditInfo.EvidenceRevision) -ne (Get-GitRevision $implementationInfo.ResultRevision)) {
                $failures.Add("implementation-audit/v2 evidence_revision must match IMP result_revision: $($auditInfo.Path) ($implementationId)")
            }
            foreach ($planAcceptanceAudit in @($implementationInfo.PlanAcceptanceAudits)) {
                if ($auditInfo.RelatedAudits -notcontains $planAcceptanceAudit) {
                    $failures.Add("implementation-audit/v2 must reference the IMP ready plan acceptance: $($auditInfo.Path) ($planAcceptanceAudit)")
                }
            }
        }
    }
    if ($auditInfo.AuditType -eq 'follow-up' -and $auditInfo.RelatedRemediations.Count -eq 1) {
        $remediationId = $auditInfo.RelatedRemediations[0]
        if ($remediationMetadata.ContainsKey($remediationId) -and
            (Get-GitRevision $auditInfo.EvidenceRevision) -ne (Get-GitRevision $remediationMetadata[$remediationId].ResultRevision)) {
            $failures.Add("Follow-up evidence_revision must match remediation result_revision: $($auditInfo.Path) ($remediationId)")
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.GovernanceContract -ne 'audit-loop/v3') { continue }
    if ($auditInfo.Status -eq 'closed') {
        $validationSection = [regex]::Match($auditInfo.Content, '(?s)##\s+验证结果\s*(?<body>.*?)(?=\r?\n##\s+|\z)')
        if (-not $validationSection.Success -or
            $validationSection.Groups['body'].Value -notmatch '`[^`\r\n]+`' -or
            $validationSection.Groups['body'].Value -notmatch '(通过|失败|exit|ExitCode|result|结果)') {
            $failures.Add("Closed audit-loop/v3 record must contain concrete command and result evidence: $($auditInfo.Path)")
        }
    }
    if ($useGitLedgerChecks -and $auditInfo.WorkflowContractRevision -eq 'audit-runtime/v1' -and $auditInfo.Status -eq 'closed') {
        $requireSuccessfulEvidence =
            ($auditInfo.Schema -eq 'plan-acceptance/v2' -and $auditInfo.Verdict -eq 'ready') -or
            ($auditInfo.Schema -eq 'implementation-acceptance/v2' -and $auditInfo.Verdict -eq 'complete')
        Test-RevisionEvidenceArtifact ([regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value) $auditInfo $requireSuccessfulEvidence $failures
    }
    if ($auditInfo.Status -eq 'closed' -and
        (($auditInfo.Schema -eq 'plan-acceptance/v2' -and $auditInfo.Verdict -eq 'ready') -or
         ($auditInfo.Schema -eq 'implementation-acceptance/v2' -and $auditInfo.Verdict -eq 'complete'))) {
        $acceptanceLabel = if ($auditInfo.Schema -eq 'plan-acceptance/v2') { 'plan readiness acceptance' } else { 'implementation completion acceptance' }
        $acceptanceAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
        Test-SubjectSpecificValidation $auditInfo $acceptanceAuditId $failures $acceptanceLabel
    }
    if ($auditInfo.Schema -in @('plan-acceptance/v2', 'implementation-acceptance/v2') -and
        $workingTreeChangedPaths -contains $auditInfo.Path) {
        $expectedEvidenceArtifactPath = if (Test-UuidV4 $auditInfo.EvidenceRunId) {
            "docs/evidence/runs/$($auditInfo.EvidenceRunId.ToLowerInvariant())/evidence.json"
        } else { $null }
        $disallowedChanges = @($workingTreeChangedPaths | Where-Object {
            $_ -ne 'docs/audits/README.md' -and
            $_ -notmatch '^docs/audits/records/AUD-\d{4}-.+\.md$' -and
            $_ -ne $expectedEvidenceArtifactPath
        })
        if ($disallowedChanges.Count -gt 0) {
            $failures.Add("New or updated acceptance audit requires a clean subject worktree; unexpected changes: $($disallowedChanges -join ', ')")
        }
    }
}

foreach ($implementationInfo in $implementationMetadata.Values) {
    if (-not $planMetadata.ContainsKey($implementationInfo.PlanId)) {
        $failures.Add("Implementation references a missing plan: $($implementationInfo.Path) ($($implementationInfo.PlanId))")
    }
    foreach ($planAcceptanceAudit in @($implementationInfo.PlanAcceptanceAudits)) {
        if (-not $auditMetadata.ContainsKey($planAcceptanceAudit)) {
            $failures.Add("Implementation references a missing plan acceptance audit: $($implementationInfo.Path) ($planAcceptanceAudit)")
            continue
        }
        $acceptanceInfo = $auditMetadata[$planAcceptanceAudit]
        if ($acceptanceInfo.Schema -ne 'plan-acceptance/v2' -or
            $acceptanceInfo.Status -ne 'closed' -or
            $acceptanceInfo.Verdict -ne 'ready' -or
            $acceptanceInfo.RelatedPlans -notcontains $implementationInfo.PlanId) {
            $failures.Add("Implementation plan acceptance audit is not the matching ready audit: $($implementationInfo.Path) ($planAcceptanceAudit)")
        }
        if ((Get-GitRevision $implementationInfo.PlanEvidenceRevision) -ne (Get-GitRevision $acceptanceInfo.EvidenceRevision)) {
            $failures.Add("Implementation plan_evidence_revision must match its plan acceptance evidence_revision: $($implementationInfo.Path) ($planAcceptanceAudit)")
        }
        if ($implementationInfo.GovernanceContract -eq 'audit-loop/v3') {
            $implementationBaselineSha = Get-GitRevision $implementationInfo.Baseline
            if (-not (Test-GitRecordMatchesSnapshot $implementationBaselineSha $acceptanceInfo)) {
                $failures.Add("Implementation baseline must contain its ready plan acceptance terminal blob: $($implementationInfo.Path) ($planAcceptanceAudit)")
            }
            $acceptancePlanAudits = @($acceptanceInfo.RelatedAudits | Where-Object {
                $auditMetadata.ContainsKey($_) -and $auditMetadata[$_].Schema -eq 'plan-audit/v2'
            })
            foreach ($acceptancePlanAuditId in $acceptancePlanAudits) {
                $acceptancePlanAudit = $auditMetadata[$acceptancePlanAuditId]
                if ($useGitLedgerChecks) {
                    $implementationPeers = @(Get-ActivePlanIdsAtRevision $implementationBaselineSha)
                    $auditedPeers = @($acceptancePlanAudit.AuditedPeerPlans | Sort-Object -Unique)
                    if (($implementationPeers -join ',') -ne ($auditedPeers -join ',')) {
                        $failures.Add("Implementation readiness peer set drifted after acceptance: $($implementationInfo.Path) ($planAcceptanceAudit; expected=$($implementationPeers -join ','); actual=$($auditedPeers -join ','))")
                    }
                }
            }
        }
        if ($docsRoot -eq (Join-Path $repoRoot 'docs') -and $planMetadata.ContainsKey($implementationInfo.PlanId)) {
            $planEvidenceSha = Get-GitRevision $implementationInfo.PlanEvidenceRevision
            $implementationBaselineSha = Get-GitRevision $implementationInfo.Baseline
            if ($null -ne $planEvidenceSha -and $null -ne $implementationBaselineSha -and
                (Test-GitCommitExists $planEvidenceSha) -and (Test-GitCommitExists $implementationBaselineSha)) {
                $planInfo = $planMetadata[$implementationInfo.PlanId]
                $planPath = "docs/plans/$($planInfo.Stem).md"
                $checklistPath = "docs/plans/$($planInfo.Stem)-checklist.md"
                & $gitExecutable -C $repoRoot diff --quiet $planEvidenceSha $implementationBaselineSha -- $planPath $checklistPath
                if ($LASTEXITCODE -eq 1) {
                    $failures.Add("Implementation plan/checklist drifted after readiness acceptance: $($implementationInfo.Path) ($planAcceptanceAudit)")
                } elseif ($LASTEXITCODE -ne 0) {
                    $failures.Add("Implementation plan drift comparison failed: $($implementationInfo.Path) ($planAcceptanceAudit)")
                }
            }
        }
        if ($implementationInfo.GovernanceContract -eq 'audit-loop/v3') {
            $implementationBaselineSha = Get-GitRevision $implementationInfo.Baseline
            $eligiblePlanAcceptances = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.Schema -eq 'plan-acceptance/v2' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedPlans -contains $implementationInfo.PlanId -and
                (($useGitLedgerChecks -and (Test-GitMetadataEntryMatchesSnapshot $implementationBaselineSha $_)) -or
                 (-not $useGitLedgerChecks -and (Test-DisplayRecordAvailableBefore $_.Value $implementationInfo)))
            })
            $latestEligible = Select-LatestGitRecord $eligiblePlanAcceptances $implementationBaselineSha $failures "Implementation ready plan acceptance"
            if ($null -eq $latestEligible -or $latestEligible.Key -ne $planAcceptanceAudit) {
                $latestEligibleId = if ($null -ne $latestEligible) { $latestEligible.Key } else { 'none' }
                $failures.Add("Implementation must reference the latest plan acceptance in its baseline: $($implementationInfo.Path) (expected=$latestEligibleId, found=$planAcceptanceAudit)")
            }
        } else {
            $eligiblePlanAcceptances = @($auditMetadata.GetEnumerator() | Where-Object {
                $_.Value.Schema -eq 'plan-acceptance/v2' -and
                $_.Value.Status -eq 'closed' -and
                $_.Value.RelatedPlans -contains $implementationInfo.PlanId -and
                (Get-AuditNumber $_.Key) -lt (Get-ImplementationNumber ([regex]::Match($implementationInfo.Path, '(IMP-\d{4})').Groups[1].Value))
            } | Sort-Object { Get-AuditNumber $_.Key })
            if ($eligiblePlanAcceptances.Count -eq 0 -or $eligiblePlanAcceptances[-1].Key -ne $planAcceptanceAudit) {
                $latestEligible = if ($eligiblePlanAcceptances.Count -gt 0) { $eligiblePlanAcceptances[-1].Key } else { 'none' }
                $failures.Add("Implementation must reference the latest plan acceptance available when it started: $($implementationInfo.Path) (expected=$latestEligible, found=$planAcceptanceAudit)")
            }
        }
    }
}

foreach ($remediationInfo in $remediationMetadata.Values) {
    foreach ($sourceAudit in @($remediationInfo.SourceAudits)) {
        if (-not $auditMetadata.ContainsKey($sourceAudit)) {
            $failures.Add("Remediation references a missing source audit: $($remediationInfo.Path) ($sourceAudit)")
        }
    }
    foreach ($relatedImplementation in @($remediationInfo.RelatedImplementations)) {
        if (-not $implementationMetadata.ContainsKey($relatedImplementation)) {
            $failures.Add("Remediation references a missing implementation: $($remediationInfo.Path) ($relatedImplementation)")
        }
    }
}

$phaseFivePlanPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle.md'
$phaseFiveChecklistPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle-checklist.md'
if ((Test-Path -LiteralPath $phaseFivePlanPath) -and (Test-Path -LiteralPath $phaseFiveChecklistPath)) {
    $phaseFivePlan = Get-Content -Raw -Encoding UTF8 $phaseFivePlanPath
    $phaseFiveChecklist = Get-Content -Raw -Encoding UTF8 $phaseFiveChecklistPath
    $phaseFiveDag = Get-PhaseFiveDag $phaseFivePlan $failures
    Test-PhaseFiveFactsOutput $phaseFivePlan $failures
    Test-PhaseFiveDependencyStatements $phaseFivePlan $phaseFiveDag $failures
    Test-PhaseFiveDependencyStatements $phaseFiveChecklist $phaseFiveDag $failures

    $deploymentContract = [regex]::Match($phaseFivePlan, '(?s)<!--\s*phase5-p0-deployment-evidence-contract\s*\r?\n(?<body>.*?)\r?\n-->')
    $forbiddenP0LiveEvidence = @()
    if ($deploymentContract.Success) {
        $forbiddenMatch = [regex]::Match($deploymentContract.Groups['body'].Value, '(?m)^forbidden_p0_live_evidence:\s*(?<value>[^\r\n]+?)\s*$')
        if ($forbiddenMatch.Success) {
            $forbiddenP0LiveEvidence = @($forbiddenMatch.Groups['value'].Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { $_ })
        }
    }
    if (-not $deploymentContract.Success -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^p0_artifact_kinds:\s*contract-fixture,disposable-spike\s*$' -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^live_evidence_gates:\s*5A-D-2,5B-4\s*$' -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^forbidden_p0_live_evidence:\s*release-binary,supervisor-run,cleanup-schedule-run,watchdog-recovery-run,enospc-run,live-profile-run\s*$') {
        $failures.Add('PLN-0005 must define the structured P0 deployment evidence contract')
    }

    $p0Items = [regex]::Matches($phaseFiveChecklist, '(?m)^- \[[ xX]\] P0-(?<number>\d+)\..*$')
    $p0Counts = @{}
    foreach ($item in $p0Items) {
        $number = [int]$item.Groups['number'].Value
        if (-not $p0Counts.ContainsKey($number)) {
            $p0Counts[$number] = 0
        }
        $p0Counts[$number]++
    }
    foreach ($number in 1..25) {
        if (-not $p0Counts.ContainsKey($number)) {
            $failures.Add("PLN-0005 checklist is missing P0-${number}")
        } elseif ($p0Counts[$number] -ne 1) {
            $failures.Add("PLN-0005 checklist contains duplicate P0-${number}")
        }
    }
    foreach ($number in $p0Counts.Keys) {
        if ($number -lt 1 -or $number -gt 25) {
            $failures.Add("PLN-0005 checklist contains an unexpected P0 item: P0-${number}")
        }
    }

    Test-PhaseFiveDeploymentClauses $phaseFiveChecklist $forbiddenP0LiveEvidence $true $failures
    Test-PhaseFiveDeploymentClauses $phaseFivePlan $forbiddenP0LiveEvidence $false $failures

    $p021Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-21\..*$')
    if (-not $p021Line.Success -or $p021Line.Value -notmatch 'WP-Baseline-Evidence.*WP-Facts') {
        $failures.Add('PLN-0005 P0-21 must reject a missing WP-Baseline-Evidence to WP-Facts edge')
    }
    $p022Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-22\..*$')
    if (-not $p022Line.Success -or
        $p022Line.Value -notmatch 'artifactKind=contract-fixture' -or
        $p022Line.Value -notmatch '5A-D-2' -or
        $p022Line.Value -notmatch '5B-4') {
        $failures.Add('PLN-0005 P0-22 must stop at a deployment contract fixture and defer live profile evidence')
    }
    $p023Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-23\..*$')
    if (-not $p023Line.Success -or
        $p023Line.Value -notmatch 'P0-22.*artifactKind=contract-fixture' -or
        $p023Line.Value -notmatch '5A-D' -or
        $p023Line.Value -notmatch '5B') {
        $failures.Add('PLN-0005 P0-23 must keep P0-22 evidence as a contract fixture')
    }
}

$auditIndexPath = Join-Path $docsRoot 'audits\README.md'
if ($auditRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $auditIndexPath)) {
        $failures.Add('Audit records exist but audits/README.md is missing')
    } else {
        $auditIndexContent = Get-Content -Raw -Encoding UTF8 $auditIndexPath
        foreach ($entry in $auditRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $auditIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Audit record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            $auditIdForEntry = [regex]::Match($entry.Key, '^(AUD-\d{4})-').Groups[1].Value
            $remediationStateMatch = [regex]::Match($indexLine, 'remediation=(?<state>pending|required|implementation-required|audit-required|decision-required|none|accepted-risk|awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4}|implemented-by:IMP-\d{4}|audited-by:AUD-\d{4})')
            if ($remediationStateMatch.Success) {
                $auditRemediationStates[$auditIdForEntry] = $remediationStateMatch.Groups['state'].Value
            }
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'remediation=(pending|required|implementation-required|audit-required|decision-required|none|accepted-risk|awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4}|implemented-by:IMP-\d{4}|audited-by:AUD-\d{4})') {
                $failures.Add("Audit index status is missing or inconsistent: $($entry.Key)")
            } else {
                $auditInfo = $auditMetadata[$auditIdForEntry]
                if ($null -ne $auditInfo -and $auditInfo.Schema -in @('plan-acceptance/v2', 'implementation-acceptance/v2')) {
                    $state = $remediationStateMatch.Groups['state'].Value
                    $expectedRouting = 'unknown'
                    $validRemediationState = if ($auditInfo.Schema -eq 'implementation-acceptance/v2') {
                        switch ($auditInfo.NextAction) {
                            'pending' { $expectedRouting = 'pending'; $state -eq 'pending' }
                            'none' { $expectedRouting = 'none'; $state -eq 'none' }
                            'implement' { $expectedRouting = 'implementation-required or implemented-by:IMP-NNNN'; $state -eq 'implementation-required' -or $state -match '^implemented-by:IMP-\d{4}$' }
                            'implementation-audit' { $expectedRouting = 'audit-required or audited-by:AUD-NNNN'; $state -eq 'audit-required' -or $state -match '^audited-by:AUD-\d{4}$' }
                            'remediate' { $expectedRouting = 'required or its REM/follow-up transition'; $state -eq 'required' -or $state -match '^(?:awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4})$' }
                            'decision' { $expectedRouting = 'decision-required'; $state -eq 'decision-required' }
                            'superseded' { $expectedRouting = 'none'; $state -eq 'none' }
                            default { $false }
                        }
                    } else {
                        $expectedRemediation = if ($auditInfo.Verdict -eq 'pending') { 'pending' } elseif ($auditInfo.Verdict -in @('ready', 'superseded')) { 'none' } elseif ($auditInfo.Verdict -eq 'blocked') { 'decision-required' } else { 'required' }
                        $expectedRouting = $expectedRemediation
                        $state -eq $expectedRemediation
                    }
                    if (-not $validRemediationState) {
                        $failures.Add("Acceptance audit index remediation does not match verdict/next action: $($entry.Key) ($($auditInfo.Verdict)/$($auditInfo.NextAction)/$state; expected $expectedRouting)")
                    }
                }
                if ($null -ne $auditInfo -and $auditInfo.GovernanceContract -eq 'audit-loop/v3') {
                    $state = $remediationStateMatch.Groups['state'].Value
                    if ($auditInfo.Status -eq 'superseded' -and $state -ne 'none') {
                        $failures.Add("Superseded audit must use remediation=none: $($entry.Key)")
                    }
                    $hasOpenDisposition = $auditInfo.Content -match '(?m)^-\s*Disposition:\s*(?:open|partially-resolved)\s*$'
                    if ($state -in @('required', 'implementation-required', 'audit-required') -and -not $hasOpenDisposition) {
                        $failures.Add("Actionable audit state must have an open finding disposition: $($entry.Key)")
                    }
                    if ($auditInfo.Status -ne 'superseded' -and $state -in @('none', 'accepted-risk') -and $hasOpenDisposition) {
                        $failures.Add("Clean audit-loop/v3 remediation state cannot retain open findings: $($entry.Key)")
                    }
                    if ($state -eq 'accepted-risk' -and $auditInfo.Content -notmatch '(?m)^-\s*Disposition:\s*accepted-risk\s*$') {
                        $failures.Add("audit-loop/v3 remediation=accepted-risk requires an accepted-risk finding: $($entry.Key)")
                    }
                    if ($state -eq 'accepted-risk') {
                        $failures.Add("audit-loop/v3 remediation=accepted-risk requires an externally verifiable approval attestation; repository-authored Owner/Disposition text is not authorization: $($entry.Key)")
                    }
                }
            }
        }
    }
}

$remediationIndexPath = Join-Path $docsRoot 'remediations\README.md'
if ($remediationRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $remediationIndexPath)) {
        $failures.Add('Remediation records exist but remediations/README.md is missing')
    } else {
        $remediationIndexContent = Get-Content -Raw -Encoding UTF8 $remediationIndexPath
        foreach ($entry in $remediationRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $remediationIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Remediation record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            $remediationIdForEntry = [regex]::Match($entry.Key, '^(REM-\d{4})-').Groups[1].Value
            $verificationStateMatch = [regex]::Match($indexLine, 'verification=(?<state>not-ready|pending|verified-by:AUD-\d{4}|partial-by:AUD-\d{4}|failed-by:AUD-\d{4})')
            if ($verificationStateMatch.Success) {
                $remediationVerificationStates[$remediationIdForEntry] = $verificationStateMatch.Groups['state'].Value
            }
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'verification=(not-ready|pending|verified-by:AUD-\d{4}|partial-by:AUD-\d{4}|failed-by:AUD-\d{4})') {
                $failures.Add("Remediation index status is missing or inconsistent: $($entry.Key)")
            }
        }
    }
}

$implementationIndexPath = Join-Path $docsRoot 'implementations\README.md'
if ($implementationRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $implementationIndexPath)) {
        $failures.Add('Implementation records exist but implementations/README.md is missing')
    } else {
        $implementationIndexContent = Get-Content -Raw -Encoding UTF8 $implementationIndexPath
        foreach ($entry in $implementationRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $implementationIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Implementation record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'audit=(not-ready|pending|audited-by:AUD-\d{4})' -or
                $indexLine -notmatch 'acceptance=(not-ready|pending|accepted-by:AUD-\d{4}|rejected-by:AUD-\d{4})') {
                $failures.Add("Implementation index status is missing or inconsistent: $($entry.Key)")
            } else {
                $implementationIdForEntry = [regex]::Match($entry.Key, '^(IMP-\d{4})-').Groups[1].Value
                $auditMatch = [regex]::Match($indexLine, 'audit=audited-by:(?<audit>AUD-\d{4})')
                if ($auditMatch.Success) {
                    $implementationAuditId = $auditMatch.Groups['audit'].Value
                    $implementationInfoForEntry = $implementationMetadata[$implementationIdForEntry]
                    $expectedImplementationAuditSchema = if ($null -ne $implementationInfoForEntry -and $implementationInfoForEntry.GovernanceContract -eq 'audit-loop/v3') { 'implementation-audit/v2' } else { 'implementation-audit/v1' }
                    if (-not $auditMetadata.ContainsKey($implementationAuditId) -or
                        $auditMetadata[$implementationAuditId].Schema -ne $expectedImplementationAuditSchema -or
                        $auditMetadata[$implementationAuditId].RelatedImplementations -notcontains $implementationIdForEntry) {
                        $failures.Add("Implementation audit index references a non-matching audit: $($entry.Key) ($implementationAuditId)")
                    }
                    $matchingImplementationAudits = @($auditMetadata.GetEnumerator() | Where-Object {
                        $_.Value.Schema -eq $expectedImplementationAuditSchema -and
                        $_.Value.RelatedImplementations -contains $implementationIdForEntry
                    })
                    if ($matchingImplementationAudits.Count -gt 0) {
                        $latestImplementationAudit = if ($null -ne $implementationInfoForEntry -and $implementationInfoForEntry.GovernanceContract -eq 'audit-loop/v3') {
                            Select-LatestGitRecord $matchingImplementationAudits 'HEAD' $failures "Implementation index audit pointer"
                        } else { $matchingImplementationAudits | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                        if ($null -ne $latestImplementationAudit -and $latestImplementationAudit.Key -ne $implementationAuditId) {
                            $failures.Add("Implementation index must reference the latest implementation audit: $($entry.Key) (expected=$($latestImplementationAudit.Key))")
                        }
                    }
                }
                $acceptanceMatch = [regex]::Match($indexLine, 'acceptance=(?<state>accepted-by|rejected-by):(?<audit>AUD-\d{4})')
                if ($acceptanceMatch.Success) {
                    if ($acceptanceMatch.Groups['state'].Value -eq 'accepted-by' -and -not $auditMatch.Success) {
                        $failures.Add("Accepted implementation must reference a completed implementation audit: $($entry.Key)")
                    }
                    $acceptanceAuditId = $acceptanceMatch.Groups['audit'].Value
                    if (-not $auditMetadata.ContainsKey($acceptanceAuditId) -or
                        $auditMetadata[$acceptanceAuditId].Schema -ne 'implementation-acceptance/v2' -or
                        $auditMetadata[$acceptanceAuditId].RelatedImplementations -notcontains $implementationIdForEntry) {
                        $failures.Add("Implementation acceptance index references a non-matching audit: $($entry.Key) ($acceptanceAuditId)")
                    } else {
                        $expectedState = if ($auditMetadata[$acceptanceAuditId].Verdict -eq 'complete') { 'accepted-by' } else { 'rejected-by' }
                        if ($acceptanceMatch.Groups['state'].Value -ne $expectedState) {
                            $failures.Add("Implementation acceptance index state does not match audit verdict: $($entry.Key)")
                        }
                        $matchingImplementationAcceptances = @($auditMetadata.GetEnumerator() | Where-Object {
                            $_.Value.Schema -eq 'implementation-acceptance/v2' -and
                            $_.Value.RelatedImplementations -contains $implementationIdForEntry
                        })
                        if ($matchingImplementationAcceptances.Count -gt 0) {
                            $latestImplementationAcceptance = if ($null -ne $implementationInfoForEntry -and $implementationInfoForEntry.GovernanceContract -eq 'audit-loop/v3') {
                                Select-LatestGitRecord $matchingImplementationAcceptances 'HEAD' $failures "Implementation index acceptance pointer"
                            } else { $matchingImplementationAcceptances | Sort-Object { Get-AuditNumber $_.Key } | Select-Object -Last 1 }
                            if ($null -ne $latestImplementationAcceptance -and $latestImplementationAcceptance.Key -ne $acceptanceAuditId) {
                                $failures.Add("Implementation index must reference the latest completion acceptance: $($entry.Key) (expected=$($latestImplementationAcceptance.Key))")
                            }
                        }
                    }
                }
            }
        }
    }
}

foreach ($auditStateEntry in $auditRemediationStates.GetEnumerator()) {
    $sourceAuditId = $auditStateEntry.Key
    $state = $auditStateEntry.Value
    if ($state -match '^implemented-by:(?<implementation>IMP-\d{4})$') {
        $targetImplementationId = $Matches['implementation']
        if (-not $auditMetadata.ContainsKey($sourceAuditId) -or
            $auditMetadata[$sourceAuditId].Schema -ne 'implementation-acceptance/v2' -or
            $auditMetadata[$sourceAuditId].NextAction -ne 'implement') {
            $failures.Add("implemented-by transition requires an implementation acceptance with next action implement: $sourceAuditId")
            continue
        }
        if (-not $implementationMetadata.ContainsKey($targetImplementationId)) {
            $failures.Add("implemented-by transition references a missing IMP: $sourceAuditId ($targetImplementationId)")
            continue
        }
        $targetImplementation = $implementationMetadata[$targetImplementationId]
        if ($targetImplementation.TriggerAudits -notcontains $sourceAuditId -or
            $auditMetadata[$sourceAuditId].RelatedPlans -notcontains $targetImplementation.PlanId) {
            $failures.Add("implemented-by transition must be bidirectionally linked to a matching IMP: $sourceAuditId ($targetImplementationId)")
        }
        if ($targetImplementation.GovernanceContract -eq 'audit-loop/v3' -and
            -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $targetImplementation.Baseline) $auditMetadata[$sourceAuditId])) {
            $failures.Add("implemented-by IMP baseline must contain the triggering acceptance terminal blob: $sourceAuditId ($targetImplementationId)")
        }
        continue
    }
    if ($state -match '^audited-by:(?<audit>AUD-\d{4})$') {
        $targetAuditId = $Matches['audit']
        if (-not $auditMetadata.ContainsKey($sourceAuditId) -or
            $auditMetadata[$sourceAuditId].Schema -ne 'implementation-acceptance/v2' -or
            $auditMetadata[$sourceAuditId].NextAction -ne 'implementation-audit') {
            $failures.Add("audited-by transition requires an implementation acceptance with next action implementation-audit: $sourceAuditId")
            continue
        }
        if (-not $auditMetadata.ContainsKey($targetAuditId)) {
            $failures.Add("audited-by transition references a missing implementation audit: $sourceAuditId ($targetAuditId)")
            continue
        }
        $targetAudit = $auditMetadata[$targetAuditId]
        if ($targetAudit.Schema -ne 'implementation-audit/v2' -or $targetAudit.Status -ne 'closed' -or
            $targetAudit.RelatedAudits -notcontains $sourceAuditId -or
            @($targetAudit.RelatedPlans | Where-Object { $auditMetadata[$sourceAuditId].RelatedPlans -contains $_ }).Count -eq 0) {
            $failures.Add("audited-by transition must target a matching closed implementation audit: $sourceAuditId ($targetAuditId)")
        }
        continue
    }
    if ($state -match '^awaiting-verification:(?<remediation>REM-\d{4})$') {
        $targetRemediation = $Matches['remediation']
        if (-not $remediationMetadata.ContainsKey($targetRemediation)) {
            $failures.Add("Audit remediation state references a missing remediation: $sourceAuditId ($targetRemediation)")
            continue
        }
        if ($remediationMetadata[$targetRemediation].SourceAudits -notcontains $sourceAuditId) {
            $failures.Add("Audit awaiting-verification target does not reference its source audit: $sourceAuditId ($targetRemediation)")
        }
        if (-not $remediationVerificationStates.ContainsKey($targetRemediation) -or
            $remediationVerificationStates[$targetRemediation] -ne 'pending') {
            $failures.Add("Audit awaiting-verification target must be pending verification: $sourceAuditId ($targetRemediation)")
        }
        continue
    }
    if ($state -match '^(?<transition>verified-by|continued-by):(?<audit>AUD-\d{4})$') {
        $targetAuditId = $Matches['audit']
        if (-not $auditMetadata.ContainsKey($targetAuditId)) {
            $failures.Add("Audit remediation state references a missing follow-up audit: $sourceAuditId ($targetAuditId)")
            continue
        }
        $targetAudit = $auditMetadata[$targetAuditId]
        if ($targetAudit.AuditType -ne 'follow-up' -or $targetAudit.Status -ne 'closed') {
            $failures.Add("Audit remediation transition must target a closed follow-up audit: $sourceAuditId ($targetAuditId)")
        }
        if ($targetAudit.RelatedAudits -notcontains $sourceAuditId) {
            $failures.Add("Audit remediation transition target does not reference its source audit: $sourceAuditId ($targetAuditId)")
        }
        if ($targetAudit.GovernanceContract -eq 'audit-loop/v3' -and
            -not (Test-GitRecordDescendsFrom $targetAudit $auditMetadata[$sourceAuditId])) {
            $failures.Add("Audit remediation transition baseline must contain its source terminal audit blob: $sourceAuditId ($targetAuditId)")
        } elseif ($targetAudit.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $targetAuditId) -le (Get-AuditNumber $sourceAuditId)) {
            $failures.Add("Legacy audit remediation transition must point to a newer audit: $sourceAuditId ($targetAuditId)")
        }
    }
}

foreach ($implementationEntry in $implementationMetadata.GetEnumerator()) {
    foreach ($triggerAudit in @($implementationEntry.Value.TriggerAudits)) {
        $expectedState = "implemented-by:$($implementationEntry.Key)"
        if (-not $auditRemediationStates.ContainsKey($triggerAudit) -or $auditRemediationStates[$triggerAudit] -ne $expectedState) {
            $failures.Add("IMP trigger_audits must use the matching implemented-by index transition: $($implementationEntry.Value.Path) ($triggerAudit)")
        }
    }
}

foreach ($implementationAuditEntry in $auditMetadata.GetEnumerator()) {
    if ($implementationAuditEntry.Value.Schema -ne 'implementation-audit/v2' -or $implementationAuditEntry.Value.Status -ne 'closed') { continue }
    foreach ($sourceAudit in @($implementationAuditEntry.Value.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($sourceAudit) -or
            $auditMetadata[$sourceAudit].Schema -ne 'implementation-acceptance/v2' -or
            $auditMetadata[$sourceAudit].NextAction -ne 'implementation-audit') {
            continue
        }
        $expectedState = "audited-by:$($implementationAuditEntry.Key)"
        if (-not $auditRemediationStates.ContainsKey($sourceAudit) -or $auditRemediationStates[$sourceAudit] -ne $expectedState) {
            $failures.Add("Implementation audit trigger must use the matching audited-by index transition: $($implementationAuditEntry.Value.Path) ($sourceAudit)")
        }
    }
}

foreach ($remediationStateEntry in $remediationVerificationStates.GetEnumerator()) {
    $remediationId = $remediationStateEntry.Key
    $state = $remediationStateEntry.Value
    if (-not $remediationMetadata.ContainsKey($remediationId)) { continue }
    $remediationInfo = $remediationMetadata[$remediationId]
    if ($state -eq 'pending') {
        if ($remediationInfo.Status -notin @('completed', 'partial')) {
            $failures.Add("Pending remediation verification requires completed or partial status: $remediationId")
        }
        foreach ($sourceAudit in @($remediationInfo.SourceAudits)) {
            $allowedSourceStates = @("awaiting-verification:$remediationId")
            if (-not $auditRemediationStates.ContainsKey($sourceAudit) -or
                $auditRemediationStates[$sourceAudit] -notin $allowedSourceStates) {
                $failures.Add("Pending remediation must remain attached to its source audit queue: $remediationId ($sourceAudit)")
            }
        }
        continue
    }
    if ($state -match '^(?<verdict>verified-by|partial-by|failed-by):(?<audit>AUD-\d{4})$') {
        $targetAuditId = $Matches['audit']
        if (-not $auditMetadata.ContainsKey($targetAuditId)) {
            $failures.Add("Remediation verification state references a missing follow-up audit: $remediationId ($targetAuditId)")
            continue
        }
        $targetAudit = $auditMetadata[$targetAuditId]
        if ($targetAudit.AuditType -ne 'follow-up' -or $targetAudit.Status -ne 'closed' -or
            $targetAudit.RelatedRemediations -notcontains $remediationId) {
            $failures.Add("Remediation verification transition must target its matching closed follow-up audit: $remediationId ($targetAuditId)")
        }
        if ($remediationInfo.Schema -eq 'remediation/v2' -and
            (Get-GitRevision $targetAudit.EvidenceRevision) -ne (Get-GitRevision $remediationInfo.ResultRevision)) {
            $failures.Add("Follow-up audit evidence_revision must match remediation/v2 result_revision: $remediationId ($targetAuditId)")
        }
        if ($remediationInfo.AffectsImplementation) {
            foreach ($relatedImplementation in @($remediationInfo.RelatedImplementations)) {
                if ($targetAudit.RelatedImplementations -notcontains $relatedImplementation) {
                    $failures.Add("Implementation remediation follow-up must preserve related_implementations: $remediationId ($targetAuditId/$relatedImplementation)")
                }
            }
        }
        foreach ($sourceAudit in @($remediationInfo.SourceAudits)) {
            if (-not $auditRemediationStates.ContainsKey($sourceAudit) -or
                $auditRemediationStates[$sourceAudit] -notin @("verified-by:$targetAuditId", "continued-by:$targetAuditId")) {
                $failures.Add("Verified remediation source audit must transition through the same follow-up: $remediationId ($sourceAudit/$targetAuditId)")
            }
        }
    }
}

foreach ($auditEntry in $auditMetadata.GetEnumerator()) {
    $auditInfo = $auditEntry.Value
    if ($auditInfo.Schema -ne 'implementation-acceptance/v2' -or
        $auditInfo.RelatedImplementations.Count -ne 1) {
        continue
    }
    $relatedImplementation = $auditInfo.RelatedImplementations[0]
    if (-not $implementationMetadata.ContainsKey($relatedImplementation)) { continue }
    if ($implementationMetadata[$relatedImplementation].Status -ne 'completed') {
        if ($auditInfo.EffectiveResultRevision -ne 'none') {
            $failures.Add("Implementation acceptance for a non-completed IMP must use effective_result_revision=none: $($auditInfo.Path) ($relatedImplementation)")
        }
        continue
    }
    if ($auditInfo.EffectiveResultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
        $failures.Add("Implementation acceptance for a completed IMP must record a full effective_result_revision: $($auditInfo.Path) ($relatedImplementation)")
        continue
    }
    $currentAcceptanceAuditId = $auditEntry.Key
    $verifiedImplementationRemediations = @($remediationMetadata.GetEnumerator() | Where-Object {
        $verificationState = if ($remediationVerificationStates.ContainsKey($_.Key)) { $remediationVerificationStates[$_.Key] } else { $null }
        $verificationAuditId = if ($verificationState -match '^verified-by:(AUD-\d{4})$') { $Matches[1] } else { $null }
        $_.Value.Schema -eq 'remediation/v2' -and
        $_.Value.AffectsImplementation -and
        $_.Value.RelatedImplementations -contains $relatedImplementation -and
        $null -ne $verificationAuditId -and
        (($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
          $auditMetadata.ContainsKey($verificationAuditId) -and
          (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$verificationAuditId]) -and
          (Test-GitMetadataEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $_)) -or
         (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
          $auditMetadata.ContainsKey($verificationAuditId) -and
          (Test-DisplayRecordAvailableBefore $auditMetadata[$verificationAuditId] $auditInfo)) -or
         ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and
          (Get-AuditNumber $verificationAuditId) -lt (Get-AuditNumber $currentAcceptanceAuditId)))
    } | Sort-Object { Get-RemediationNumber $_.Key })
    foreach ($verifiedRemediation in $verifiedImplementationRemediations) {
        if ($auditInfo.RelatedRemediations -notcontains $verifiedRemediation.Key) {
            $failures.Add("Implementation acceptance must include every verified implementation remediation: $($auditInfo.Path) ($($verifiedRemediation.Key))")
        }
        $verificationState = $remediationVerificationStates[$verifiedRemediation.Key]
        $verificationAuditId = [regex]::Match($verificationState, 'AUD-\d{4}').Value
        if ($auditInfo.RelatedAudits -notcontains $verificationAuditId) {
            $failures.Add("Implementation acceptance must include each effective remediation follow-up audit: $($auditInfo.Path) ($verificationAuditId)")
        }
    }
    $expectedEffectiveRevision = $implementationMetadata[$relatedImplementation].ResultRevision
    if ($auditInfo.GovernanceContract -eq 'audit-loop/v3' -and $verifiedImplementationRemediations.Count -gt 0) {
        $legacyEffectiveRemediations = @($verifiedImplementationRemediations | Where-Object { $_.Value.GovernanceContract -ne 'audit-loop/v3' })
        if ($legacyEffectiveRemediations.Count -gt 0) {
            $failures.Add("audit-loop/v3 completion acceptance cannot infer a linear chain from legacy implementation remediations: $($auditInfo.Path)")
        } else {
            $remainingRemediations = New-Object System.Collections.ArrayList
            foreach ($verifiedRemediation in $verifiedImplementationRemediations) { [void]$remainingRemediations.Add($verifiedRemediation) }
            while ($remainingRemediations.Count -gt 0) {
                $currentRevisionSha = Get-GitRevision $expectedEffectiveRevision
                $candidates = @($remainingRemediations | Where-Object { (Get-GitRevision $_.Value.ParentResultRevision) -eq $currentRevisionSha })
                if ($candidates.Count -eq 0) {
                    $failures.Add("Implementation remediation chain is disconnected from the current effective revision: $($auditInfo.Path)")
                    break
                }
                if ($candidates.Count -gt 1) {
                    $failures.Add("Implementation remediation chain branches at ${expectedEffectiveRevision}: $($auditInfo.Path)")
                    break
                }
                $nextRemediation = $candidates[0]
                $expectedEffectiveRevision = $nextRemediation.Value.ResultRevision
                [void]$remainingRemediations.Remove($nextRemediation)
            }
        }
    } elseif ($verifiedImplementationRemediations.Count -gt 0) {
        $expectedEffectiveRevision = $verifiedImplementationRemediations[-1].Value.ResultRevision
    }
    if ((Get-GitRevision $auditInfo.EffectiveResultRevision) -ne (Get-GitRevision $expectedEffectiveRevision)) {
        $failures.Add("Implementation acceptance effective_result_revision does not match the IMP/REM chain: $($auditInfo.Path) (expected=$expectedEffectiveRevision)")
    }
}

foreach ($auditEntry in $auditMetadata.GetEnumerator()) {
    $auditInfo = $auditEntry.Value
    if ($auditInfo.Schema -notin @('plan-acceptance/v2', 'implementation-acceptance/v2')) {
        continue
    }
    $currentAuditNumber = Get-AuditNumber $auditEntry.Key
    foreach ($remediationEntry in $remediationMetadata.GetEnumerator()) {
        if (-not $remediationVerificationStates.ContainsKey($remediationEntry.Key)) { continue }
        $verificationAuditId = [regex]::Match($remediationVerificationStates[$remediationEntry.Key], 'AUD-\d{4}').Value
        if ([string]::IsNullOrWhiteSpace($verificationAuditId) -or
            ($useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
             (-not $auditMetadata.ContainsKey($verificationAuditId) -or
              -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$verificationAuditId]) -or
              -not (Test-GitMetadataEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $remediationEntry))) -or
            (-not $useGitLedgerChecks -and $auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
             (-not $auditMetadata.ContainsKey($verificationAuditId) -or
              -not (Test-DisplayRecordAvailableBefore $auditMetadata[$verificationAuditId] $auditInfo))) -or
            ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and
             (Get-AuditNumber $verificationAuditId) -ge $currentAuditNumber)) {
            continue
        }
        $planRelated = @($remediationEntry.Value.RelatedPlans | Where-Object { $auditInfo.RelatedPlans -contains $_ }).Count -gt 0
        $implementationRelated = @($remediationEntry.Value.RelatedImplementations | Where-Object { $auditInfo.RelatedImplementations -contains $_ }).Count -gt 0
        if (-not $planRelated -and -not $implementationRelated) { continue }
        if ($auditInfo.Schema -eq 'plan-acceptance/v2') {
            $readinessRemediation = $false
            foreach ($sourceAuditId in @($remediationEntry.Value.SourceAudits)) {
                if (Test-PlanReadinessChainAudit $sourceAuditId (New-Object 'System.Collections.Generic.HashSet[string]')) {
                    $readinessRemediation = $true
                    break
                }
            }
            if (-not $readinessRemediation) { continue }
        }
        if ($auditInfo.RelatedRemediations -notcontains $remediationEntry.Key) {
            $failures.Add("Acceptance audit must include every prior related remediation: $($auditInfo.Path) ($($remediationEntry.Key))")
        }
        if ($auditInfo.IndependenceBasis -eq 'separate-auditor' -and $auditInfo.Auditor -eq $remediationEntry.Value.Implementer) {
            $failures.Add("Separate-auditor acceptance must differ from prior remediation implementers: $($auditInfo.Path) ($($remediationEntry.Key))")
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.Schema -notin @('plan-acceptance/v2', 'implementation-acceptance/v2') -or
        $auditInfo.Verdict -notin @('ready', 'complete')) {
        continue
    }
    foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($relatedAudit)) { continue }
        if ($auditMetadata[$relatedAudit].Status -ne 'closed') {
            $failures.Add("Successful acceptance cannot reference an open audit: $($auditInfo.Path) ($relatedAudit)")
        }
        if (-not $auditRemediationStates.ContainsKey($relatedAudit)) {
            $failures.Add("Successful acceptance cannot verify an unindexed audit chain: $($auditInfo.Path) ($relatedAudit)")
            continue
        }
        $relatedRemediationState = $auditRemediationStates[$relatedAudit]
        if ($relatedRemediationState -in @('pending', 'required', 'implementation-required', 'audit-required', 'decision-required') -or $relatedRemediationState -like 'awaiting-verification:*') {
            $failures.Add("Successful acceptance requires a clean related audit chain: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
        }
        if ($relatedRemediationState -match '^(?:verified-by|continued-by):(?<audit>AUD-\d{4})$') {
            $acceptanceAuditId = [regex]::Match($auditInfo.Path, '(AUD-\d{4})').Groups[1].Value
            $transitionAuditId = $Matches['audit']
            if ($auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
                $auditMetadata.ContainsKey($transitionAuditId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$transitionAuditId])) {
                $failures.Add("Successful acceptance baseline must contain its remediation transition audit blob: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
            } elseif ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $transitionAuditId) -ge (Get-AuditNumber $acceptanceAuditId)) {
                $failures.Add("Successful acceptance cannot rely on a later audit transition: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
            }
        }
        if ($relatedRemediationState -match '^implemented-by:(?<implementation>IMP-\d{4})$') {
            $transitionImplementationId = $Matches['implementation']
            if ($implementationMetadata.ContainsKey($transitionImplementationId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $implementationMetadata[$transitionImplementationId])) {
                $failures.Add("Successful acceptance baseline must contain its implementation transition blob: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
            }
        }
        if ($relatedRemediationState -match '^audited-by:(?<audit>AUD-\d{4})$') {
            $transitionAuditId = $Matches['audit']
            if ($auditMetadata.ContainsKey($transitionAuditId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$transitionAuditId])) {
                $failures.Add("Successful acceptance baseline must contain its implementation audit transition blob: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
            }
        }
    }
}

foreach ($auditEntry in $auditMetadata.GetEnumerator()) {
    $auditId = $auditEntry.Key
    $auditInfo = $auditEntry.Value
    if ($auditInfo.Schema -notin @('plan-acceptance/v2', 'implementation-acceptance/v2') -or
        $auditInfo.Verdict -notin @('ready', 'complete')) {
        continue
    }
    $currentAuditNumber = Get-AuditNumber $auditId
    foreach ($candidateEntry in $auditMetadata.GetEnumerator()) {
        if ($candidateEntry.Key -eq $auditId) { continue }
        if ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $candidateEntry.Key) -ge $currentAuditNumber) {
            continue
        }
        $candidate = $candidateEntry.Value
        if ($candidate.Status -eq 'superseded') { continue }
        $planRelated = @($candidate.RelatedPlans | Where-Object { $auditInfo.RelatedPlans -contains $_ }).Count -gt 0
        $implementationRelated = @($candidate.RelatedImplementations | Where-Object { $auditInfo.RelatedImplementations -contains $_ }).Count -gt 0
        if (-not $planRelated -and -not $implementationRelated) {
            continue
        }
        if ($auditInfo.Schema -eq 'plan-acceptance/v2' -and
            -not (Test-PlanReadinessChainAudit $candidateEntry.Key (New-Object 'System.Collections.Generic.HashSet[string]'))) {
            continue
        }
        if ($auditInfo.GovernanceContract -eq 'audit-loop/v3') {
            if ($useGitLedgerChecks) {
                if (-not (Test-GitEntryMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $candidateEntry)) { continue }
            } elseif (-not (Test-DisplayRecordAvailableBefore $candidate $auditInfo)) {
                continue
            }
        }
        if (-not $auditRemediationStates.ContainsKey($candidateEntry.Key)) {
            $failures.Add("Successful acceptance cannot omit an unindexed related audit: $($auditInfo.Path) ($($candidateEntry.Key))")
            continue
        }
        $state = $auditRemediationStates[$candidateEntry.Key]
        if ($state -in @('pending', 'required', 'implementation-required', 'audit-required', 'decision-required') -or $state -like 'awaiting-verification:*') {
            $failures.Add("Successful acceptance cannot bypass a dirty derived audit chain: $($auditInfo.Path) ($($candidateEntry.Key)=$state)")
        }
        if ($state -match '^(?:verified-by|continued-by):(?<audit>AUD-\d{4})$') {
            $transitionAuditId = $Matches['audit']
            if ($auditInfo.GovernanceContract -eq 'audit-loop/v3' -and
                $auditMetadata.ContainsKey($transitionAuditId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$transitionAuditId])) {
                $failures.Add("Successful acceptance baseline does not contain the derived transition audit blob: $($auditInfo.Path) ($($candidateEntry.Key)=$state)")
            } elseif ($auditInfo.GovernanceContract -ne 'audit-loop/v3' -and (Get-AuditNumber $transitionAuditId) -ge $currentAuditNumber) {
                $failures.Add("Successful acceptance cannot derive cleanliness from a later audit: $($auditInfo.Path) ($($candidateEntry.Key)=$state)")
            }
        }
        if ($state -match '^implemented-by:(?<implementation>IMP-\d{4})$') {
            $transitionImplementationId = $Matches['implementation']
            if ($implementationMetadata.ContainsKey($transitionImplementationId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $implementationMetadata[$transitionImplementationId])) {
                $failures.Add("Successful acceptance baseline does not contain the derived implementation blob: $($auditInfo.Path) ($($candidateEntry.Key)=$state)")
            }
        }
        if ($state -match '^audited-by:(?<audit>AUD-\d{4})$') {
            $transitionAuditId = $Matches['audit']
            if ($auditMetadata.ContainsKey($transitionAuditId) -and
                -not (Test-GitRecordMatchesSnapshot (Get-GitRevision $auditInfo.Baseline) $auditMetadata[$transitionAuditId])) {
                $failures.Add("Successful acceptance baseline does not contain the derived implementation audit blob: $($auditInfo.Path) ($($candidateEntry.Key)=$state)")
            }
        }
    }
}

$repositoryDocsRoot = (Join-Path $repoRoot 'docs')
if ($docsRoot -eq $repositoryDocsRoot) {
    $requiredAuditEntrypoints = @(
        '.github/prompts/backend-plan-audit.prompt.md',
        '.github/prompts/backend-plan-acceptance-audit.prompt.md',
        '.github/prompts/backend-implement-plan.prompt.md',
        '.github/prompts/backend-implementation-audit.prompt.md',
        '.github/prompts/backend-implementation-acceptance-audit.prompt.md',
        '.github/prompts/backend-fix-audit-findings.prompt.md',
        '.github/prompts/backend-follow-up-audit.prompt.md',
        'docs/audits/templates/follow-up-audit-record.md',
        '.agents/skills/backend-plan-audit/SKILL.md',
        '.agents/skills/backend-plan-audit/agents/openai.yaml',
        '.agents/skills/backend-plan-acceptance-audit/SKILL.md',
        '.agents/skills/backend-plan-acceptance-audit/agents/openai.yaml',
        '.agents/skills/backend-implement-plan/SKILL.md',
        '.agents/skills/backend-implement-plan/agents/openai.yaml',
        '.agents/skills/backend-implementation-audit/SKILL.md',
        '.agents/skills/backend-implementation-audit/agents/openai.yaml',
        '.agents/skills/backend-implementation-acceptance-audit/SKILL.md',
        '.agents/skills/backend-implementation-acceptance-audit/agents/openai.yaml',
        '.agents/skills/backend-fix-audit-findings/SKILL.md',
        '.agents/skills/backend-fix-audit-findings/agents/openai.yaml',
        '.agents/skills/backend-follow-up-audit/SKILL.md',
        '.agents/skills/backend-follow-up-audit/agents/openai.yaml',
        '.agents/skills/backend-plan-audit-until-ready/SKILL.md',
        '.agents/skills/backend-plan-audit-until-ready/agents/openai.yaml',
        '.github/prompts/backend-plan-audit-until-ready.prompt.md',
        '.agents/skills/backend-implement-audit-until-complete/SKILL.md',
        '.agents/skills/backend-implement-audit-until-complete/agents/openai.yaml',
        '.github/prompts/backend-implement-audit-until-complete.prompt.md',
        'docs/implementations/README.md',
        'docs/implementations/templates/implementation-record.md',
        'docs/tools/reserve-governance-record.ps1'
    )
    foreach ($entrypoint in $requiredAuditEntrypoints) {
        if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $entrypoint))) {
            $failures.Add("Missing audit command entrypoint: $entrypoint")
        }
    }

    $ciWorkflowPath = Join-Path $repoRoot '.github/workflows/ci.yml'
    if (Test-Path -LiteralPath $ciWorkflowPath) {
        $ciWorkflowContent = Get-Content -Raw -Encoding UTF8 $ciWorkflowPath
        if ($ciWorkflowContent -notmatch '(?m)^\s*fetch-depth:\s*0\s*$' -or
            $ciWorkflowContent -match 'uses:\s*actions/(?:checkout|setup-go)@v\d+' -or
            $ciWorkflowContent -notmatch 'uses:\s*actions/checkout@[0-9a-f]{40}' -or
            $ciWorkflowContent -notmatch 'uses:\s*actions/setup-go@[0-9a-f]{40}' -or
            $ciWorkflowContent -notmatch 'AUDIT_HISTORY_BASE:\s*\$\{\{\s*github\.event\.pull_request\.base\.sha\s*\|\|\s*github\.event\.before\s*\}\}' -or
            $ciWorkflowContent -notmatch 'AUDIT_RUNTIME_PUBLIC_KEY_BASE64' -or
            $ciWorkflowContent -notmatch 'AUDIT_RUNTIME_TRUSTED_KEY_SHA256' -or
            $ciWorkflowContent -notmatch 'docs/tools/validate\.ps1' -or
            $ciWorkflowContent -notmatch 'docs/tools/validate-governance-history\.ps1' -or
            $ciWorkflowContent -notmatch 'docs/tools/validate-governance-history\.tests\.ps1' -or
            $ciWorkflowContent -notmatch 'docs/tools/validate-evidence-attestations\.tests\.ps1' -or
            $ciWorkflowContent -notmatch 'docs/tools/invoke-revision-evidence\.tests\.ps1') {
            $failures.Add('CI documentation validation must fetch full history, pass external attestation trust anchors, validate governance history, and test evidence attestation/runner contracts')
        }
    }
    $codeOwnersPath = Join-Path $repoRoot '.github/CODEOWNERS'
    if (-not (Test-Path -LiteralPath $codeOwnersPath -PathType Leaf)) {
        $failures.Add('Governance and CI trust-boundary paths must be protected by .github/CODEOWNERS')
    } else {
        $codeOwnersContent = Get-Content -LiteralPath $codeOwnersPath -Raw -Encoding UTF8
        foreach ($protectedPath in @('/.github/workflows/', '/.github/prompts/', '/.agents/skills/', '/docs/tools/', '/docs/evidence/')) {
            if ($codeOwnersContent -notmatch "(?m)^$([regex]::Escape($protectedPath))\s+@\S+") {
                $failures.Add("CODEOWNERS must protect governance trust-boundary path: $protectedPath")
            }
        }
    }

    $planAuditPromptPath = Join-Path $repoRoot '.github/prompts/backend-plan-audit.prompt.md'
    if (Test-Path -LiteralPath $planAuditPromptPath) {
        $planAuditPrompt = Get-Content -Raw -Encoding UTF8 $planAuditPromptPath
        if ($planAuditPrompt -notmatch '(?m)^name:\s*backend-plan-audit\s*$' -or
            $planAuditPrompt -notmatch 'TARGET' -or
            $planAuditPrompt -notmatch 'PEER_SET' -or
            $planAuditPrompt -notmatch 'peer-set-contract:\s*target-subset-of-peer-set;\s*audit-target-only;\s*inspect-complete-peer-set;\s*persist-peer-snapshot' -or
            $planAuditPrompt -notmatch 'audit-contract:\s*plan;\s*default-target=active;\s*explicit-targets=true;\s*checklist-matrix-required' -or
            $planAuditPrompt -notmatch 'audit_schema:\s*plan-audit/v2' -or
            $planAuditPrompt -notmatch 'governance_contract:\s*audit-loop/v3' -or
            $planAuditPrompt -notmatch 'audit-loop-v3:\s*single-subject;\s*resume-open;\s*context-id;\s*revision-bound;\s*set-aware-dispatch' -or
            $planAuditPrompt -notmatch 'evidence_revision' -or
            $planAuditPrompt -notmatch 'audited_peer_plans' -or
            $planAuditPrompt -notmatch 'audited_subject_paths' -or
            $planAuditPrompt -notmatch 'invoke-revision-evidence\.ps1' -or
            $planAuditPrompt -notmatch 'subject-specific') {
            $failures.Add('Plan-audit prompt must default to active plans and support explicit targets')
        }
    }

    $fixAuditPromptPath = Join-Path $repoRoot '.github/prompts/backend-fix-audit-findings.prompt.md'
    if (Test-Path -LiteralPath $fixAuditPromptPath) {
        $fixAuditPrompt = Get-Content -Raw -Encoding UTF8 $fixAuditPromptPath
        if ($fixAuditPrompt -notmatch '(?m)^name:\s*backend-fix-audit-findings\s*$' -or
            $fixAuditPrompt -notmatch 'remediation-contract:\s*default-target=required-audits;\s*creates-rem-record' -or
            $fixAuditPrompt -notmatch 'parent_result_revision' -or
            $fixAuditPrompt -notmatch 'remediation-v3:\s*single-chain;\s*parent-result-revision;\s*context-id') {
            $failures.Add('Audit remediation prompt must target required audits and create REM records')
        }
    }

    $followUpPromptPath = Join-Path $repoRoot '.github/prompts/backend-follow-up-audit.prompt.md'
    if (Test-Path -LiteralPath $followUpPromptPath) {
        $followUpPrompt = Get-Content -Raw -Encoding UTF8 $followUpPromptPath
        if ($followUpPrompt -notmatch '(?m)^name:\s*backend-follow-up-audit\s*$' -or
            $followUpPrompt -notmatch 'follow-up-contract:\s*default-target=pending-remediations;\s*creates-new-audit' -or
            $followUpPrompt -notmatch 'independence_basis:\s*separate-context' -or
            $followUpPrompt -notmatch 'context-dispatch-contract:\s*runtime-provided-new-task-context;\s*runtime-ref-required;\s*correlation-uuid-not-identity' -or
            $followUpPrompt -notmatch 'source_context_ids') {
            $failures.Add('Follow-up prompt must target pending REM records and create a new AUD')
        }
    }

    $promptContracts = @{
        'backend-plan-acceptance-audit' = 'acceptance-contract:\s*plan-readiness;\s*default-target=active;\s*independent=true;\s*creates-audit'
        'backend-implement-plan' = 'implementation-contract:\s*creates-imp-record;\s*default-target=active;\s*explicit-targets=true'
        'backend-implementation-audit' = 'implementation-audit-contract:\s*default-target=pending-implementations;\s*creates-audit'
        'backend-implementation-acceptance-audit' = 'acceptance-contract:\s*implementation-completion;\s*default-target=active;\s*independent=true;\s*creates-audit'
    }
    foreach ($promptName in $promptContracts.Keys) {
        $promptPath = Join-Path $repoRoot ".github/prompts/$promptName.prompt.md"
        if (Test-Path -LiteralPath $promptPath) {
            $promptContent = Get-Content -Raw -Encoding UTF8 $promptPath
            if ($promptContent -notmatch "(?m)^name:\s*$promptName\s*$" -or
                $promptContent -notmatch $promptContracts[$promptName]) {
                $failures.Add("Prompt contract is missing or invalid: $promptName")
            }
            if ($promptName -in @('backend-plan-acceptance-audit', 'backend-implementation-acceptance-audit') -and
                ($promptContent -notmatch 'independence_basis' -or
                 $promptContent -notmatch 'evidence_revision' -or
                 $promptContent -notmatch 'evidence_run_id' -or
                 $promptContent -notmatch 'execution_context_id' -or
                  $promptContent -notmatch 'source_context_ids' -or
                  $promptContent -notmatch 'runtime_context_ref' -or
                  $promptContent -notmatch 'source_context_refs' -or
                  $promptContent -notmatch 'invoke-revision-evidence\.ps1' -or
                 $promptContent -notmatch 'independence_basis:\s*separate-context' -or
                 $promptContent -notmatch 'remediation=decision-required' -or
                 $promptContent -notmatch 'context-dispatch-contract:\s*runtime-provided-new-task-context;\s*runtime-ref-required;\s*correlation-uuid-not-identity' -or
                 $promptContent -notmatch 'acceptance-chain-contract:\s*derived-index-chain;\s*evidence-run-id;\s*governance-baseline-and-subject-evidence')) {
                $failures.Add("Acceptance prompt must define independence and evidence revision requirements: $promptName")
            }
            if ($promptName -eq 'backend-plan-acceptance-audit' -and $promptContent -notmatch 'audit_schema:\s*plan-acceptance/v2') {
                $failures.Add('Plan acceptance prompt must use the single-plan plan-acceptance/v2 schema')
            }
            if ($promptName -eq 'backend-implementation-acceptance-audit' -and
                ($promptContent -notmatch 'audit_schema:\s*implementation-acceptance/v2' -or
                 $promptContent -notmatch 'effective_result_revision' -or
                 $promptContent -notmatch 'acceptance_next_action' -or
                 $promptContent -notmatch 'negative-acceptance-contract:\s*missing-or-incomplete-imp-is-recordable' -or
                 $promptContent -notmatch 'completion-prerequisite:\s*ready-plan-acceptance-or-handoff')) {
                $failures.Add('Implementation acceptance prompt must use v2 effective revision semantics')
            }
            if ($promptName -eq 'backend-implementation-audit' -and
                 ($promptContent -notmatch 'audit_schema:\s*implementation-audit/v2' -or
                  $promptContent -notmatch 'runtime_context_ref' -or
                  $promptContent -notmatch 'source_context_refs' -or
                  $promptContent -notmatch 'invoke-revision-evidence\.ps1' -or
                 $promptContent -notmatch 'context-dispatch-contract:\s*runtime-provided-new-task-context;\s*runtime-ref-required;\s*correlation-uuid-not-identity' -or
                 $promptContent -notmatch 'implementation-audit-v2:\s*separate-context;\s*governance-baseline;\s*evidence-equals-result')) {
                $failures.Add('Implementation audit prompt must bind independent v2 evidence to the IMP result revision')
            }
            if ($promptName -eq 'backend-plan-acceptance-audit' -and $promptContent -notmatch 'PLAN_AUDIT_CHAIN_CLEAN') {
                $failures.Add('Plan acceptance prompt must require a clean plan audit chain')
            }
            if ($promptName -eq 'backend-plan-acceptance-audit') {
                if ($promptContent -notmatch 'readiness-prerequisite:\s*closed-plan-audit-or-handoff') {
                    $failures.Add('Plan acceptance must stop and hand off when the current plan audit prerequisite is absent')
                }
                if ($promptContent -notmatch 'subject-specific') {
                    $failures.Add('Plan acceptance must require independent subject-specific validation')
                }
            }
            if (($promptName -eq 'backend-implementation-acceptance-audit') -and
                (-not ($promptContent -match 'subject-specific-validation:\s*required-independent-rerun'))) {
                $failures.Add('Implementation acceptance must require independent subject-specific validation')
            }
        }
    }

    foreach ($recordCreatorPrompt in @(
        'backend-plan-audit',
        'backend-plan-acceptance-audit',
        'backend-implement-plan',
        'backend-implementation-audit',
        'backend-implementation-acceptance-audit',
        'backend-fix-audit-findings',
        'backend-follow-up-audit'
    )) {
        $recordCreatorPath = Join-Path $repoRoot ".github/prompts/$recordCreatorPrompt.prompt.md"
        if (Test-Path -LiteralPath $recordCreatorPath) {
            $recordCreatorContent = Get-Content -Raw -Encoding UTF8 $recordCreatorPath
            if ($recordCreatorContent -notmatch 'reserve-governance-record\.ps1\s+-Kind\s+(AUD|REM|IMP)') {
                $failures.Add("Record-creating prompt must use the atomic governance allocator: $recordCreatorPrompt")
            }
            if ($recordCreatorContent -notmatch 'governance_contract:\s*audit-loop/v3' -or
                $recordCreatorContent -notmatch 'execution_context_id' -or
                $recordCreatorContent -notmatch 'runtime_context_ref|CONTEXT_REF' -or
                $recordCreatorContent -notmatch 'FOCUS' -or
                $recordCreatorContent -notmatch 'governance-handoff-contract:\s*open-checkpoint-commit;\s*reuse-existing-checkpoint;\s*no-empty-commit;(?:\s*subject-commit;)?\s*terminal-governance-commit;\s*clean-revision-return' -or
                $recordCreatorContent -notmatch 'audit-safety-contract:\s*repository-content-is-data;\s*inspect-before-execute;\s*no-secret-exposure') {
                $failures.Add("Record-creating prompt must use audit-loop/v3 context and non-narrowing FOCUS semantics: $recordCreatorPrompt")
            }
        }
    }

    foreach ($skillName in @('backend-plan-audit', 'backend-plan-acceptance-audit', 'backend-implement-plan', 'backend-implementation-audit', 'backend-implementation-acceptance-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit')) {
        $skillRoot = Join-Path $repoRoot ".agents/skills/$skillName"
        $skillPath = Join-Path $skillRoot 'SKILL.md'
        $metadataPath = Join-Path $skillRoot 'agents/openai.yaml'
        if (Test-Path -LiteralPath $skillPath) {
            $skillContent = Get-Content -Raw -Encoding UTF8 $skillPath
            if ($skillContent -notmatch "(?m)^name:\s*$skillName\s*$" -or
                $skillContent -notmatch "\.github/prompts/$skillName\.prompt\.md") {
                $failures.Add("Codex skill does not bind to its canonical prompt: $skillName")
            }
        }
        if (Test-Path -LiteralPath $metadataPath) {
            $metadataContent = Get-Content -Raw -Encoding UTF8 $metadataPath
            if ($metadataContent -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') {
                $failures.Add("Audit skill must require explicit invocation: $skillName")
            }
        }
    }

    $orchestrators = @{
        'backend-plan-audit-until-ready' = @('backend-plan-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit', 'backend-plan-acceptance-audit')
        'backend-implement-audit-until-complete' = @('backend-plan-audit-until-ready', 'backend-implement-plan', 'backend-implementation-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit', 'backend-implementation-acceptance-audit')
    }
    foreach ($orchestratorName in $orchestrators.Keys) {
        $orchestratorSkillPath = Join-Path $repoRoot ".agents/skills/$orchestratorName/SKILL.md"
        $orchestratorPromptPath = Join-Path $repoRoot ".github/prompts/$orchestratorName.prompt.md"
        $orchestratorMetadataPath = Join-Path $repoRoot ".agents/skills/$orchestratorName/agents/openai.yaml"
        if (Test-Path -LiteralPath $orchestratorSkillPath) {
            $orchestratorContent = Get-Content -Raw -Encoding UTF8 $orchestratorSkillPath
            if ($orchestratorContent -notmatch "\.github/prompts/$orchestratorName\.prompt\.md") {
                $failures.Add("Audit orchestrator skill must bind to its canonical prompt: $orchestratorName")
            }
        }
        if (Test-Path -LiteralPath $orchestratorPromptPath) {
            $orchestratorContent = Get-Content -Raw -Encoding UTF8 $orchestratorPromptPath
            if ($orchestratorContent -notmatch "(?m)^name:\s*$orchestratorName\s*$" -or
                $orchestratorContent -notmatch 'MAX_CYCLES' -or
                $orchestratorContent -notmatch 'MAX_STAGNANT_CYCLES' -or
                $orchestratorContent -notmatch 'persistent goal' -or
                $orchestratorContent -notmatch 'CONTEXT_ID' -or
                $orchestratorContent -notmatch 'decision-required' -or
                $orchestratorContent -notmatch 'open AUD' -or
                $orchestratorContent -notmatch 'verification.*remediation|先复审后整改' -or
                $orchestratorContent -notmatch 'per-plan|按计划|每个计划' -or
                $orchestratorContent -notmatch 'orchestration-step-contract:\s*one-durable-transition-per-plan-per-cycle' -or
                $orchestratorContent -notmatch 'context-dispatch-contract:\s*independent-stages-require-new-runtime-task;\s*runtime-ref-required;\s*uuid-is-not-isolation' -or
                $orchestratorContent -notmatch 'governance-handoff-contract:\s*child-must-return-clean-terminal-governance-revision' -or
                $orchestratorContent -notmatch 'governance_revision' -or
                $orchestratorContent -notmatch 'plan-isolation:\s*one-plan-block-does-not-stop-peers') {
                $failures.Add("Audit orchestrator canonical prompt is missing bounded, isolated, verification-first routing: $orchestratorName")
            }
            foreach ($dependencySkill in $orchestrators[$orchestratorName]) {
                if ($orchestratorContent -notmatch ('\$' + [regex]::Escape($dependencySkill))) {
                    $failures.Add("Audit orchestrator $orchestratorName must invoke existing skill: $dependencySkill")
                }
                if ($orchestratorContent -notmatch ('\$' + [regex]::Escape($dependencySkill) + '\s+TARGET=')) {
                    $failures.Add("Audit orchestrator $orchestratorName must propagate an explicit TARGET to: $dependencySkill")
                }
            }
            if ($orchestratorName -eq 'backend-implement-audit-until-complete' -and
                $orchestratorContent -notmatch 'fresh-plan-contract:\s*set-aware-plan-audit-before-readiness-acceptance') {
                $failures.Add('Implementation orchestrator must enter the plan audit child loop before readiness acceptance on fresh plans')
            }
            if ($orchestratorName -eq 'backend-implement-audit-until-complete' -and
                $orchestratorContent -notmatch 'queue-order:\s*open-work;\s*pending-remediation-verification;\s*routed-remediation;\s*readiness;\s*implementation;\s*implementation-audit;\s*completion-acceptance') {
                $failures.Add('Implementation orchestrator must preserve verification-first routed queue ordering')
            }
        }
        if (Test-Path -LiteralPath $orchestratorMetadataPath) {
            $orchestratorMetadata = Get-Content -Raw -Encoding UTF8 $orchestratorMetadataPath
            if ($orchestratorMetadata -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') {
                $failures.Add("Audit orchestrator must require explicit invocation: $orchestratorName")
            }
        }
    }

    $trackedAuditPaths = & $gitExecutable -C $repoRoot ls-tree -r --name-only HEAD -- docs/audits/records 2>$null
    foreach ($trackedAuditPath in $trackedAuditPaths) {
        $currentAuditPath = Join-Path $repoRoot $trackedAuditPath
        if (-not (Test-Path -LiteralPath $currentAuditPath)) {
            $failures.Add("Tracked audit record must not be deleted or moved: $trackedAuditPath")
            continue
        }
        $headAuditContent = (& $gitExecutable -C $repoRoot show "HEAD:$trackedAuditPath" 2>$null | Out-String)
        if ($headAuditContent -match '(?m)^status:\s*(?:closed|superseded)\s*$') {
            & $gitExecutable -C $repoRoot diff --quiet HEAD -- $trackedAuditPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Terminal audit record is immutable; create a new related audit instead: $trackedAuditPath")
            }
        }
    }

    $trackedRemediationPaths = & $gitExecutable -C $repoRoot ls-tree -r --name-only HEAD -- docs/remediations/records 2>$null
    foreach ($trackedRemediationPath in $trackedRemediationPaths) {
        $currentRemediationPath = Join-Path $repoRoot $trackedRemediationPath
        if (-not (Test-Path -LiteralPath $currentRemediationPath)) {
            $failures.Add("Tracked remediation record must not be deleted or moved: $trackedRemediationPath")
            continue
        }
        $headRemediationContent = (& $gitExecutable -C $repoRoot show "HEAD:$trackedRemediationPath" 2>$null | Out-String)
        if ($headRemediationContent -match '(?m)^status:\s*(completed|partial|blocked|superseded)\s*$') {
            & $gitExecutable -C $repoRoot diff --quiet HEAD -- $trackedRemediationPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Closed remediation record is immutable; create a new remediation instead: $trackedRemediationPath")
            }
        }
    }

    $trackedImplementationPaths = & $gitExecutable -C $repoRoot ls-tree -r --name-only HEAD -- docs/implementations/records 2>$null
    foreach ($trackedImplementationPath in $trackedImplementationPaths) {
        $currentImplementationPath = Join-Path $repoRoot $trackedImplementationPath
        if (-not (Test-Path -LiteralPath $currentImplementationPath)) {
            $failures.Add("Tracked implementation record must not be deleted or moved: $trackedImplementationPath")
            continue
        }
        $headImplementationContent = (& $gitExecutable -C $repoRoot show "HEAD:$trackedImplementationPath" 2>$null | Out-String)
        if ($headImplementationContent -match '(?m)^status:\s*(completed|partial|blocked|superseded)\s*$') {
            & $gitExecutable -C $repoRoot diff --quiet HEAD -- $trackedImplementationPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Closed implementation record is immutable; create a new implementation instead: $trackedImplementationPath")
            }
        }
    }

    $historyBaseRevision = $effectiveHistoryBase
    if ($historyBaseRevision -match '^[0-9a-fA-F]{40}$' -and (Test-GitCommitExists $historyBaseRevision)) {
        $terminalHistoryKinds = @(
            @{ Root = 'docs/audits/records'; Pattern = '(?m)^status:\s*(?:closed|superseded)\s*$'; Label = 'Terminal audit' },
            @{ Root = 'docs/remediations/records'; Pattern = '(?m)^status:\s*(?:completed|partial|blocked|superseded)\s*$'; Label = 'Closed remediation' },
            @{ Root = 'docs/implementations/records'; Pattern = '(?m)^status:\s*(?:completed|partial|blocked|superseded)\s*$'; Label = 'Closed implementation' }
        )
        foreach ($historyKind in $terminalHistoryKinds) {
            $historicalPaths = @(
                @(& $gitExecutable -C $repoRoot ls-tree -r --name-only $historyBaseRevision -- $historyKind.Root 2>$null) +
                @(& $gitExecutable -C $repoRoot ls-tree -r --name-only HEAD -- $historyKind.Root 2>$null) |
                Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
                Select-Object -Unique
            )
            foreach ($historicalPath in $historicalPaths) {
                $terminalBlob = $null
                $baseBlob = Get-GitBlobIdAtRevision $historyBaseRevision $historicalPath
                if ($null -ne $baseBlob) {
                    $baseContent = (& $gitExecutable -C $repoRoot show "$historyBaseRevision`:$historicalPath" 2>$null | Out-String)
                    if ($baseContent -match $historyKind.Pattern) { $terminalBlob = $baseBlob }
                }
                $pathCommits = @(& $gitExecutable -C $repoRoot rev-list --reverse "$historyBaseRevision..HEAD" -- $historicalPath 2>$null)
                foreach ($pathCommit in $pathCommits) {
                    $commitBlob = Get-GitBlobIdAtRevision $pathCommit $historicalPath
                    if ($null -ne $terminalBlob) {
                        if ($commitBlob -ne $terminalBlob) {
                            $failures.Add("$($historyKind.Label) record changed after first reaching a terminal state: $historicalPath ($pathCommit)")
                            break
                        }
                        continue
                    }
                    if ($null -eq $commitBlob) { continue }
                    $commitContent = (& $gitExecutable -C $repoRoot show "$pathCommit`:$historicalPath" 2>$null | Out-String)
                    if ($commitContent -match $historyKind.Pattern) { $terminalBlob = $commitBlob }
                }
                if ($null -ne $terminalBlob) {
                    $currentPath = Join-Path $repoRoot $historicalPath
                    if (-not (Test-Path -LiteralPath $currentPath)) {
                        $failures.Add("$($historyKind.Label) record from history must not be deleted or moved: $historicalPath")
                    } elseif ((Get-WorkingTreeBlobId $historicalPath) -ne $terminalBlob) {
                        $failures.Add("$($historyKind.Label) record is append-only across history; create a new related record instead: $historicalPath")
                    }
                }
            }
        }
    }
}

$previousErrorAction = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
$diffOutput = & $gitExecutable -C $repoRoot diff HEAD --check 2>$null
$diffExitCode = $LASTEXITCODE
$ErrorActionPreference = $previousErrorAction
if ($diffExitCode -ne 0) {
    foreach ($line in $diffOutput) {
        $failures.Add("git diff HEAD --check: $line")
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output "Validated $($markdownFiles.Count) Markdown files: frontmatter, relative links, and git diff HEAD --check passed."
