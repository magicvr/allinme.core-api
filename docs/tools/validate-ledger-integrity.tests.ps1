$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$sourceValidator = Join-Path $PSScriptRoot 'validate.ps1'
$fixtureRoot = Join-Path (Join-Path $repoRoot '.tmp') ('.validate-ledger-' + [Guid]::NewGuid().ToString('N'))
$gitCommand = @(Get-Command git -CommandType Application -ErrorAction Stop) | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath([string]$gitCommand.Source)
$validationShellCommand = @(Get-Command powershell.exe -CommandType Application -ErrorAction Stop) | Select-Object -First 1
$validationShellExecutable = [IO.Path]::GetFullPath([string]$validationShellCommand.Source)
$repoRootPrefix = $repoRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
foreach ($executable in @($gitExecutable, $validationShellExecutable)) {
    if ([string]::Equals($executable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
        $executable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to execute a repository-controlled binary: $executable"
    }
}

function Set-Utf8File([string]$Path, [string]$Content) {
    $parent = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $parent)) { New-Item -ItemType Directory -Path $parent -Force | Out-Null }
    $normalized = $Content.Replace("`r`n", "`n").Replace("`r", "`n").Replace("`n", [Environment]::NewLine)
    [IO.File]::WriteAllText($Path, $normalized, [Text.UTF8Encoding]::new($false))
}

function Invoke-Git([string[]]$Arguments) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $gitExecutable -C $fixtureRoot @Arguments 2>&1)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    if ($exitCode -ne 0) { throw "git $($Arguments -join ' ') failed: $($output -join [Environment]::NewLine)" }
    return $output
}

