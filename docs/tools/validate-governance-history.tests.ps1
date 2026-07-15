$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$validatorScript = Join-Path $PSScriptRoot 'validate-governance-history.ps1'
$fixtureBase = Join-Path $repoRoot '.tmp'
$fixtureRoot = Join-Path $fixtureBase ('.validate-governance-history-' + [Guid]::NewGuid().ToString('N'))
$gitExecutable = (Get-Command git -CommandType Application | Select-Object -First 1).Source

function Set-Utf8File([string]$Path, [string]$Content) {
    $parent = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        [IO.Directory]::CreateDirectory($parent) | Out-Null
    }
    $normalized = $Content.Replace("`r`n", "`n").Replace("`r", "`n").Replace("`n", [Environment]::NewLine)
    [IO.File]::WriteAllText($Path, $normalized, (New-Object Text.UTF8Encoding($false)))
}

function Invoke-Git([string]$Repository, [string[]]$Arguments) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $gitExecutable -C $Repository @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    if ($exitCode -ne 0) {
        throw "git $($Arguments -join ' ') failed in $Repository`: $($output -join [Environment]::NewLine)"
    }
    return $output
}

function Get-GitValue([string]$Repository, [string[]]$Arguments) {
    return ([string](Invoke-Git $Repository $Arguments | Select-Object -First 1)).Trim()
}

function Get-Head([string]$Repository) {
    return Get-GitValue $Repository @('rev-parse', 'HEAD')
}

function Commit-Paths([string]$Repository, [string[]]$Paths, [string]$Message) {
    Invoke-Git $Repository (@('add', '--') + $Paths) | Out-Null
    Invoke-Git $Repository @('commit', '-q', '-m', $Message) | Out-Null
    return Get-Head $Repository
}

function New-TestRepository([string]$Name) {
    $path = Join-Path $fixtureRoot $Name
    [IO.Directory]::CreateDirectory($path) | Out-Null
    Invoke-Git $path @('init', '-q') | Out-Null
    Invoke-Git $path @('config', 'user.name', 'Governance History Test') | Out-Null
    Invoke-Git $path @('config', 'user.email', 'governance-history@example.invalid') | Out-Null
    Invoke-Git $path @('config', 'commit.gpgSign', 'false') | Out-Null
    Invoke-Git $path @('config', 'core.autocrlf', 'false') | Out-Null
    Set-Utf8File (Join-Path $path 'docs\audits\README.md') "# Audits`n"
    Set-Utf8File (Join-Path $path 'docs\remediations\README.md') "# Remediations`n"
    Set-Utf8File (Join-Path $path 'docs\implementations\README.md') "# Implementations`n"
    Set-Utf8File (Join-Path $path 'src\base.txt') "base subject`n"
    Invoke-Git $path @('add', '.') | Out-Null
    Invoke-Git $path @('commit', '-q', '-m', 'fixture base') | Out-Null
    return [pscustomobject]@{
        Path = $path
        Base = Get-Head $path
    }
}

function Invoke-HistoryValidator([string]$Repository, [string]$HistoryBase) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $validatorScript -HistoryBase $HistoryBase -RepositoryRoot $Repository 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    return [pscustomobject]@{
        ExitCode = $exitCode
        Output = ($output | Out-String).Trim()
    }
}

function Assert-Pass([string]$Label, [pscustomobject]$Fixture) {
    $result = Invoke-HistoryValidator $Fixture.Path $Fixture.Base
    if ($result.ExitCode -ne 0) {
        throw "$Label unexpectedly failed: $($result.Output)"
    }
}

function Assert-Fail([string]$Label, [pscustomobject]$Fixture, [string]$Pattern) {
    $result = Invoke-HistoryValidator $Fixture.Path $Fixture.Base
    if ($result.ExitCode -eq 0 -or $result.Output -notmatch $Pattern) {
        throw "$Label was not rejected with /$Pattern/: $($result.Output)"
    }
}

