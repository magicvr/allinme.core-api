$ErrorActionPreference = 'Stop'

$fixtureRoot = Join-Path $PSScriptRoot ('.validate-fixture-' + [Guid]::NewGuid().ToString('N'))
$validator = Join-Path $PSScriptRoot 'validate.ps1'

function Invoke-Validator([string]$DocsRoot) {
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $shell = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($null -eq $shell) {
        $shell = Get-Command powershell -ErrorAction Stop
    }
    $output = & $shell.Source -NoProfile -ExecutionPolicy Bypass -File $validator -DocsRoot $DocsRoot 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    return @{
        ExitCode = $exitCode
        Output = ($output | Out-String).Trim()
    }
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot | Out-Null
    $frontmatter = @'
---
status: active
owner: validator-test
last_updated: 2026-07-12
applies_to: validator fixture
---
'@
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'target.md') -Value ($frontmatter + "`n# Target") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'valid.md') -Value ($frontmatter + "`n[Target](./target.md#heading-is-not-validated)") -Encoding UTF8

    $plansRoot = Join-Path $fixtureRoot 'plans'
    New-Item -ItemType Directory -Path $plansRoot | Out-Null
    $planFrontmatter = @'
---
status: active
plan_id: PLN-0001
owner: validator-test
created: 2026-07-14
last_updated: 2026-07-14
applies_to: validator fixture
---
'@
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture.md') -Value ($planFrontmatter + "`n# Plan") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture-checklist.md') -Value ($planFrontmatter + "`n# Checklist") -Encoding UTF8

    $phaseFiveFrontmatter = $planFrontmatter.Replace('PLN-0001', 'PLN-0005')
    $phaseFivePlanPath = Join-Path $plansRoot 'PLN-0005-phase-05-attachment-lifecycle.md'
    $phaseFiveChecklistPath = Join-Path $plansRoot 'PLN-0005-phase-05-attachment-lifecycle-checklist.md'
    $phaseFivePlan = @'
# Phase Five

<!-- phase5-p0-deployment-evidence-contract
p0_artifact_kinds: contract-fixture,disposable-spike
live_evidence_gates: 5A-D-2,5B-4
forbidden_p0_live_evidence: release-binary,supervisor-run,cleanup-schedule-run,watchdog-recovery-run,enospc-run,live-profile-run
-->

| Work package | P0 items | owner / reviewer | Inputs | Timebox | Evidence |
|---|---|---|---|---:|---|
| WP-Facts | P0-1 | owner / reviewer | plan revision | 1 day | facts |
| WP-Schema-Recovery | P0-2 | owner / reviewer | WP-Facts | 1 day | schema |
| WP-HTTP-Order | P0-3 | owner / reviewer | WP-Facts | 1 day | http |
| WP-Lock | P0-4 | owner / reviewer | WP-Facts | 1 day | lock |
| WP-Baseline-Evidence | P0-14, P0-23, P0-24, P0-25 | owner / reviewer | WP-Facts; plan revision | 3 days | validator |
| WP-Files | P0-15 | owner / reviewer | WP-Lock | 1 day | files |
| WP-Runtime | P0-17 | owner / reviewer | WP-Lock | 1 day | runtime |
| WP-Release | P0-16 | owner / reviewer | WP-Facts; WP-Schema-Recovery; WP-HTTP-Order; WP-Lock; WP-Baseline-Evidence; WP-Files; WP-Runtime | 1 day | release |

P0 dependency DAG: WP-Facts precedes Schema-Recovery, HTTP-Order, Lock, and Baseline-Evidence.
'@
    $phaseFiveChecklist = @'
# Phase Five Checklist