function Invoke-FixtureValidator {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $scriptPath = (Join-Path $fixtureRoot 'docs\tools\validate.ps1')
    $command = "& '$($scriptPath.Replace("'", "''"))'; exit `$LASTEXITCODE"
    $output = @(& $validationShellExecutable -NoProfile -ExecutionPolicy Bypass -Command $command 2>&1)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return @{ ExitCode = $exitCode; Output = ($output | Out-String).Trim() }
}

function Assert-Pass([string]$Label) {
    $result = Invoke-FixtureValidator
    if ($result.ExitCode -ne 0) { throw "$Label unexpectedly failed: $($result.Output)" }
}

function Assert-Fail([string]$Label, [string]$Pattern) {
    $result = Invoke-FixtureValidator
    if ($result.ExitCode -eq 0 -or $result.Output -notmatch $Pattern) {
        throw "$Label was not rejected with /$Pattern/: $($result.Output)"
    }
}

function Get-Plan([string]$Id, [string]$Subject, [string]$Status = 'active') {
    return @"
---
status: $Status
plan_id: $Id
owner: validator-test
created: 2026-07-15
last_updated: 2026-07-15
applies_to: ledger integrity fixture
---

# $Id $Subject
"@
}

function Get-PlanAudit([string]$Baseline, [string]$EvidenceRevision, [string]$StartedAt, [string]$CompletedAt, [string]$RunId = '11111111-1111-4111-8111-111111111119') {
    return @"
---
status: closed
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: plan-audit/v2
audit_id: AUD-0001
auditor: validator-test
execution_context_id: 11111111-1111-4111-8111-111111111111
runtime_context_ref: task://plan-audit
audit_type: targeted
scope: plan:PLN-0001
subject: fixture
baseline: git:$Baseline; worktree:clean
evidence_revision: git:$EvidenceRevision; worktree:clean
evidence_worktree_revision: git:$EvidenceRevision
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["git","rev-parse","HEAD"]
evidence_run_id: $RunId
evidence_artifact: docs/evidence/runs/$RunId/evidence.json
audited_peer_plans: PLN-0001
audited_subject_paths: docs/plans/PLN-0001-fixture.md, docs/plans/PLN-0001-fixture-checklist.md
started_at: $StartedAt
completed_at: $CompletedAt
last_updated: 2026-07-15
related_audits: none
related_remediations: none
supersedes: none
superseded_by: none
supersession_reason: none
related_plans: PLN-0001
---

# Plan audit

- Plan: [fixture](../../plans/PLN-0001-fixture.md)
- Checklist: [fixture checklist](../../plans/PLN-0001-fixture-checklist.md)

<!-- plan-checklist-audit: PLN-0001 -->

- Plan: [fixture](../../plans/PLN-0001-fixture.md)
- Checklist: [fixture checklist](../../plans/PLN-0001-fixture-checklist.md)

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| PAIRING | plan/checklist pair | pass | none |
| PLAN_TO_CHECKLIST | traced obligations | pass | none |
| CHECKLIST_TO_PLAN | traced checklist | pass | none |
| CHECKED_EVIDENCE | no completed items | not-applicable | none |
| GATE_COMPLETENESS | gates mapped | pass | none |
| ARCHIVE_CLOSURE | active plan | pass | none |

## 验证结果

``git rev-parse HEAD`` result=pass
"@
}

function Get-PlanAcceptance([string]$Baseline, [string]$EvidenceRevision, [string]$RunId) {
    return @"
---
status: closed
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: plan-acceptance/v2
audit_id: AUD-0002
auditor: validator-acceptance
execution_context_id: 22222222-2222-4222-8222-222222222222
runtime_context_ref: task://plan-acceptance
runtime_context_attestation: docs/evidence/runtime-attestations/22222222-2222-4222-8222-222222222222.json
source_context_ids: 11111111-1111-4111-8111-111111111111
source_context_refs: task://plan-audit
audit_type: acceptance
acceptance_type: plan-readiness
acceptance_verdict: ready
scope: plan:PLN-0001
subject: fixture readiness
plan_status_at_acceptance: active
independence_basis: separate-context
baseline: git:$Baseline; worktree:clean
evidence_revision: git:$EvidenceRevision; worktree:clean
evidence_worktree_revision: git:$EvidenceRevision
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["git","rev-parse","HEAD"]
evidence_run_id: $RunId
evidence_artifact: docs/evidence/runs/$RunId/evidence.json
evidence_attestation: docs/evidence/runs/$RunId/attestation.json
started_at: 2026-07-15T12:00:00+08:00
completed_at: 2026-07-15T12:01:00+08:00
last_updated: 2026-07-15
related_audits: AUD-0001
related_remediations: none
supersedes: none
superseded_by: none
supersession_reason: none
related_plans: PLN-0001
---

# Plan readiness

<!-- plan-acceptance-audit: PLN-0001 -->

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| READY_IDENTITY | identity checked | pass | none |
| READY_SCOPE | scope checked | pass | none |
| READY_FACTS | facts checked | pass | none |
| READY_DEPENDENCIES | dependencies checked | pass | none |
| READY_DESIGN | design checked | pass | none |
| READY_EVIDENCE | docs/evidence/runs/$RunId/evidence.json | pass | none |
| READY_GATES | gates checked | pass | none |
| PLAN_AUDIT_CHAIN_CLEAN | source chain checked | pass | none |

## 验证结果

``git rev-parse HEAD`` result=pass; artifact ``docs/evidence/runs/$RunId/evidence.json``
"@
}

function Get-EvidenceArtifact([string]$RunId, [string]$Revision, [string]$ExitCode = '0') {
    $tree = (Invoke-Git @('rev-parse', "$Revision^{tree}") | Select-Object -First 1).Trim()
    return @"
{
  "schema": "revision-evidence/v1",
  "evidence_run_id": "$RunId",
  "evidence_revision": "$Revision",
  "evidence_tree": "$tree",
  "evidence_worktree": "detached",
  "argv": ["git", "rev-parse", "HEAD"],
  "exit_code": $ExitCode,
  "isolation": {
    "engine": "docker",
    "image": "docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a",
    "image_id": "sha256:dd2d88d0c7034f9e48bb74156ea562e66d3064971aed54ccbb23554637580f1c",
    "approved_image": true,
    "entrypoint": "/usr/bin/env",
    "network": "none",
    "repository_mount": "read-only",
    "host_repository_mounted": false,
    "snapshot_source": "git-archive-tar+manifest/v1",
    "snapshot_mount": "read-only",
    "snapshot_archive_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "snapshot_manifest_sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "snapshot_manifest_entries": 1,
    "root_filesystem": "read-only",
    "capabilities": "none",
    "no_new_privileges": true,
    "user": "65534:65534",
    "memory_megabytes": 1024,
    "cpus": 1,
    "pids_limit": 256,
    "timeout_seconds": 900,
    "max_output_bytes": 4194304,
    "output_capture": "streaming-bounded",
    "sanitized_environment": true,
    "preflight_passed": true,
    "failure_kind": "none",
    "failure_message_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "workspace": "/tmp/workspace (tmpfs)",
    "workspace_mount_options": "rw,exec,nosuid,nodev,size=512m"
  },
  "output": {
    "stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "combined_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "stdout_bytes": 0,
    "stderr_bytes": 0,
    "captured_bytes": 0,
    "observed_bytes_at_least": 0,
    "truncated": false,
    "capture_complete": true
  },
  "clean_status": {
    "host_tracked_clean_before_run": true,
    "host_tracked_clean_after_run": true,
    "host_tracked_state_unchanged": true,
    "subject_tracked_clean_before_run": true,
    "subject_tracked_clean_after_run": true,
    "subject_workspace_discarded": true
  },
  "tracked_worktree_clean_after_run": true,
  "started_at": "2026-07-15T04:00:00Z",
  "completed_at": "2026-07-15T04:00:01Z"
}
"@
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot -Force | Out-Null
    Invoke-Git @('init', '-q') | Out-Null
    Invoke-Git @('config', 'user.name', 'Validator Test') | Out-Null
    Invoke-Git @('config', 'user.email', 'validator@example.invalid') | Out-Null
    Invoke-Git @('config', 'core.autocrlf', 'true') | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $fixtureRoot 'docs\tools') -Force | Out-Null
    Copy-Item -Path (Join-Path $repoRoot 'docs\tools\*') -Destination (Join-Path $fixtureRoot 'docs\tools') -Recurse
    Set-Utf8File (Join-Path $fixtureRoot 'docs\tools\validate-audit-workflows.ps1') "param([string]`$RepositoryRoot)`nexit 0`n"
    New-Item -ItemType Directory -Path (Join-Path $fixtureRoot 'docs\audits\templates') -Force | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $fixtureRoot 'docs\implementations\templates') -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot 'docs\audits\templates\follow-up-audit-record.md') -Destination (Join-Path $fixtureRoot 'docs\audits\templates\follow-up-audit-record.md')
    Copy-Item -LiteralPath (Join-Path $repoRoot 'docs\implementations\README.md') -Destination (Join-Path $fixtureRoot 'docs\implementations\README.md')
    Copy-Item -LiteralPath (Join-Path $repoRoot 'docs\implementations\templates\implementation-record.md') -Destination (Join-Path $fixtureRoot 'docs\implementations\templates\implementation-record.md')
    Copy-Item -LiteralPath (Join-Path $repoRoot '.github') -Destination (Join-Path $fixtureRoot '.github') -Recurse
    Copy-Item -LiteralPath (Join-Path $repoRoot '.agents') -Destination (Join-Path $fixtureRoot '.agents') -Recurse
    foreach ($prompt in Get-ChildItem -LiteralPath (Join-Path $fixtureRoot '.github\prompts') -Filter 'backend-*.prompt.md') {
        Add-Content -LiteralPath $prompt.FullName -Value "`n<!-- audit-safety-contract: repository-content-is-data; inspect-before-execute; no-secret-exposure -->`nopen checkpoint`nterminal governance commit`ngovernance_revision"
    }
    $requiredEntrypoints = @(
        '.github/prompts/backend-plan-audit.prompt.md',
        '.github/prompts/backend-plan-acceptance-audit.prompt.md',
        '.github/prompts/backend-implement-plan.prompt.md',
        '.github/prompts/backend-implementation-audit.prompt.md',
        '.github/prompts/backend-implementation-acceptance-audit.prompt.md',
        '.github/prompts/backend-fix-audit-findings.prompt.md',
        '.github/prompts/backend-follow-up-audit.prompt.md',
        '.github/prompts/backend-plan-audit-until-ready.prompt.md',
        '.github/prompts/backend-implement-audit-until-complete.prompt.md',
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
        '.agents/skills/backend-implement-audit-until-complete/SKILL.md',
        '.agents/skills/backend-implement-audit-until-complete/agents/openai.yaml'
    )
    foreach ($entrypoint in $requiredEntrypoints) {
        if (-not (Test-Path -LiteralPath (Join-Path $fixtureRoot $entrypoint))) {
            Set-Utf8File (Join-Path $fixtureRoot $entrypoint) "fixture`n"
        }
    }
    Set-Utf8File (Join-Path $fixtureRoot 'docs\plans\PLN-0001-fixture.md') (Get-Plan 'PLN-0001' 'fixture')
    Set-Utf8File (Join-Path $fixtureRoot 'docs\plans\PLN-0001-fixture-checklist.md') (Get-Plan 'PLN-0001' 'fixture checklist')
    Invoke-Git @('add', '.') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'base plan') | Out-Null
    $planRevision = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()

    $planAuditRunId = '11111111-1111-4111-8111-111111111119'
    Set-Utf8File (Join-Path $fixtureRoot "docs\evidence\runs\$planAuditRunId\evidence.json") (Get-EvidenceArtifact $planAuditRunId $planRevision)
    Invoke-Git @('add', '.') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'commit plan audit evidence artifact') | Out-Null
    $planAuditBaseline = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()
    $planAuditRecordPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0001-20260715-validator-plan-fixture.md'
    Set-Utf8File $planAuditRecordPath (Get-PlanAudit $planAuditBaseline $planRevision '2026-07-15T23:00:00+08:00' '2026-07-15T23:01:00+08:00' $planAuditRunId)
    Set-Utf8File (Join-Path $fixtureRoot 'docs\audits\README.md') "# Audits`n`n- [AUD-0001](./records/AUD-0001-20260715-validator-plan-fixture.md): status=closed; remediation=none`n"
    Invoke-Git @('add', '.') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'historical plan audit') | Out-Null
    $auditRevision = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()

    Set-Utf8File (Join-Path $fixtureRoot 'docs\plans\PLN-0002-later-peer.md') (Get-Plan 'PLN-0002' 'later peer')
    Set-Utf8File (Join-Path $fixtureRoot 'docs\plans\PLN-0002-later-peer-checklist.md') (Get-Plan 'PLN-0002' 'later peer checklist')
    Invoke-Git @('add', '.') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'add later active peer') | Out-Null
    $currentRevision = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()
    if ((Get-Content -Raw -Encoding UTF8 $planAuditRecordPath) -notmatch '\]\(\.\./\.\./plans/PLN-0001-fixture\.md\)') {
        throw 'fixture plan audit lost its plan link'
    }
    Assert-Pass 'historical plan audit peer snapshot'

    $repositoryGitPath = Join-Path $fixtureRoot 'git.cmd'
    Set-Utf8File $repositoryGitPath "@echo off`r`nexit /b 99`r`n"
    $savedPath = $env:PATH
    try {
        $env:PATH = "$fixtureRoot$([IO.Path]::PathSeparator)$savedPath"
        Assert-Fail 'repository-controlled Git executable' 'Refusing to execute a repository-controlled Git binary'
    } finally {
        $env:PATH = $savedPath
        Remove-Item -LiteralPath $repositoryGitPath -Force
    }

    $repositoryShellPath = Join-Path $fixtureRoot 'pwsh.cmd'
    Set-Utf8File $repositoryShellPath "@echo off`r`nexit /b 99`r`n"
    $savedPath = $env:PATH
    try {
        $env:PATH = "$fixtureRoot$([IO.Path]::PathSeparator)$savedPath"
        Assert-Fail 'repository-controlled validation shell' 'Refusing to execute a repository-controlled validation shell'
    } finally {
        $env:PATH = $savedPath
        Remove-Item -LiteralPath $repositoryShellPath -Force
    }

    $runId = '33333333-3333-4333-8333-333333333333'
    $evidenceArtifactPath = Join-Path $fixtureRoot "docs\evidence\runs\$runId\evidence.json"
    Set-Utf8File $evidenceArtifactPath (Get-EvidenceArtifact $runId $currentRevision)
    Invoke-Git @('add', '.') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'commit evidence artifact') | Out-Null
    $acceptanceBaseline = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()
    $workingAuditBlob = (& $gitExecutable -C $fixtureRoot hash-object -- $planAuditRecordPath | Select-Object -First 1).Trim()
    $baselineAuditBlob = (& $gitExecutable -C $fixtureRoot rev-parse "$acceptanceBaseline`:docs/audits/records/AUD-0001-20260715-validator-plan-fixture.md" | Select-Object -First 1).Trim()
    if ($workingAuditBlob -ne $baselineAuditBlob) { throw "fixture audit blob drift: working=$workingAuditBlob baseline=$baselineAuditBlob" }
    $artifact = Get-Content -Raw -Encoding UTF8 $evidenceArtifactPath | ConvertFrom-Json
    $expectedEvidenceTree = (& $gitExecutable -C $fixtureRoot rev-parse "$currentRevision^{tree}" | Select-Object -First 1).Trim()
    if ($artifact.evidence_tree -ne $expectedEvidenceTree) { throw "fixture artifact tree drift: actual=$($artifact.evidence_tree) expected=$expectedEvidenceTree" }
    $acceptanceRecordPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0002-20260715-validator-plan-fixture-readiness.md'
    $validAcceptanceRecord = Get-PlanAcceptance $acceptanceBaseline $currentRevision $runId
    Set-Utf8File $acceptanceRecordPath $validAcceptanceRecord
    $auditIndexPath = Join-Path $fixtureRoot 'docs\audits\README.md'
    $auditIndexContent = (Get-Content -Raw -Encoding UTF8 $auditIndexPath).TrimEnd() + "`n- [AUD-0002](./records/AUD-0002-20260715-validator-plan-fixture-readiness.md): status=closed; remediation=none`n"
    Set-Utf8File $auditIndexPath $auditIndexContent
    Assert-Fail 'readiness peer drift' 'peer snapshot drifted'

    Set-Utf8File $acceptanceRecordPath ($validAcceptanceRecord -replace '(?m)^evidence_argv_json:.*\r?\n', '')
    Assert-Fail 'missing evidence argv declaration' "Missing frontmatter field 'evidence_argv_json'"

    Set-Utf8File $acceptanceRecordPath ($validAcceptanceRecord -replace '(?m)^evidence_argv_json:.*$', 'evidence_argv_json: ["git",]')
    Assert-Fail 'invalid evidence argv JSON' 'Audit evidence_argv_json must be strict JSON'

    Set-Utf8File $acceptanceRecordPath ($validAcceptanceRecord -replace '(?m)^evidence_argv_json:.*$', 'evidence_argv_json: ["git","HEAD","rev-parse"]')
    Assert-Fail 'mismatched evidence argv declaration' 'must exactly match the ordered evidence artifact argv'

    Set-Utf8File $acceptanceRecordPath $validAcceptanceRecord

    $validAcceptanceArtifact = Get-EvidenceArtifact $runId $currentRevision
    $governanceArtifact = $validAcceptanceArtifact | ConvertFrom-Json
    $governanceArtifact.argv = @('powershell.exe', '-File', 'docs/tools/validate-governance-history.ps1')
    Set-Utf8File $evidenceArtifactPath ($governanceArtifact | ConvertTo-Json -Depth 8)
    Set-Utf8File $acceptanceRecordPath ($validAcceptanceRecord -replace '(?m)^evidence_argv_json:.*$', 'evidence_argv_json: ["powershell.exe","-File","docs/tools/validate-governance-history.ps1"]')
    Assert-Fail 'governance validator argv as subject evidence' 'must contain a subject-specific command'

    Set-Utf8File $acceptanceRecordPath $validAcceptanceRecord
    Set-Utf8File $evidenceArtifactPath $validAcceptanceArtifact

    $validAcceptanceArtifact = Get-EvidenceArtifact $runId $currentRevision
    $forgedArtifact = $validAcceptanceArtifact | ConvertFrom-Json
    $forgedArtifact.isolation.no_new_privileges = 'true'
    Set-Utf8File $evidenceArtifactPath ($forgedArtifact | ConvertTo-Json -Depth 8)
    Assert-Fail 'string boolean isolation claim' 'required isolated execution'

    $forgedArtifact = $validAcceptanceArtifact | ConvertFrom-Json
    $forgedArtifact.isolation.memory_megabytes = '1024'
    Set-Utf8File $evidenceArtifactPath ($forgedArtifact | ConvertTo-Json -Depth 8)
    Assert-Fail 'string resource isolation claim' 'required isolated execution'

    $forgedArtifact = $validAcceptanceArtifact | ConvertFrom-Json
    $forgedArtifact.isolation.preflight_passed = $false
    Set-Utf8File $evidenceArtifactPath ($forgedArtifact | ConvertTo-Json -Depth 8)
    Assert-Fail 'failed evidence preflight' 'required isolated execution'

    $forgedArtifact = $validAcceptanceArtifact | ConvertFrom-Json
    $forgedArtifact.clean_status.subject_tracked_clean_before_run = $false
    Set-Utf8File $evidenceArtifactPath ($forgedArtifact | ConvertTo-Json -Depth 8)
    Assert-Fail 'dirty subject before successful evidence' 'must start and finish with the detached subject tracked-clean'

    Set-Utf8File $evidenceArtifactPath $validAcceptanceArtifact

    (Get-Content -Raw -Encoding UTF8 (Join-Path $fixtureRoot 'docs\audits\README.md')).Replace('remediation=none', 'remediation=accepted-risk') |
        Set-Content -LiteralPath (Join-Path $fixtureRoot 'docs\audits\README.md') -Encoding UTF8
    $auditPath = $planAuditRecordPath
    Add-Content -LiteralPath $auditPath -Value "`n### AUD-0001-F001 - accepted risk`n`n- Severity: high`n- Evidence: evidence`n- Impact: impact`n- Recommendation: recommendation`n- Owner: author-entered-owner`n- Disposition: accepted-risk`n"
    Assert-Fail 'repository-authored accepted risk' 'externally verifiable approval attestation'

    (Get-Content -Raw -Encoding UTF8 (Join-Path $fixtureRoot 'docs\audits\README.md')).Replace('remediation=accepted-risk', 'remediation=none') |
        Set-Content -LiteralPath (Join-Path $fixtureRoot 'docs\audits\README.md') -Encoding UTF8
    Set-Utf8File $auditPath (Get-PlanAudit $planAuditBaseline $planRevision '2026-07-15T23:00:00+08:00' '2026-07-15T23:01:00+08:00' $planAuditRunId)
    Remove-Item -LiteralPath (Join-Path $fixtureRoot "docs\evidence\runs\$runId\evidence.json")
    Assert-Fail 'missing structured evidence artifact' 'Evidence artifact is missing'

    Write-Output 'Ledger integrity validator tests passed: historical peers, readiness drift, exact argv declaration binding, repository-local Git/validation-shell rejection, accepted-risk authorization, and typed structured evidence fail-closed behavior.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot) { Remove-Item -LiteralPath $fixtureRoot -Recurse -Force }
}