function Get-AuditRecord(
    [string]$Status,
    [string]$EvidenceRevision,
    [string]$EvidenceRunId,
    [string]$RuntimeAttestationPath
) {
    $runtimeLine = if ([string]::IsNullOrWhiteSpace($RuntimeAttestationPath)) {
        ''
    } else {
        "runtime_context_attestation: $RuntimeAttestationPath`n"
    }
    return @"
---
status: $Status
workflow_contract_revision: audit-runtime/v1
${runtimeLine}evidence_revision: git:$EvidenceRevision; worktree:clean
evidence_run_id: $EvidenceRunId
evidence_artifact: docs/evidence/runs/$EvidenceRunId/evidence.json
evidence_attestation: docs/evidence/runs/$EvidenceRunId/attestation.json
---

# Governance history audit fixture
"@
}

function Add-AuditHistory(
    [pscustomobject]$Fixture,
    [string]$EvidenceRevision,
    [bool]$IncludeRuntimeField = $true,
    [bool]$IncludeRuntimeFile = $true,
    [bool]$IncludeEvidenceArtifact = $true,
    [bool]$IncludeEvidenceAttestation = $true,
    [bool]$IncludeUnrelatedTerminalFile = $false
) {
    $recordName = 'AUD-0001-20260715-history-fixture.md'
    $recordPath = "docs/audits/records/$recordName"
    $indexPath = 'docs/audits/README.md'
    $runtimePath = 'docs/evidence/runtime-attestations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1.json'
    $runId = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1'
    $boundRuntimePath = if ($IncludeRuntimeField) { $runtimePath } else { $null }

    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-AuditRecord 'open' $EvidenceRevision $runId $boundRuntimePath)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# Audits`n`n- [AUD-0001](./records/$recordName) status=open`n"
    $openPaths = @($recordPath, $indexPath)
    if ($IncludeRuntimeFile) {
        Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{}`n"
        $openPaths += $runtimePath
    }
    $openRevision = Commit-Paths $Fixture.Path $openPaths 'open AUD checkpoint'

    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-AuditRecord 'closed' $EvidenceRevision $runId $boundRuntimePath)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# Audits`n`n- [AUD-0001](./records/$recordName) status=closed`n"
    $terminalPaths = @($recordPath, $indexPath)
    if ($IncludeEvidenceArtifact) {
        $artifactPath = "docs/evidence/runs/$runId/evidence.json"
        Set-Utf8File (Join-Path $Fixture.Path $artifactPath) "{}`n"
        $terminalPaths += $artifactPath
    }
    if ($IncludeEvidenceAttestation) {
        $attestationPath = "docs/evidence/runs/$runId/attestation.json"
        Set-Utf8File (Join-Path $Fixture.Path $attestationPath) "{}`n"
        $terminalPaths += $attestationPath
    }
    if ($IncludeUnrelatedTerminalFile) {
        $unrelatedPath = 'terminal-unrelated.txt'
        Set-Utf8File (Join-Path $Fixture.Path $unrelatedPath) "must not share the terminal transaction`n"
        $terminalPaths += $unrelatedPath
    }
    $terminalRevision = Commit-Paths $Fixture.Path $terminalPaths 'close AUD checkpoint'
    return [pscustomobject]@{
        Open = $openRevision
        Terminal = $terminalRevision
    }
}

