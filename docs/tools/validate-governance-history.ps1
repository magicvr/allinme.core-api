param(
    [string]$HistoryBase,
    [string]$RepositoryRoot
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}

$gitCommand = Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath($gitCommand.Source)
$repoPrefix = $repoRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if ($gitExecutable.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing repository-local git executable: $gitExecutable"
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

if ([string]::IsNullOrWhiteSpace($HistoryBase)) { $HistoryBase = $env:AUDIT_HISTORY_BASE }
if ([string]::IsNullOrWhiteSpace($HistoryBase)) { throw 'HistoryBase or AUDIT_HISTORY_BASE is required for governance history validation' }
if ($HistoryBase -match '^0{40}$') {
    $rootResult = Invoke-GitProbe @('rev-list', '--max-parents=0', 'HEAD')
    if ($rootResult.ExitCode -ne 0 -or $rootResult.Output.Count -eq 0) { throw 'Unable to resolve the repository root commit' }
    $HistoryBase = @($rootResult.Output | Select-Object -First 1)[0].Trim()
}

$baseResult = Invoke-GitProbe @('rev-parse', '--verify', "$HistoryBase`^{commit}")
$baseRevision = if ($baseResult.Output.Count -gt 0) { @($baseResult.Output | Select-Object -First 1)[0].Trim() } else { '' }
if ($baseResult.ExitCode -ne 0 -or $baseRevision -notmatch '^[0-9a-f]{40}$') { throw "HistoryBase is not a commit: $HistoryBase" }
$headResult = Invoke-GitProbe @('rev-parse', '--verify', 'HEAD^{commit}')
$headRevision = if ($headResult.Output.Count -gt 0) { @($headResult.Output | Select-Object -First 1)[0].Trim() } else { '' }
if ($headResult.ExitCode -ne 0 -or $headRevision -notmatch '^[0-9a-f]{40}$') { throw 'HEAD is not a commit' }
$baseAncestorResult = Invoke-GitProbe @('merge-base', '--is-ancestor', $baseRevision, $headRevision)
if ($baseAncestorResult.ExitCode -ne 0) { throw "HistoryBase must be an ancestor of HEAD: $baseRevision" }

$failures = New-Object Collections.Generic.List[string]

function Get-ContentAtRevision([string]$Revision, [string]$Path) {
    $contentResult = Invoke-GitProbe @('show', "$Revision`:$Path")
    if ($contentResult.ExitCode -ne 0) { return $null }
    return ($contentResult.Output -join [Environment]::NewLine)
}

function Get-Field([string]$Content, [string]$Name) {
    if ([string]::IsNullOrWhiteSpace($Content)) { return $null }
    $match = [regex]::Match($Content, "(?m)^$([regex]::Escape($Name)):\s*(?<value>.+?)\s*$")
    if (-not $match.Success) { return $null }
    return $match.Groups['value'].Value
}

function Test-Ancestor([string]$Ancestor, [string]$Descendant) {
    return (Invoke-GitProbe @('merge-base', '--is-ancestor', $Ancestor, $Descendant)).ExitCode -eq 0
}

function Get-CommitPaths([string]$Revision) {
    $pathsResult = Invoke-GitProbe @('diff-tree', '--root', '--no-commit-id', '--name-only', '-r', $Revision)
    if ($pathsResult.ExitCode -ne 0) { throw "Unable to inspect governance commit paths: $Revision" }
    return @($pathsResult.Output | Where-Object { $_ } | ForEach-Object { $_.Replace('\', '/') } | Sort-Object -Unique)
}

function Get-BlobAtRevision([string]$Revision, [string]$Path) {
    $blobResult = Invoke-GitProbe @('rev-parse', '--verify', "$Revision`:$Path")
    if ($blobResult.ExitCode -ne 0 -or $blobResult.Output.Count -eq 0) { return $null }
    $blob = @($blobResult.Output | Select-Object -First 1)[0].Trim()
    if ($blob -notmatch '^[0-9a-f]{40}$') { return $null }
    return $blob
}

function Get-RecordId([string]$Content) {
    foreach ($field in @('audit_id', 'remediation_id', 'implementation_id')) {
        $value = Get-Field $Content $field
        if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    }
    return $null
}

function Get-ListValues([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -eq 'none') { return @() }
    return @($Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

$roots = @('docs/audits/records', 'docs/remediations/records', 'docs/implementations/records')
$historyResult = Invoke-GitProbe (@('log', '--format=', '--name-only', "$baseRevision..$headRevision", '--') + $roots)
if ($historyResult.ExitCode -ne 0) { throw 'Unable to inspect governance record history' }
$historyPaths = @($historyResult.Output | Where-Object { $_ -match '^docs/(?:audits|remediations|implementations)/records/.+\.md$' } | Sort-Object -Unique)

# Attestation files can change without touching their records, so record-path history
# alone is insufficient. Pin every pre-existing record's runtime attestation blob
# directly from HistoryBase to HEAD.
$headRecordResult = Invoke-GitProbe (@('ls-tree', '-r', '--name-only', $headRevision, '--') + $roots)
if ($headRecordResult.ExitCode -ne 0) { throw 'Unable to enumerate governance records at HEAD' }
foreach ($recordPath in @($headRecordResult.Output | Where-Object { $_ -match '^docs/(?:audits|remediations|implementations)/records/.+\.md$' })) {
    $baseRecordContent = Get-ContentAtRevision $baseRevision $recordPath
    if ($null -eq $baseRecordContent) { continue }
    $headRecordContent = Get-ContentAtRevision $headRevision $recordPath
    $baseRuntimePath = Get-Field $baseRecordContent 'runtime_context_attestation'
    $headRuntimePath = Get-Field $headRecordContent 'runtime_context_attestation'
    if ([string]::IsNullOrWhiteSpace($baseRuntimePath)) { continue }
    if ($headRuntimePath -ne $baseRuntimePath -or
        (Get-BlobAtRevision $baseRevision $baseRuntimePath) -ne (Get-BlobAtRevision $headRevision $baseRuntimePath)) {
        $failures.Add("Runtime context attestation blob must remain immutable after HistoryBase: $recordPath ($baseRuntimePath)")
    }
}

foreach ($path in $historyPaths) {
    $basePathResult = Invoke-GitProbe @('cat-file', '-e', "$baseRevision`:$path")
    $isNewRecord = $basePathResult.ExitCode -ne 0
    $baseContent = if ($isNewRecord) { $null } else { Get-ContentAtRevision $baseRevision $path }
    $headContent = Get-ContentAtRevision $headRevision $path
    if ($null -eq $headContent) {
        $failures.Add("Governance record was deleted before HEAD: $path")
        continue
    }

    $kind = if ($path.StartsWith('docs/audits/records/')) { 'AUD' } elseif ($path.StartsWith('docs/remediations/records/')) { 'REM' } else { 'IMP' }
    $openState = if ($kind -eq 'AUD') { 'open' } else { 'in-progress' }
    $terminalStates = if ($kind -eq 'AUD') { @('closed', 'superseded') } else { @('completed', 'partial', 'blocked', 'superseded') }
    $primaryIndex = if ($kind -eq 'AUD') { 'docs/audits/README.md' } elseif ($kind -eq 'REM') { 'docs/remediations/README.md' } else { 'docs/implementations/README.md' }
    $pathHistoryResult = Invoke-GitProbe @('rev-list', '--reverse', "$baseRevision..$headRevision", '--', $path)
    if ($pathHistoryResult.ExitCode -ne 0) {
        $failures.Add("Unable to inspect governance record commits: $path")
        continue
    }
    $pathCommits = @($pathHistoryResult.Output)
    if ($pathCommits.Count -eq 0) {
        $failures.Add("New governance record has no commits: $path")
        continue
    }

    $openCommit = $null
    $openContent = $null
    $terminalCommit = $null
    if ($isNewRecord) {
        foreach ($commit in $pathCommits) {
            $content = Get-ContentAtRevision $commit $path
            $status = Get-Field $content 'status'
            if ($null -eq $openCommit -and $status -eq $openState) {
                $openCommit = $commit
                $openContent = $content
            }
            if ($null -eq $terminalCommit -and $terminalStates -contains $status) { $terminalCommit = $commit }
        }
    } else {
        $baseStatus = Get-Field $baseContent 'status'
        if ($terminalStates -contains $baseStatus) {
            $failures.Add("Terminal governance record changed after HistoryBase: $path")
            continue
        }
        if ($baseStatus -ne $openState) {
            $failures.Add("Existing governance record has an invalid HistoryBase state: $path ($baseStatus)")
            continue
        }
        $openCommitResult = Invoke-GitProbe @('rev-list', '-1', $baseRevision, '--', $path)
        $openCommit = if ($openCommitResult.ExitCode -eq 0 -and $openCommitResult.Output.Count -gt 0) {
            @($openCommitResult.Output | Select-Object -First 1)[0].Trim()
        } else { $null }
        $openContent = if ($null -ne $openCommit) { Get-ContentAtRevision $openCommit $path } else { $null }
        foreach ($commit in $pathCommits) {
            $content = Get-ContentAtRevision $commit $path
            $status = Get-Field $content 'status'
            if ($null -eq $terminalCommit -and $terminalStates -contains $status) { $terminalCommit = $commit }
        }
    }
    if ($null -eq $openCommit) {
        $failures.Add("New governance record lacks a committed open checkpoint: $path")
        continue
    }
    if ($isNewRecord -and $pathCommits[0] -ne $openCommit) {
        $failures.Add("New governance record must be created in its open state: $path")
    }
    $openPaths = @(Get-CommitPaths $openCommit)
    if ($openPaths -notcontains $path -or $openPaths -notcontains $primaryIndex) {
        $failures.Add("Open checkpoint must atomically include record and primary index: $path ($openCommit)")
    }
    $runtimeAttestationPath = Get-Field $openContent 'runtime_context_attestation'
    if ($runtimeAttestationPath -notmatch '^docs/evidence/runtime-attestations/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\.json$') {
        $failures.Add("Open checkpoint must bind a stable runtime context attestation: $path ($openCommit)")
        $runtimeAttestationPath = $null
    } elseif ($openPaths -notcontains $runtimeAttestationPath) {
        $failures.Add("Open checkpoint must atomically include its runtime context attestation: $path ($openCommit)")
    } else {
        $openRuntimeBlob = Get-BlobAtRevision $openCommit $runtimeAttestationPath
        $headRuntimeBlob = Get-BlobAtRevision $headRevision $runtimeAttestationPath
        if ($null -eq $openRuntimeBlob -or $headRuntimeBlob -ne $openRuntimeBlob) {
            $failures.Add("Runtime context attestation blob must remain immutable after the open checkpoint: $path ($runtimeAttestationPath)")
        }
    }
    $allowedOpenPaths = @(
        $path,
        'docs/audits/README.md',
        'docs/remediations/README.md',
        'docs/implementations/README.md'
    )
    if (-not [string]::IsNullOrWhiteSpace($runtimeAttestationPath)) {
        $allowedOpenPaths += $runtimeAttestationPath
    }
    $openRecordId = Get-RecordId $openContent
    foreach ($predecessorId in @(Get-ListValues (Get-Field $openContent 'supersedes'))) {
        foreach ($candidatePath in @($openPaths | Where-Object { $_ -match "^docs/(?:audits|remediations|implementations)/records/$([regex]::Escape($predecessorId))-" })) {
            $candidateContent = Get-ContentAtRevision $openCommit $candidatePath
            if ((Get-Field $candidateContent 'status') -eq 'superseded' -and
                (Get-Field $candidateContent 'superseded_by') -eq $openRecordId -and
                (Get-Field $candidateContent 'supersession_reason') -eq 'context-loss') {
                $allowedOpenPaths += $candidatePath
            }
        }
    }
    $unexpectedOpenPaths = @($openPaths | Where-Object { $_ -notin $allowedOpenPaths })
    if ($unexpectedOpenPaths.Count -gt 0) {
        $failures.Add("Open checkpoint mixes subject or unrelated record paths: $path ($($unexpectedOpenPaths -join ', '))")
    }
    $openIndexContent = Get-ContentAtRevision $openCommit $primaryIndex
    $recordName = [IO.Path]::GetFileName($path)
    $openIndexLine = [regex]::Match($openIndexContent, "(?m)^.*\]\(\./records/$([regex]::Escape($recordName))\).*status=$([regex]::Escape($openState)).*$")
    if (-not $openIndexLine.Success) {
        $failures.Add("Open checkpoint index must expose the matching open state: $path ($openCommit)")
    }

    $headStatus = Get-Field $headContent 'status'
    foreach ($immutableField in @(
        'governance_contract',
        'workflow_contract_revision',
        'baseline',
        'evidence_revision',
        'evidence_worktree_revision',
        'evidence_runner',
        'evidence_run_id',
        'evidence_artifact',
        'evidence_attestation',
        'runtime_context_attestation'
    )) {
        $openValue = Get-Field $openContent $immutableField
        $headValue = Get-Field $headContent $immutableField
        if (-not [string]::IsNullOrWhiteSpace($openValue) -and $openValue -ne $headValue) {
            $failures.Add("Governance record field '$immutableField' cannot be rebound after the open checkpoint: $path")
        }
    }
    if ($terminalStates -contains $headStatus) {
        if ($null -eq $terminalCommit -or $terminalCommit -eq $openCommit) {
            $failures.Add("Terminal governance record must follow an earlier open checkpoint: $path")
            continue
        }
        if (-not (Test-Ancestor $openCommit $terminalCommit)) {
            $failures.Add("Terminal governance commit must descend from the open checkpoint: $path")
            continue
        }
        if ($pathCommits[-1] -ne $terminalCommit) {
            $failures.Add("Terminal governance record changed after its terminal commit: $path")
        }
        $terminalPaths = @(Get-CommitPaths $terminalCommit)
        if ($terminalPaths -notcontains $path -or $terminalPaths -notcontains $primaryIndex) {
            $failures.Add("Terminal governance commit must atomically include record and primary index: $path ($terminalCommit)")
        }
        $allowedTerminalPaths = @(
            $path,
            'docs/audits/README.md',
            'docs/remediations/README.md',
            'docs/implementations/README.md'
        )
        if ($kind -eq 'AUD' -and $headStatus -eq 'closed' -and (Get-Field $headContent 'workflow_contract_revision') -eq 'audit-runtime/v1') {
            $evidenceRunId = Get-Field $headContent 'evidence_run_id'
            $runIdMatch = [regex]::Match(
                [string]$evidenceRunId,
                '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$'
            )
            if (-not $runIdMatch.Success) {
                $failures.Add("Terminal audit-runtime/v1 AUD must bind a UUIDv4 evidence_run_id: $path")
            } else {
                $normalizedRunId = $evidenceRunId.ToLowerInvariant()
                $expectedEvidenceArtifact = "docs/evidence/runs/$normalizedRunId/evidence.json"
                $expectedEvidenceAttestation = "docs/evidence/runs/$normalizedRunId/attestation.json"
                if ((Get-Field $headContent 'evidence_artifact') -ne $expectedEvidenceArtifact -or
                    (Get-Field $headContent 'evidence_attestation') -ne $expectedEvidenceAttestation) {
                    $failures.Add("Terminal audit-runtime/v1 AUD must bind exact evidence artifact and attestation paths: $path")
                }
                foreach ($evidencePath in @($expectedEvidenceArtifact, $expectedEvidenceAttestation)) {
                    $allowedTerminalPaths += $evidencePath
                    if ($terminalPaths -notcontains $evidencePath) {
                        $failures.Add("Terminal audit-runtime/v1 commit must atomically include signed evidence: $path ($evidencePath)")
                    }
                }
            }
        }
        if ($headStatus -eq 'superseded') {
            $supersededBy = Get-Field $headContent 'superseded_by'
            $currentRecordId = Get-RecordId $headContent
            foreach ($replacementPath in @($terminalPaths | Where-Object { $_ -match "^docs/(?:audits|remediations|implementations)/records/$([regex]::Escape($supersededBy))-" })) {
                $replacementContent = Get-ContentAtRevision $terminalCommit $replacementPath
                if ((Get-Field $replacementContent 'status') -eq $openState -and
                    (Get-ListValues (Get-Field $replacementContent 'supersedes')) -contains $currentRecordId) {
                    $allowedTerminalPaths += $replacementPath
                    $replacementAttestation = Get-Field $replacementContent 'runtime_context_attestation'
                    if (-not [string]::IsNullOrWhiteSpace($replacementAttestation)) {
                        $allowedTerminalPaths += $replacementAttestation
                    }
                }
            }
        }
        $unexpectedTerminalPaths = @($terminalPaths | Where-Object { $_ -notin $allowedTerminalPaths })
        if ($unexpectedTerminalPaths.Count -gt 0) {
            $failures.Add("Terminal governance commit mixes non-transaction paths: $path ($($unexpectedTerminalPaths -join ', '))")
        }
        $terminalIndexContent = Get-ContentAtRevision $terminalCommit $primaryIndex
        $terminalIndexLine = [regex]::Match($terminalIndexContent, "(?m)^.*\]\(\./records/$([regex]::Escape($recordName))\).*status=$([regex]::Escape($headStatus)).*$")
        if (-not $terminalIndexLine.Success) {
            $failures.Add("Terminal governance index must expose the matching terminal state: $path ($terminalCommit)")
        }

        if ($kind -eq 'AUD') {
            $evidenceValue = Get-Field $headContent 'evidence_revision'
            if ($evidenceValue -notmatch '^git:(?<sha>[0-9a-fA-F]{40})(?:;\s*worktree:clean)?$') {
                $failures.Add("Terminal AUD must bind a full evidence_revision: $path")
            } else {
                $evidenceRevision = $Matches['sha'].ToLowerInvariant()
                if ($evidenceRevision -eq $openCommit -or -not (Test-Ancestor $evidenceRevision $openCommit)) {
                    $failures.Add("AUD evidence/open/terminal ancestry is invalid: $path")
                }
            }
        } elseif ($headStatus -notin @('blocked', 'superseded')) {
            $resultValue = Get-Field $headContent 'result_revision'
            if ($resultValue -notmatch '^git:(?<sha>[0-9a-fA-F]{40})$') {
                $failures.Add("Terminal $kind must bind a full result_revision: $path")
            } else {
                $resultRevision = $Matches['sha'].ToLowerInvariant()
                if ($resultRevision -eq $openCommit -or $resultRevision -eq $terminalCommit -or -not (Test-Ancestor $openCommit $resultRevision) -or -not (Test-Ancestor $resultRevision $terminalCommit)) {
                    $failures.Add("$kind open/result/terminal ancestry is invalid: $path")
                }
                $resultPaths = @(Get-CommitPaths $resultRevision)
                $governanceResultPaths = @($resultPaths | Where-Object { $_ -match '^docs/(?:audits|remediations|implementations)/(?:README\.md|records/)' })
                if ($governanceResultPaths.Count -gt 0) {
                    $failures.Add("$kind result_revision must be a pure subject commit: $path ($($governanceResultPaths -join ', '))")
                }
                $subjectPaths = @($resultPaths | Where-Object { $_ -notmatch '^docs/(?:audits|remediations|implementations)/(?:README\.md|records/)' })
                if ($subjectPaths.Count -eq 0) {
                    $failures.Add("$kind result_revision must contain a distinct subject change: $path ($resultRevision)")
                }
            }
        }
    } elseif ($headStatus -ne $openState) {
        $failures.Add("New governance record has an invalid nonterminal status: $path ($headStatus)")
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output "Governance history passed from $baseRevision to ${headRevision}: open checkpoints, subject/evidence ancestry, terminal commits, and exact transaction paths are valid."
