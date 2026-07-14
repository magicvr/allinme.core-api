$ErrorActionPreference = 'Stop'

$fixtureRoot = Join-Path $PSScriptRoot ('.validate-fixture-' + [Guid]::NewGuid().ToString('N'))
$allocatorRoot = Join-Path $PSScriptRoot ('.allocator-fixture-' + [Guid]::NewGuid().ToString('N'))
$validator = Join-Path $PSScriptRoot 'validate.ps1'
$allocator = Join-Path $PSScriptRoot 'reserve-governance-record.ps1'

function Invoke-Validator([string]$DocsRoot) {
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $shell = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($null -eq $shell) {
        $shell = Get-Command powershell -ErrorAction Stop
    }
    $output = & $shell.Source -NoProfile -ExecutionPolicy Bypass -File $validator -DocsRoot $DocsRoot 2>&1
    $exitCode = $LASTEXITCODE
    Start-Sleep -Milliseconds 100
    $ErrorActionPreference = $previousErrorAction
    return @{
        ExitCode = $exitCode
        Output = ($output | Out-String).Trim()
    }
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $allocatorRoot 'docs\audits\records') -Force | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $allocatorRoot 'docs\remediations\records') -Force | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $allocatorRoot 'docs\implementations\records') -Force | Out-Null
    $firstAllocation = & $allocator -Kind AUD -Suffix '20260714-validator-plan-first' -RepositoryRoot $allocatorRoot
    $secondAllocation = & $allocator -Kind AUD -Suffix '20260714-validator-plan-second' -RepositoryRoot $allocatorRoot
    if ($firstAllocation -notmatch '^AUD-0001\s+docs/audits/records/AUD-0001-' -or
        $secondAllocation -notmatch '^AUD-0002\s+docs/audits/records/AUD-0002-') {
        throw "governance allocator did not reserve monotonically increasing IDs: $firstAllocation / $secondAllocation"
    }
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
| WP-Facts | P0-1 | owner / reviewer | plan revision | 1 day | docs/01-architecture.md, docs/05-domain-model.md, docs/03-http-api-target.md, docs/06-implementation-roadmap.md, docs/04-validation.md, plan/checklist diffs |
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

    $missingFactSourcePlan = $phaseFivePlan.Replace('docs/01-architecture.md, ', '')
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $missingFactSourcePlan) -Encoding UTF8
    $missingFactSourceResult = Invoke-Validator $fixtureRoot
    if ($missingFactSourceResult.ExitCode -eq 0 -or $missingFactSourceResult.Output -notmatch 'WP-Facts output is missing required fact source: docs/01-architecture.md') {
        throw "validator did not reject a missing WP-Facts source: $($missingFactSourceResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $unknownFactSourcePlan = $phaseFivePlan.Replace('plan/checklist diffs', 'plan/checklist diffs, docs/99-untracked-fact.md')
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $unknownFactSourcePlan) -Encoding UTF8
    $unknownFactSourceResult = Invoke-Validator $fixtureRoot
    if ($unknownFactSourceResult.ExitCode -eq 0 -or $unknownFactSourceResult.Output -notmatch 'WP-Facts output contains unknown or non-canonical token: docs/99-untracked-fact.md') {
        throw "validator did not reject an unknown WP-Facts source: $($unknownFactSourceResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $duplicateFactSourcePlan = $phaseFivePlan.Replace('docs/04-validation.md, plan/checklist diffs', 'docs/04-validation.md, docs/04-validation.md, plan/checklist diffs')
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $duplicateFactSourcePlan) -Encoding UTF8
    $duplicateFactSourceResult = Invoke-Validator $fixtureRoot
    if ($duplicateFactSourceResult.ExitCode -eq 0 -or $duplicateFactSourceResult.Output -notmatch 'WP-Facts output contains duplicate token: docs/04-validation.md') {
        throw "validator did not reject a duplicate WP-Facts source: $($duplicateFactSourceResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $ambiguousFactSourcePlan = $phaseFivePlan.Replace('docs/04-validation.md, plan/checklist diffs', 'docs/04-validation.md, docs/04-validation, plan/checklist diffs')
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $ambiguousFactSourcePlan) -Encoding UTF8
    $ambiguousFactSourceResult = Invoke-Validator $fixtureRoot
    if ($ambiguousFactSourceResult.ExitCode -eq 0 -or $ambiguousFactSourceResult.Output -notmatch 'WP-Facts output contains unknown or non-canonical token: docs/04-validation') {
        throw "validator did not reject an ambiguous WP-Facts source spelling: $($ambiguousFactSourceResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $negativePronounDagPlan = $phaseFivePlan + "`nWP-Baseline-Evidence can consume WP-Facts metadata but does not depend on it."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $negativePronounDagPlan) -Encoding UTF8
    $negativePronounDagResult = Invoke-Validator $fixtureRoot
    if ($negativePronounDagResult.ExitCode -eq 0 -or $negativePronounDagResult.Output -notmatch 'denies a tracked edge') {
        throw "validator did not reject a pronoun phase-five DAG negation: $($negativePronounDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $conjoinedDagPlan = $phaseFivePlan + "`nWP-Release depends on WP-Facts and WP-Unknown."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $conjoinedDagPlan) -Encoding UTF8
    $conjoinedDagResult = Invoke-Validator $fixtureRoot
    if ($conjoinedDagResult.ExitCode -eq 0 -or $conjoinedDagResult.Output -notmatch 'WP-Release depends on WP-Unknown') {
        throw "validator did not reject a conjoined unknown phase-five dependency: $($conjoinedDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $reverseAfterDagPlan = $phaseFivePlan + "`nWP-Facts runs after WP-Baseline-Evidence."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $reverseAfterDagPlan) -Encoding UTF8
    $reverseAfterDagResult = Invoke-Validator $fixtureRoot
    if ($reverseAfterDagResult.ExitCode -eq 0 -or $reverseAfterDagResult.Output -notmatch 'dependency ordering contradicts') {
        throw "validator did not reject a reverse after phase-five dependency: $($reverseAfterDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $rejectionContaminationDagPlan = $phaseFivePlan + "`nThe validator must reject malformed dependency prose, but WP-Release depends on WP-Unknown."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $rejectionContaminationDagPlan) -Encoding UTF8
    $rejectionContaminationDagResult = Invoke-Validator $fixtureRoot
    if ($rejectionContaminationDagResult.ExitCode -eq 0 -or $rejectionContaminationDagResult.Output -notmatch 'WP-Release depends on WP-Unknown') {
        throw "validator let a dependency hide behind rejection prose on the same line: $($rejectionContaminationDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $dependsUponDagPlan = $phaseFivePlan + "`nWP-Release depends upon WP-Unknown."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $dependsUponDagPlan) -Encoding UTF8
    $dependsUponDagResult = Invoke-Validator $fixtureRoot
    if ($dependsUponDagResult.ExitCode -eq 0 -or $dependsUponDagResult.Output -notmatch 'WP-Release depends on WP-Unknown') {
        throw "validator did not reject a depends-upon phase-five dependency: $($dependsUponDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $unknownConnectorDagPlan = $phaseFivePlan + "`nWP-Release relies on WP-Unknown."
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $unknownConnectorDagPlan) -Encoding UTF8
    $unknownConnectorDagResult = Invoke-Validator $fixtureRoot
    if ($unknownConnectorDagResult.ExitCode -eq 0 -or $unknownConnectorDagResult.Output -notmatch 'dependency statement could not be fully parsed') {
        throw "validator did not fail closed on an unknown multi-package connector: $($unknownConnectorDagResult.Output)"
    }
    Set-Content -LiteralPath $phaseFivePlanPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFivePlan) -Encoding UTF8

    $contradictoryProfileChecklist = $phaseFiveChecklist.Replace('- [ ] P0-20. Fixture.', '- [ ] P0-20. P0 completion requires live supervisor, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $contradictoryProfileChecklist) -Encoding UTF8
    $contradictoryProfileResult = Invoke-Validator $fixtureRoot
    if ($contradictoryProfileResult.ExitCode -eq 0 -or $contradictoryProfileResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator did not reject an additive phase-five P0 deployment gate: $($contradictoryProfileResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $deferralThenObligationChecklist = $phaseFiveChecklist.Replace('Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4.', 'Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4. P0 completion requires a real supervisor run, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $deferralThenObligationChecklist) -Encoding UTF8
    $deferralThenObligationResult = Invoke-Validator $fixtureRoot
    if ($deferralThenObligationResult.ExitCode -eq 0 -or $deferralThenObligationResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator let a live obligation hide behind a P0 deferral: $($deferralThenObligationResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $sameClauseDeferralMaskChecklist = $phaseFiveChecklist.Replace('Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4.', 'P0 does not require an ENOSPC run，但是 P0 completion requires a real supervisor run, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $sameClauseDeferralMaskChecklist) -Encoding UTF8
    $sameClauseDeferralMaskResult = Invoke-Validator $fixtureRoot
    if ($sameClauseDeferralMaskResult.ExitCode -eq 0 -or $sameClauseDeferralMaskResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator let a same-clause live obligation hide behind a Chinese transition: $($sameClauseDeferralMaskResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $obligationThenDeferralChecklist = $phaseFiveChecklist.Replace('Produce `artifactKind=contract-fixture`; live validation belongs to 5A-D-2 or 5B-4.', 'P0 completion requires a real supervisor run, but live profile validation belongs to 5B-4.')
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $obligationThenDeferralChecklist) -Encoding UTF8
    $obligationThenDeferralResult = Invoke-Validator $fixtureRoot
    if ($obligationThenDeferralResult.ExitCode -eq 0 -or $obligationThenDeferralResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator let a P0 obligation hide before a later deferral: $($obligationThenDeferralResult.Output)"
    }
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $phaseFiveChecklist) -Encoding UTF8

    $continuationProfileChecklist = $phaseFiveChecklist.Replace('- [ ] P0-20. Fixture.', "- [ ] P0-20. Fixture.`n  Completion requires a real supervisor run, cleanup schedule, watchdog/recovery, ENOSPC evidence, and a live deployment profile.")
    Set-Content -LiteralPath $phaseFiveChecklistPath -Value ($phaseFiveFrontmatter + "`n" + $continuationProfileChecklist) -Encoding UTF8
    $continuationProfileResult = Invoke-Validator $fixtureRoot
    if ($continuationProfileResult.ExitCode -eq 0 -or $continuationProfileResult.Output -notmatch 'P0 deployment evidence clause must stop at contract fixtures') {
        throw "validator did not reject a multiline phase-five P0 deployment gate: $($continuationProfileResult.Output)"
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
status: closed
audit_schema: plan-audit/v2
audit_id: AUD-0004
auditor: validator-test
audit_type: targeted
scope: plan:PLN-0001,PLN-0005
subject: validator plan fixture
baseline: git:0000000; worktree:clean
started_at: 2026-07-14T00:30:00+08:00
completed_at: 2026-07-14T00:45:00+08:00
last_updated: 2026-07-14
related_audits: none
related_remediations: none
supersedes: none
related_plans: PLN-0001,PLN-0005
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

<!-- plan-checklist-audit: PLN-0005 -->
### PLN-0005 Plan/Checklist 审计

- Plan: [Plan](../../plans/PLN-0005-phase-05-attachment-lifecycle.md)
- Checklist: [Checklist](../../plans/PLN-0005-phase-05-attachment-lifecycle-checklist.md)

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
    $auditIndexContent += "`n- [AUD-0004](./records/$planAuditRecordName): ``status=closed``; ``remediation=none``; fixture."
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

$acceptancePlanAuditFrontmatter = @'
---
status: closed
audit_schema: plan-acceptance/v1
audit_id: AUD-0005
auditor: validator-test
audit_type: acceptance
acceptance_type: plan-readiness
acceptance_verdict: ready
independence_basis: fresh-context-independent-rerun
scope: plan:PLN-0001,PLN-0005
subject: validator plan readiness
baseline: git:0000000000000000000000000000000000000000; worktree:clean
evidence_revision: git:0000000000000000000000000000000000000000; worktree:clean
evidence_run_id: 11111111-1111-4111-8111-111111111111
started_at: 2026-07-14T03:00:00+08:00
completed_at: 2026-07-14T03:15:00+08:00
last_updated: 2026-07-14
related_audits: AUD-0004
related_remediations: none
supersedes: none
related_plans: PLN-0001,PLN-0005
---
'@
    $acceptancePlanAuditName = 'AUD-0005-20260714-validator-plan-validator-readiness.md'
    $acceptancePlanMatrix = @'
# Plan readiness acceptance

<!-- plan-acceptance-audit: PLN-0001 -->

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| READY_IDENTITY | fixture identity | pass | none |
| READY_SCOPE | fixture scope | pass | none |
| READY_FACTS | fixture facts | pass | none |
| READY_DEPENDENCIES | fixture dependencies | pass | none |
| READY_DESIGN | fixture design | pass | none |
| READY_EVIDENCE | fixture evidence | pass | none |
| READY_GATES | fixture gates | pass | none |
| PLAN_AUDIT_CHAIN_CLEAN | fixture plan audit chain | pass | none |

<!-- plan-acceptance-audit: PLN-0005 -->

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| READY_IDENTITY | fixture identity | pass | none |
| READY_SCOPE | fixture scope | pass | none |
| READY_FACTS | fixture facts | pass | none |
| READY_DEPENDENCIES | fixture dependencies | pass | none |
| READY_DESIGN | fixture design | pass | none |
| READY_EVIDENCE | fixture evidence | pass | none |
| READY_GATES | fixture gates | pass | none |
| PLAN_AUDIT_CHAIN_CLEAN | fixture plan audit chain | pass | none |
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $implementationRecordsRoot = Join-Path $fixtureRoot 'implementations\records'
    $implementationTemplatesRoot = Join-Path $fixtureRoot 'implementations\templates'
    New-Item -ItemType Directory -Path $implementationRecordsRoot -Force | Out-Null
    New-Item -ItemType Directory -Path $implementationTemplatesRoot -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value "# Implementations`n`n- [IMP-0001](./records/IMP-0001-20260714-validator-plan-pln-0001-fixture.md): ``status=completed``; ``audit=audited-by:AUD-0006``; ``acceptance=accepted-by:AUD-0007``; fixture." -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $implementationTemplatesRoot 'implementation-record.md') -Value '# Template' -Encoding UTF8
    $implementationFrontmatter = @'
---
status: completed
implementation_id: IMP-0001
implementer: validator
scope: plan:PLN-0001
related_plans: PLN-0001
plan_acceptance_audits: AUD-0005
baseline: git:0000000; worktree:clean
result_revision: git:1111111111111111111111111111111111111111
started_at: 2026-07-14T03:30:00+08:00
completed_at: 2026-07-14T04:00:00+08:00
last_updated: 2026-07-14
---
'@
    $implementationRecordName = 'IMP-0001-20260714-validator-plan-pln-0001-fixture.md'
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($implementationFrontmatter + "`n# Implementation") -Encoding UTF8

$implementationAuditFrontmatter = @'
---
status: closed
audit_schema: implementation-audit/v1
audit_id: AUD-0006
auditor: validator-test
audit_type: implementation
scope: implementation:IMP-0001
subject: validator implementation
baseline: git:1111111; worktree:clean
started_at: 2026-07-14T04:30:00+08:00
completed_at: 2026-07-14T04:45:00+08:00
last_updated: 2026-07-14
related_audits: none
related_remediations: none
related_implementations: IMP-0001
supersedes: none
related_plans: PLN-0001
---
'@
    $implementationAuditName = 'AUD-0006-20260714-validator-implementation-imp-0001-fixture.md'
    $implementationAuditMatrix = @'
# Implementation audit

<!-- implementation-audit: IMP-0001 -->

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_TRACEABILITY | fixture traceability | pass | none |
| CHECKLIST_EVIDENCE | fixture checklist | pass | none |
| CODE_CONTRACT | fixture contract | pass | none |
| TEST_FAILURE | fixture tests | pass | none |
| SECURITY_DATA | fixture security | pass | none |
| MIGRATION_RECOVERY | fixture recovery | pass | none |
| DOCS_CI_RELEASE | fixture release | pass | none |
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAuditName) -Value ($implementationAuditFrontmatter + "`n" + $implementationAuditMatrix) -Encoding UTF8

$implementationAcceptanceFrontmatter = @'
---
status: closed
audit_schema: implementation-acceptance/v1
audit_id: AUD-0007
auditor: validator-test
audit_type: acceptance
acceptance_type: implementation-completion
acceptance_verdict: complete
independence_basis: fresh-context-independent-rerun
scope: plan:PLN-0001
subject: validator implementation completion
baseline: git:1111111111111111111111111111111111111111; worktree:clean
evidence_revision: git:1111111111111111111111111111111111111111; worktree:clean
evidence_run_id: 22222222-2222-4222-8222-222222222222
started_at: 2026-07-14T05:00:00+08:00
completed_at: 2026-07-14T05:15:00+08:00
last_updated: 2026-07-14
related_audits: AUD-0005, AUD-0006
related_remediations: none
related_implementations: IMP-0001
supersedes: none
related_plans: PLN-0001
---
'@
    $implementationAcceptanceName = 'AUD-0007-20260714-validator-plan-pln-0001-completion-acceptance.md'
    $implementationAcceptanceMatrix = @'
# Implementation completion acceptance

<!-- implementation-acceptance-audit: PLN-0001 -->

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_PRESENT | fixture IMP | pass | none |
| SCOPE_COMPLETE | fixture scope | pass | none |
| CHECKLIST_COMPLETE | fixture checklist | pass | none |
| VALIDATION_GATES | fixture gates | pass | none |
| AUDIT_CHAIN_CLEAN | fixture plan and implementation audit chains | pass | none |
| RESIDUAL_RISK | fixture risk | pass | none |
| ARCHIVE_READY | fixture archive | pass | none |
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($implementationAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $auditIndexContent += "`n- [AUD-0005](./records/$acceptancePlanAuditName): ``status=closed``; ``remediation=none``; fixture."
    $auditIndexContent += "`n- [AUD-0006](./records/$implementationAuditName): ``status=closed``; ``remediation=none``; fixture."
    $auditIndexContent += "`n- [AUD-0007](./records/$implementationAcceptanceName): ``status=closed``; ``remediation=none``; fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $validResult = Invoke-Validator $fixtureRoot
    if ($validResult.ExitCode -ne 0) {
        throw "validator rejected valid fixture: $($validResult.Output)"
    }

    $requiredSourceAuditIndex = [regex]::Replace($auditIndexContent, '(?m)^(.*\[AUD-0004\].*?)remediation=none', '${1}remediation=required')
    Set-Content -LiteralPath $auditIndexPath -Value $requiredSourceAuditIndex -Encoding UTF8
    $requiredSourceAuditResult = Invoke-Validator $fixtureRoot
    if ($requiredSourceAuditResult.ExitCode -eq 0 -or $requiredSourceAuditResult.Output -notmatch 'clean related audit chain') {
        throw "validator accepted readiness over a required source audit: $($requiredSourceAuditResult.Output)"
    }
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $readyWithFailedControl = $acceptancePlanMatrix.Replace(
        '| READY_GATES | fixture gates | pass | none |',
        '| READY_GATES | fixture gates | fail | AUD-0005-F001 |'
    ) + "`n### AUD-0005-F001 - Fixture readiness failure"
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $readyWithFailedControl) -Encoding UTF8
    $readyWithFailedControlResult = Invoke-Validator $fixtureRoot
    if ($readyWithFailedControlResult.ExitCode -eq 0 -or $readyWithFailedControlResult.Output -notmatch 'Acceptance verdict ready requires every Control to pass') {
        throw "validator accepted ready with a failed Control: $($readyWithFailedControlResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $invalidIndependenceFrontmatter = $acceptancePlanAuditFrontmatter.Replace('independence_basis: fresh-context-independent-rerun', 'independence_basis: self-approved')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($invalidIndependenceFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $invalidIndependenceResult = Invoke-Validator $fixtureRoot
    if ($invalidIndependenceResult.ExitCode -eq 0 -or $invalidIndependenceResult.Output -notmatch 'valid independence_basis') {
        throw "validator accepted an untracked independence claim: $($invalidIndependenceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $missingRelatedPlanAudit = $acceptancePlanAuditFrontmatter.Replace('related_audits: AUD-0004', 'related_audits: none')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($missingRelatedPlanAudit + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $missingRelatedPlanAuditResult = Invoke-Validator $fixtureRoot
    if ($missingRelatedPlanAuditResult.ExitCode -eq 0 -or $missingRelatedPlanAuditResult.Output -notmatch 'latest matching plan audit|matching closed plan-audit/v2') {
        throw "validator accepted plan readiness without a matching plan audit: $($missingRelatedPlanAuditResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $mismatchedEvidenceRevision = $acceptancePlanAuditFrontmatter.Replace(
        'evidence_revision: git:0000000000000000000000000000000000000000; worktree:clean',
        'evidence_revision: git:3333333333333333333333333333333333333333; worktree:clean'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($mismatchedEvidenceRevision + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $mismatchedEvidenceRevisionResult = Invoke-Validator $fixtureRoot
    if ($mismatchedEvidenceRevisionResult.ExitCode -eq 0 -or $mismatchedEvidenceRevisionResult.Output -notmatch 'evidence_revision must match baseline') {
        throw "validator accepted acceptance evidence from a different revision: $($mismatchedEvidenceRevisionResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $sameAuditorFrontmatter = $acceptancePlanAuditFrontmatter.Replace('independence_basis: fresh-context-independent-rerun', 'independence_basis: separate-auditor')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($sameAuditorFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $sameAuditorResult = Invoke-Validator $fixtureRoot
    if ($sameAuditorResult.ExitCode -eq 0 -or $sameAuditorResult.Output -notmatch 'must use a different auditor') {
        throw "validator accepted a separate-auditor claim from the same auditor: $($sameAuditorResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $dirtyOmittedAuditFrontmatter = $planAuditFrontmatter.Replace('AUD-0004', 'AUD-0003').Replace('2026-07-14T00:30:00+08:00', '2026-07-14T00:20:00+08:00').Replace('2026-07-14T00:45:00+08:00', '2026-07-14T00:25:00+08:00')
    $dirtyOmittedAuditName = 'AUD-0003-20260714-validator-plan-omitted-dirty-fixture.md'
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $dirtyOmittedAuditName) -Value ($dirtyOmittedAuditFrontmatter + "`n" + $planAuditMatrix) -Encoding UTF8
    $dirtyOmittedIndex = $auditIndexContent + "`n- [AUD-0003](./records/$dirtyOmittedAuditName): ``status=closed``; ``remediation=required``; omitted dirty fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $dirtyOmittedIndex -Encoding UTF8
    $dirtyOmittedResult = Invoke-Validator $fixtureRoot
    if ($dirtyOmittedResult.ExitCode -eq 0 -or $dirtyOmittedResult.Output -notmatch 'dirty derived audit chain') {
        throw "validator accepted readiness while omitting a dirty related audit: $($dirtyOmittedResult.Output)"
    }
    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $dirtyOmittedAuditName)
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $missingLatestPlanAcceptance = $implementationAcceptanceFrontmatter.Replace('related_audits: AUD-0005, AUD-0006', 'related_audits: AUD-0006')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($missingLatestPlanAcceptance + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $missingLatestPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($missingLatestPlanAcceptanceResult.ExitCode -eq 0 -or $missingLatestPlanAcceptanceResult.Output -notmatch 'latest plan acceptance') {
        throw "validator accepted completion without the latest plan acceptance: $($missingLatestPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($implementationAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8

    $duplicateEvidenceRun = $implementationAcceptanceFrontmatter.Replace('22222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($duplicateEvidenceRun + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $duplicateEvidenceRunResult = Invoke-Validator $fixtureRoot
    if ($duplicateEvidenceRunResult.ExitCode -eq 0 -or $duplicateEvidenceRunResult.Output -notmatch 'evidence_run_id must be globally unique') {
        throw "validator accepted a reused acceptance evidence run: $($duplicateEvidenceRunResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($implementationAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8

    $partialImplementation = $implementationFrontmatter.Replace('status: completed', 'status: partial')
    $partialImplementationIndex = (Get-Content -Raw -Encoding UTF8 (Join-Path $fixtureRoot 'implementations\README.md')).Replace('status=completed', 'status=partial')
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($partialImplementation + "`n# Implementation") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value $partialImplementationIndex -Encoding UTF8
    $partialImplementationResult = Invoke-Validator $fixtureRoot
    if ($partialImplementationResult.ExitCode -eq 0 -or $partialImplementationResult.Output -notmatch 'may only reference a completed IMP') {
        throw "validator accepted an implementation audit for a partial IMP: $($partialImplementationResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($implementationFrontmatter + "`n# Implementation") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value "# Implementations`n`n- [IMP-0001](./records/IMP-0001-20260714-validator-plan-pln-0001-fixture.md): ``status=completed``; ``audit=audited-by:AUD-0006``; ``acceptance=accepted-by:AUD-0007``; fixture." -Encoding UTF8

    $missingChainControl = $acceptancePlanMatrix.Replace('| PLAN_AUDIT_CHAIN_CLEAN | fixture plan audit chain | pass | none |', '')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $missingChainControl) -Encoding UTF8
    $missingChainControlResult = Invoke-Validator $fixtureRoot
    if ($missingChainControlResult.ExitCode -eq 0 -or $missingChainControlResult.Output -notmatch 'PLAN_AUDIT_CHAIN_CLEAN') {
        throw "validator accepted plan readiness without audit-chain control: $($missingChainControlResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $dirtyAcceptanceFrontmatter = $acceptancePlanAuditFrontmatter.Replace('worktree:clean', 'worktree:dirty')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($dirtyAcceptanceFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $dirtyAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($dirtyAcceptanceResult.ExitCode -eq 0 -or $dirtyAcceptanceResult.Output -notmatch 'full git SHA on a clean worktree') {
        throw "validator accepted dirty acceptance evidence: $($dirtyAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $missingPlanAcceptance = $implementationFrontmatter.Replace('plan_acceptance_audits: AUD-0005', 'plan_acceptance_audits: AUD-9999')
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($missingPlanAcceptance + "`n# Implementation") -Encoding UTF8
    $missingPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($missingPlanAcceptanceResult.ExitCode -eq 0 -or $missingPlanAcceptanceResult.Output -notmatch 'missing plan acceptance audit') {
        throw "validator accepted a missing plan acceptance reference: $($missingPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($implementationFrontmatter + "`n# Implementation") -Encoding UTF8

    $mismatchedImplementationPlan = $implementationFrontmatter.Replace('related_plans: PLN-0001', 'related_plans: PLN-0005')
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($mismatchedImplementationPlan + "`n# Implementation") -Encoding UTF8
    $mismatchedImplementationPlanResult = Invoke-Validator $fixtureRoot
    if ($mismatchedImplementationPlanResult.ExitCode -eq 0 -or $mismatchedImplementationPlanResult.Output -notmatch 'related plan does not match filename') {
        throw "validator accepted an IMP with mismatched plan identity: $($mismatchedImplementationPlanResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $implementationRecordName) -Value ($implementationFrontmatter + "`n# Implementation") -Encoding UTF8

    $missingAcceptanceImplementation = $implementationAcceptanceFrontmatter.Replace('related_implementations: IMP-0001', 'related_implementations: IMP-9999')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($missingAcceptanceImplementation + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $missingAcceptanceImplementationResult = Invoke-Validator $fixtureRoot
    if ($missingAcceptanceImplementationResult.ExitCode -eq 0 -or $missingAcceptanceImplementationResult.Output -notmatch 'missing implementation') {
        throw "validator accepted completion against a missing IMP: $($missingAcceptanceImplementationResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($implementationAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8

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
    Write-Output 'Validator self-test passed: valid governance and atomic ID allocation accepted; stale or dirty audit chains, reused evidence runs, revision drift, partial IMP audits, incomplete matrices, broken references, and invalid lifecycle states rejected.'
} finally {
    if (Test-Path -LiteralPath $allocatorRoot) {
        $resolvedAllocatorRoot = (Resolve-Path $allocatorRoot).Path
        $allowedAllocatorPrefix = (Resolve-Path $PSScriptRoot).Path + [System.IO.Path]::DirectorySeparatorChar + '.allocator-fixture-'
        if (-not $resolvedAllocatorRoot.StartsWith($allowedAllocatorPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected allocator fixture path: $resolvedAllocatorRoot"
        }
        Remove-Item -LiteralPath $resolvedAllocatorRoot -Recurse -Force
    }
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