function Add-ExistingOpenAuditTransition(
    [pscustomobject]$Fixture,
    [bool]$IncludeIndex = $true,
    [bool]$IncludeEvidenceAttestation = $true,
    [bool]$IncludeUnrelatedTerminalFile = $false,
    [bool]$TamperRuntimeAttestation = $false
) {
    $subjectRevision = $Fixture.Base
    $recordName = 'AUD-0001-20260715-history-fixture.md'
    $recordPath = "docs/audits/records/$recordName"
    $indexPath = 'docs/audits/README.md'
    $runtimePath = 'docs/evidence/runtime-attestations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1.json'
    $runId = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1'
    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-AuditRecord 'open' $subjectRevision $runId $runtimePath)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# Audits`n`n- [AUD-0001](./records/$recordName) status=open`n"
    Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{}`n"
    $Fixture.Base = Commit-Paths $Fixture.Path @($recordPath, $indexPath, $runtimePath) 'historical open AUD checkpoint'

    if ($TamperRuntimeAttestation) {
        Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{`"tampered`":true}`n"
        Commit-Paths $Fixture.Path @($runtimePath) 'tamper historical runtime attestation' | Out-Null
    }

    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-AuditRecord 'closed' $subjectRevision $runId $runtimePath)
    $terminalPaths = @($recordPath)
    if ($IncludeIndex) {
        Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# Audits`n`n- [AUD-0001](./records/$recordName) status=closed`n"
        $terminalPaths += $indexPath
    }
    $artifactPath = "docs/evidence/runs/$runId/evidence.json"
    Set-Utf8File (Join-Path $Fixture.Path $artifactPath) "{}`n"
    $terminalPaths += $artifactPath
    if ($IncludeEvidenceAttestation) {
        $attestationPath = "docs/evidence/runs/$runId/attestation.json"
        Set-Utf8File (Join-Path $Fixture.Path $attestationPath) "{}`n"
        $terminalPaths += $attestationPath
    }
    if ($IncludeUnrelatedTerminalFile) {
        Set-Utf8File (Join-Path $Fixture.Path 'src/mixed-terminal.txt') "must not share terminal transaction`n"
        $terminalPaths += 'src/mixed-terminal.txt'
    }
    Commit-Paths $Fixture.Path $terminalPaths 'terminalize historical open AUD' | Out-Null
}

function Add-ExistingOpenSubjectTransition([pscustomobject]$Fixture, [string]$Kind) {
    if ($Kind -eq 'REM') {
        $root = 'docs/remediations'; $recordName = 'REM-0001-20260715-history-fixture.md'; $runtimePath = 'docs/evidence/runtime-attestations/cccccccc-cccc-4ccc-8ccc-ccccccccccc1.json'
    } elseif ($Kind -eq 'IMP') {
        $root = 'docs/implementations'; $recordName = 'IMP-0001-20260715-history-fixture.md'; $runtimePath = 'docs/evidence/runtime-attestations/dddddddd-dddd-4ddd-8ddd-ddddddddddd1.json'
    } else { throw "Unsupported fixture kind: $Kind" }
    $recordPath = "$root/records/$recordName"; $indexPath = "$root/README.md"
    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-SubjectRecord $Kind 'in-progress' $runtimePath $null)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# $Kind records`n`n- [$Kind-0001](./records/$recordName) status=in-progress`n"
    Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{}`n"
    $Fixture.Base = Commit-Paths $Fixture.Path @($recordPath, $indexPath, $runtimePath) "historical open $Kind checkpoint"
    $subjectPath = "src/$($Kind.ToLowerInvariant())-existing-result.txt"
    Set-Utf8File (Join-Path $Fixture.Path $subjectPath) "$Kind existing-open subject result`n"
    $resultRevision = Commit-Paths $Fixture.Path @($subjectPath) "$Kind existing-open subject result"
    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-SubjectRecord $Kind 'completed' $runtimePath $resultRevision)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# $Kind records`n`n- [$Kind-0001](./records/$recordName) status=completed`n"
    Commit-Paths $Fixture.Path @($recordPath, $indexPath) "terminalize historical open $Kind" | Out-Null
}

function Add-AttestationOnlyTamper([pscustomobject]$Fixture) {
    $recordName = 'AUD-0001-20260715-history-fixture.md'
    $recordPath = "docs/audits/records/$recordName"
    $indexPath = 'docs/audits/README.md'
    $runtimePath = 'docs/evidence/runtime-attestations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1.json'
    $runId = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1'
    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-AuditRecord 'open' $Fixture.Base $runId $runtimePath)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# Audits`n`n- [AUD-0001](./records/$recordName) status=open`n"
    Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{}`n"
    $Fixture.Base = Commit-Paths $Fixture.Path @($recordPath, $indexPath, $runtimePath) 'historical open AUD for attestation pin'
    Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{`"tampered`":true}`n"
    Commit-Paths $Fixture.Path @($runtimePath) 'attestation-only tamper' | Out-Null
}

function Get-SubjectRecord(
    [string]$Kind,
    [string]$Status,
    [string]$RuntimeAttestationPath,
    [string]$ResultRevision
) {
    $resultLine = if ([string]::IsNullOrWhiteSpace($ResultRevision)) { '' } else { "result_revision: git:$ResultRevision`n" }
    return @"