- [ ] P0-1. Fixture.
- [ ] P0-2. Fixture.
- [ ] P0-3. Fixture.
- [ ] P0-4. Fixture.
- [ ] P0-5. Fixture.
- [ ] P0-6. Fixture.
- [ ] P0-7. Fixture.
- [ ] P0-8. Fixture.
- [ ] P0-9. Fixture.
- [ ] P0-10. Fixture.
- [ ] P0-11. Fixture.
- [ ] P0-12. Fixture.
- [ ] P0-13. Fixture.
- [ ] P0-14. Fixture.
- [ ] P0-15. Fixture.
- [ ] P0-16. Fixture.
- [ ] P0-17. Fixture.
- [ ] P0-18. Fixture.
- [ ] P0-19. Fixture.
- [ ] P0-20. Fixture.
- [ ] P0-21. Reject WP-Baseline-Evidence without WP-Facts.
- [ ] P0-22. Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4.
- [ ] P0-23. P0-22 uses `artifactKind=contract-fixture`; live evidence belongs to 5A-D and 5B.
- [ ] P0-24. Fixture.
- [ ] P0-25. Fixture.
'@
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $contradictoryDagPlan = $phaseFivePlan + "`nWP-Baseline-Evidence may run before WP-Facts and does not depend on it."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $contradictoryDagPlan) -Encoding UTF8
    $contradictoryDagResult = Invoke-Validator $fixtureRoot
    if ($contradictoryDagResult.ExitCode -eq 0 -or $contradictoryDagResult.Output -notmatch 'dependency ordering contradicts|denies a tracked edge') {
        throw "validator did not reject an additive phase-five DAG contradiction: $($contradictoryDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $contradictoryProfileChecklist = $phaseFiveChecklist.Replace('- [ ] P0-20. Fixture.', '- [ ] P0-20. P0 completion requires live supervisor, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $contradictoryProfileChecklist) -Encoding UTF8
    $contradictoryProfileResult = Invoke-Validator $fixtureRoot
    if ($contradictoryProfileResult.ExitCode -eq 0 -or $contradictoryProfileResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator did not reject an additive phase-five P0 deployment gate: $($contradictoryProfileResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $unexpectedP0Checklist = $phaseFiveChecklist + "`n- [ ] P0-26. Require live supervisor, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile before P0 completes."
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $unexpectedP0Checklist) -Encoding UTF8
    $unexpectedP0Result = Invoke-Validator $fixtureRoot
    if ($unexpectedP0Result.ExitCode -eq 0 -or $unexpectedP0Result.Output -notmatch 'unexpected P0 item: P0-26') {
        throw "validator did not reject an additive phase-five P0 item: $($unexpectedP0Result.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $contradictoryProfilePlan = $phaseFivePlan + "`nP0 completion requires live supervisor, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $contradictoryProfilePlan) -Encoding UTF8
    $contradictoryProfilePlanResult = Invoke-Validator $fixtureRoot
    if ($contradictoryProfilePlanResult.ExitCode -eq 0 -or $contradictoryProfilePlanResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator did not reject additive phase-five P0 deployment prose: $($contradictoryProfilePlanResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $auditRecordsRoot = Join-Path $fixtureRoot 'audits\records'
    New-Item -ItemType Directory -Path $auditRecordsRoot -Force | Out-Null
    $auditFrontmatter = @'
---
status: open
audit_id: AUD-0001
auditor: validator-test
audit_type: targeted
scope: feature:validator
subject: validator fixture
baseline: git:0000000; worktree:clean
started_at: 2026-07-14T00:00:00+08:00
completed_at: pending
last_updated: 2026-07-14
---
'@
    $auditRecordName = 'AUD-0001-20260714-validator-feature-validator-fixture.md'
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $auditRecordName) -Value ($auditFrontmatter + "`n# Audit") -Encoding UTF8
    $auditIndexPath = Join-Path $fixtureRoot 'audits\README.md'
    $auditIndexContent = "# Audits`n`n- [AUD-0001](./records/$auditRecordName): ``status=open``; ``remediation=pending``; fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $planAuditFrontmatter = @'
---
status: open
audit_schema: plan-audit/v2
audit_id: AUD-0004
auditor: validator-test
audit_type: targeted
scope: plan:PLN-0001
subject: validator plan fixture
baseline: git:0000000; worktree:clean
started_at: 2026-07-14T00:30:00+08:00
completed_at: pending
last_updated: 2026-07-14
related_audits: none
related_remediations: none
supersedes: none
related_plans: PLN-0001
---
'@
    $planAuditRecordName = 'AUD-0004-20260714-validator-plan-validator-fixture.md'
    $planAuditMatrix = @'
# Plan Audit

<!-- plan-checklist-audit: PLN-0001 -->
### PLN-0001 Plan/Checklist 审计

- Plan: [Plan](../../plans/PLN-0001-validator-fixture.md)
- Checklist: [Checklist](../../plans/PLN-0001-validator-fixture-checklist.md)

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| PAIRING | IDs, frontmatter, and links checked | pass | none |
| PLAN_TO_CHECKLIST | mandatory obligations mapped to checklist items | pass | none |
| CHECKLIST_TO_PLAN | no unsupported contract additions | pass | none |
| CHECKED_EVIDENCE | no checked items in unstarted fixture | not-applicable | none |
| GATE_COMPLETENESS | validation and release gates inspected | pass | none |
| ARCHIVE_CLOSURE | completion and archive conditions inspected | pass | none |
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $planAuditRecordName) -Value ($planAuditFrontmatter + "`n" + $planAuditMatrix) -Encoding UTF8

    $missingDagEdgePlan = $phaseFivePlan.Replace('WP-Facts; plan revision', 'plan revision')
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $missingDagEdgePlan) -Encoding UTF8
    $missingDagEdgeResult = Invoke-Validator $fixtureRoot
    if ($missingDagEdgeResult.ExitCode -eq 0 -or $missingDagEdgeResult.Output -notmatch 'tracked contract for WP-Baseline-Evidence') {
        throw "validator did not reject a missing phase-five DAG edge: $($missingDagEdgeResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $liveProfileChecklist = $phaseFiveChecklist.Replace('Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4.', 'Select and live-test one deployment profile during P0.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $liveProfileChecklist) -Encoding UTF8
    $liveProfileResult = Invoke-Validator $fixtureRoot
    if ($liveProfileResult.ExitCode -eq 0 -or $liveProfileResult.Output -notmatch 'stop at a deployment contract fixture') {
        throw "validator did not reject a live phase-five P0 profile gate: $($liveProfileResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8
    $auditIndexContent += "`n- [AUD-0004](./records/$planAuditRecordName): ``status=open``; ``remediation=pending``; fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $remediationRecordsRoot = Join-Path $fixtureRoot 'remediations\records'
    New-Item -ItemType Directory -Path $remediationRecordsRoot -Force | Out-Null
    $remediationFrontmatter = @'
---
status: completed
remediation_id: REM-0001
implementer: validator-test
scope: audit:AUD-0001
source_audits: AUD-0001
source_findings: AUD-0001-F001
baseline: git:0000000; worktree:clean
started_at: 2026-07-14T01:00:00+08:00
completed_at: 2026-07-14T02:00:00+08:00
last_updated: 2026-07-14
related_plans: none
---
'@
    $remediationRecordName = 'REM-0001-20260714-validator-audit-validator-fixture.md'
    Set-Content -LiteralPath (Join-Path $remediationRecordsRoot $remediationRecordName) -Value ($remediationFrontmatter + "`n# Remediation") -Encoding UTF8
    $remediationIndexPath = Join-Path $fixtureRoot 'remediations\README.md'
    Set-Content -LiteralPath $remediationIndexPath -Value ("# Remediations`n`n- [REM-0001](./records/$remediationRecordName): ``status=completed``; ``verification=pending``; fixture.") -Encoding UTF8

    $validResult = Invoke-Validator $fixtureRoot
    if ($validResult.ExitCode -ne 0) {
        throw "validator rejected valid fixture: $($validResult.Output)"
    }

    $invalidPlanAudit = $planAuditMatrix.Replace('| CHECKLIST_TO_PLAN | no unsupported contract additions | pass | none |', '')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $planAuditRecordName) -Value ($planAuditFrontmatter + "`n" + $invalidPlanAudit) -Encoding UTF8
    $missingChecklistMatrixResult = Invoke-Validator $fixtureRoot
    if ($missingChecklistMatrixResult.ExitCode -eq 0 -or $missingChecklistMatrixResult.Output -notmatch 'CHECKLIST_TO_PLAN') {
        throw "validator did not reject a plan audit without the complete checklist matrix: $($missingChecklistMatrixResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $planAuditRecordName) -Value ($planAuditFrontmatter + "`n" + $planAuditMatrix) -Encoding UTF8

    Set-Content -LiteralPath $auditIndexPath -Value '# Audits' -Encoding UTF8
    $missingAuditIndexResult = Invoke-Validator $fixtureRoot
    if ($missingAuditIndexResult.ExitCode -eq 0 -or $missingAuditIndexResult.Output -notmatch 'Audit record must be indexed exactly once') {
        throw "validator did not reject an unindexed audit: $($missingAuditIndexResult.Output)"
    }
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    Set-Content -LiteralPath (Join-Path $fixtureRoot 'invalid.md') -Value ($frontmatter + "`n[Missing](./does-not-exist.md#missing)") -Encoding UTF8
    $invalidResult = Invoke-Validator $fixtureRoot
    if ($invalidResult.ExitCode -eq 0) {
        throw 'validator accepted invalid fixture'
    }
    if ($invalidResult.Output -notmatch 'Missing relative link target') {
        throw "validator failed for an unexpected reason: $($invalidResult.Output)"
    }
    Remove-Item -LiteralPath (Join-Path $fixtureRoot 'invalid.md')

    $orphanFrontmatter = $planFrontmatter.Replace('PLN-0001', 'PLN-0002')
    $orphanPath = Join-Path $plansRoot 'PLN-0002-orphan-plan.md'
    Set-Content -LiteralPath $orphanPath -Value ($orphanFrontmatter + "`n# Orphan") -Encoding UTF8
    $orphanResult = Invoke-Validator $fixtureRoot
    if ($orphanResult.ExitCode -eq 0 -or $orphanResult.Output -notmatch 'Plan/checklist pair') {
        throw "validator did not reject an orphan plan: $($orphanResult.Output)"
    }
    Remove-Item -LiteralPath $orphanPath

    $closedAudit = $auditFrontmatter.Replace('status: open', 'status: closed')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $auditRecordName) -Value ($closedAudit + "`n# Audit") -Encoding UTF8
    $closedResult = Invoke-Validator $fixtureRoot
    if ($closedResult.ExitCode -eq 0 -or $closedResult.Output -notmatch 'completed_at') {
        throw "validator did not reject a closed audit without completed_at: $($closedResult.Output)"
    }

    $global:LASTEXITCODE = 0
    Write-Output 'Validator self-test passed: valid governance accepted; phase-five DAG/profile removals and additive contradictions, incomplete checklist matrices, unindexed audits, missing links, orphan plans, and incomplete closed audits rejected.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot) {
        $resolvedFixtureRoot = (Resolve-Path $fixtureRoot).Path
        $allowedPrefix = (Resolve-Path $PSScriptRoot).Path + [System.IO.Path]::DirectorySeparatorChar + '.validate-fixture-'
        if (-not $resolvedFixtureRoot.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected validator fixture path: $resolvedFixtureRoot"
        }
        for ($attempt = 1; $attempt -le 5; $attempt++) {
            try {
                Remove-Item -LiteralPath $resolvedFixtureRoot -Recurse -Force -ErrorAction Stop
                break
            } catch {
                if ($attempt -eq 5) {
                    throw
                }
                Start-Sleep -Milliseconds 200
            }
        }
    }
}