---
status: $Status
runtime_context_attestation: $RuntimeAttestationPath
${resultLine}---

# $Kind governance history fixture
"@
}

function Add-SubjectGovernanceHistory(
    [pscustomobject]$Fixture,
    [string]$Kind,
    [bool]$ImpureResult,
    [string]$TerminalStatus = 'completed',
    [bool]$MixTerminal = $false
) {
    if ($Kind -eq 'REM') {
        $root = 'docs/remediations'
        $recordName = 'REM-0001-20260715-history-fixture.md'
        $runtimePath = 'docs/evidence/runtime-attestations/cccccccc-cccc-4ccc-8ccc-ccccccccccc1.json'
    } elseif ($Kind -eq 'IMP') {
        $root = 'docs/implementations'
        $recordName = 'IMP-0001-20260715-history-fixture.md'
        $runtimePath = 'docs/evidence/runtime-attestations/dddddddd-dddd-4ddd-8ddd-ddddddddddd1.json'
    } else {
        throw "Unsupported fixture kind: $Kind"
    }
    $recordPath = "$root/records/$recordName"
    $indexPath = "$root/README.md"

    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-SubjectRecord $Kind 'in-progress' $runtimePath $null)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# $Kind records`n`n- [$Kind-0001](./records/$recordName) status=in-progress`n"
    Set-Utf8File (Join-Path $Fixture.Path $runtimePath) "{}`n"
    $openRevision = Commit-Paths $Fixture.Path @($recordPath, $indexPath, $runtimePath) "open $Kind checkpoint"

    $resultRevision = $null
    if ($TerminalStatus -ne 'blocked') {
        $subjectPath = "src/$($Kind.ToLowerInvariant())-result.txt"
        Set-Utf8File (Join-Path $Fixture.Path $subjectPath) "$Kind subject result`n"
        $resultPaths = @($subjectPath)
        if ($ImpureResult) {
            Set-Utf8File (Join-Path $Fixture.Path 'docs/audits/README.md') "# Audits`n`nimpure $Kind result governance change`n"
            $resultPaths += 'docs/audits/README.md'
        }
        $resultRevision = Commit-Paths $Fixture.Path $resultPaths "$Kind subject result"
    }

    Set-Utf8File (Join-Path $Fixture.Path $recordPath) (Get-SubjectRecord $Kind $TerminalStatus $runtimePath $resultRevision)
    Set-Utf8File (Join-Path $Fixture.Path $indexPath) "# $Kind records`n`n- [$Kind-0001](./records/$recordName) status=$TerminalStatus`n"
    $terminalPaths = @($recordPath, $indexPath)
    if ($MixTerminal) {
        Set-Utf8File (Join-Path $Fixture.Path 'src/mixed-subject-terminal.txt') "must not share terminal transaction`n"
        $terminalPaths += 'src/mixed-subject-terminal.txt'
    }
    $terminalRevision = Commit-Paths $Fixture.Path $terminalPaths "$TerminalStatus $Kind checkpoint"
    return [pscustomobject]@{
        Open = $openRevision
        Result = $resultRevision
        Terminal = $terminalRevision
    }
}

try {
    [IO.Directory]::CreateDirectory($fixtureBase) | Out-Null
    [IO.Directory]::CreateDirectory($fixtureRoot) | Out-Null

    $validAudit = New-TestRepository 'audit-valid'
    Add-AuditHistory $validAudit $validAudit.Base | Out-Null
    Assert-Pass 'valid audit-runtime/v1 base/open/terminal chain' $validAudit

    $missingRuntimePath = New-TestRepository 'audit-missing-runtime-path'
    Add-AuditHistory $missingRuntimePath $missingRuntimePath.Base $false $false | Out-Null
    Assert-Fail 'AUD open checkpoint missing runtime attestation path' $missingRuntimePath 'bind a stable runtime context attestation'

    $missingRuntimeFile = New-TestRepository 'audit-missing-runtime-file'
    Add-AuditHistory $missingRuntimeFile $missingRuntimeFile.Base $true $false | Out-Null
    Assert-Fail 'AUD open checkpoint missing runtime attestation file' $missingRuntimeFile 'atomically include its runtime context attestation'

    $missingEvidenceArtifact = New-TestRepository 'audit-missing-evidence-artifact'
    Add-AuditHistory $missingEvidenceArtifact $missingEvidenceArtifact.Base $true $true $false $true | Out-Null
    Assert-Fail 'AUD terminal checkpoint missing evidence.json' $missingEvidenceArtifact 'atomically include signed evidence:[\s\S]*evidence\.json'

    $missingEvidenceAttestation = New-TestRepository 'audit-missing-evidence-attestation'
    Add-AuditHistory $missingEvidenceAttestation $missingEvidenceAttestation.Base $true $true $true $false | Out-Null
    Assert-Fail 'AUD terminal checkpoint missing attestation.json' $missingEvidenceAttestation 'atomically include signed evidence:[\s\S]*attestation\.json'

    $mixedTerminal = New-TestRepository 'audit-mixed-terminal'
    Add-AuditHistory $mixedTerminal $mixedTerminal.Base $true $true $true $true $true | Out-Null
    Assert-Fail 'AUD terminal checkpoint mixed with unrelated file' $mixedTerminal 'mixes non-transaction paths'

    $invalidEvidenceAncestry = New-TestRepository 'audit-invalid-evidence-ancestry'
    $baseTree = Get-GitValue $invalidEvidenceAncestry.Path @('rev-parse', "$($invalidEvidenceAncestry.Base)^{tree}")
    $siblingEvidenceRevision = Get-GitValue $invalidEvidenceAncestry.Path @('commit-tree', $baseTree, '-p', $invalidEvidenceAncestry.Base, '-m', 'sibling evidence revision')
    Add-AuditHistory $invalidEvidenceAncestry $siblingEvidenceRevision | Out-Null
    Assert-Fail 'AUD evidence revision that is not an ancestor before open' $invalidEvidenceAncestry 'AUD evidence/open/terminal ancestry is invalid'

    $existingOpenAudit = New-TestRepository 'audit-existing-open-valid'
    Add-ExistingOpenAuditTransition $existingOpenAudit
    Assert-Pass 'existing open AUD terminal transition is validated' $existingOpenAudit

    $existingOpenMissingIndex = New-TestRepository 'audit-existing-open-missing-index'
    Add-ExistingOpenAuditTransition $existingOpenMissingIndex $false
    Assert-Fail 'existing open AUD terminal transition missing index' $existingOpenMissingIndex 'atomically include record and primary index'

    $existingOpenMissingAttestation = New-TestRepository 'audit-existing-open-missing-attestation'
    Add-ExistingOpenAuditTransition $existingOpenMissingAttestation $true $false
    Assert-Fail 'existing open AUD terminal transition missing evidence attestation' $existingOpenMissingAttestation 'atomically include signed evidence:[\s\S]*attestation\.json'

    $existingOpenMixedTerminal = New-TestRepository 'audit-existing-open-mixed-terminal'
    Add-ExistingOpenAuditTransition $existingOpenMixedTerminal $true $true $true
    Assert-Fail 'existing open AUD terminal transition mixed subject path' $existingOpenMixedTerminal 'mixes non-transaction paths'

    $existingOpenTamperedRuntime = New-TestRepository 'audit-existing-open-tampered-runtime'
    Add-ExistingOpenAuditTransition $existingOpenTamperedRuntime $true $true $false $true
    Assert-Fail 'existing open AUD runtime attestation blob changed after open' $existingOpenTamperedRuntime 'Runtime context attestation blob must remain immutable'

    $attestationOnlyTamper = New-TestRepository 'audit-attestation-only-tamper'
    Add-AttestationOnlyTamper $attestationOnlyTamper
    Assert-Fail 'runtime attestation changed without touching its record' $attestationOnlyTamper 'Runtime context attestation blob must remain immutable after HistoryBase'

    $validRemediation = New-TestRepository 'rem-valid-subject'
    Add-SubjectGovernanceHistory $validRemediation 'REM' $false | Out-Null
    Assert-Pass 'valid REM open/result/terminal pure-subject chain' $validRemediation

    $invalidRemediation = New-TestRepository 'rem-impure-subject'
    Add-SubjectGovernanceHistory $invalidRemediation 'REM' $true | Out-Null
    Assert-Fail 'REM result revision containing governance paths' $invalidRemediation 'REM result_revision must be a pure subject commit'

    $partialRemediation = New-TestRepository 'rem-partial-valid'
    Add-SubjectGovernanceHistory $partialRemediation 'REM' $false 'partial' | Out-Null
    Assert-Pass 'partial REM uses a distinct result and terminal transaction' $partialRemediation

    $blockedRemediation = New-TestRepository 'rem-blocked-valid'
    Add-SubjectGovernanceHistory $blockedRemediation 'REM' $false 'blocked' | Out-Null
    Assert-Pass 'blocked REM uses a standalone terminal transaction' $blockedRemediation

    $mixedBlockedRemediation = New-TestRepository 'rem-blocked-mixed-terminal'
    Add-SubjectGovernanceHistory $mixedBlockedRemediation 'REM' $false 'blocked' $true | Out-Null
    Assert-Fail 'blocked REM terminal transaction mixed subject path' $mixedBlockedRemediation 'mixes non-transaction paths'

    $validImplementation = New-TestRepository 'imp-valid-subject'
    Add-SubjectGovernanceHistory $validImplementation 'IMP' $false | Out-Null
    Assert-Pass 'valid IMP open/result/terminal pure-subject chain' $validImplementation

    $invalidImplementation = New-TestRepository 'imp-impure-subject'
    Add-SubjectGovernanceHistory $invalidImplementation 'IMP' $true | Out-Null
    Assert-Fail 'IMP result revision containing governance paths' $invalidImplementation 'IMP result_revision must be a pure subject commit'

    $partialImplementation = New-TestRepository 'imp-partial-valid'
    Add-SubjectGovernanceHistory $partialImplementation 'IMP' $false 'partial' | Out-Null
    Assert-Pass 'partial IMP uses a distinct result and terminal transaction' $partialImplementation

    $blockedImplementation = New-TestRepository 'imp-blocked-valid'
    Add-SubjectGovernanceHistory $blockedImplementation 'IMP' $false 'blocked' | Out-Null
    Assert-Pass 'blocked IMP uses a standalone terminal transaction' $blockedImplementation

    $existingOpenRemediation = New-TestRepository 'rem-existing-open-valid'
    Add-ExistingOpenSubjectTransition $existingOpenRemediation 'REM'
    Assert-Pass 'existing open REM result and terminal transition is validated' $existingOpenRemediation

    $existingOpenImplementation = New-TestRepository 'imp-existing-open-valid'
    Add-ExistingOpenSubjectTransition $existingOpenImplementation 'IMP'
    Assert-Pass 'existing open IMP result and terminal transition is validated' $existingOpenImplementation

    Write-Output 'Governance history validator tests passed: new and pre-existing open AUD/REM/IMP transitions, signed AUD terminal paths, and pure-subject result ancestry fail closed.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot -PathType Container) {
        $resolvedFixtureRoot = (Resolve-Path $fixtureRoot).Path
        $resolvedFixtureBase = (Resolve-Path $fixtureBase).Path
        $allowedPrefix = $resolvedFixtureBase.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar + '.validate-governance-history-'
        if (-not $resolvedFixtureRoot.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected governance history fixture path: $resolvedFixtureRoot"
        }
        for ($attempt = 1; $attempt -le 5; $attempt++) {
            try {
                Remove-Item -LiteralPath $resolvedFixtureRoot -Recurse -Force -ErrorAction Stop
                break
            } catch {
                if ($attempt -eq 5) { throw }
                Start-Sleep -Milliseconds 200
            }
        }
    }
}
