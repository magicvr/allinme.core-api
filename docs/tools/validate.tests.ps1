$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$fixtureBase = Join-Path $repoRoot '.tmp'
$fixtureRoot = Join-Path $fixtureBase ('.validate-fixture-' + [Guid]::NewGuid().ToString('N'))
$allocatorRoot = Join-Path $fixtureBase ('.allocator-fixture-' + [Guid]::NewGuid().ToString('N'))
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
    New-Item -ItemType Directory -Path $fixtureBase -Force | Out-Null
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
baseline: git:0000000000000000000000000000000000000000; worktree:clean
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

    $openV3AuditOne = $auditFrontmatter.Replace(
        'status: open',
        "status: open`ngovernance_contract: audit-loop/v3`nexecution_context_id: 10101010-1010-4010-8010-101010101010"
    )
    $openV3AuditTwo = $openV3AuditOne.Replace('audit_id: AUD-0001', 'audit_id: AUD-0011').Replace(
        'execution_context_id: 10101010-1010-4010-8010-101010101010',
        'execution_context_id: 11111111-1010-4010-8010-101010101010'
    )
    $duplicateOpenAuditName = 'AUD-0011-20260714-validator-feature-validator-fixture-copy.md'
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $auditRecordName) -Value ($openV3AuditOne + "`n# Audit") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $duplicateOpenAuditName) -Value ($openV3AuditTwo + "`n# Audit") -Encoding UTF8
    $duplicateOpenAuditIndex = $auditIndexContent + "`n- [AUD-0011](./records/$duplicateOpenAuditName): ``status=open``; ``remediation=pending``; duplicate open fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $duplicateOpenAuditIndex -Encoding UTF8
    $duplicateOpenAuditResult = Invoke-Validator $fixtureRoot
    if ($duplicateOpenAuditResult.ExitCode -eq 0 -or $duplicateOpenAuditResult.Output -notmatch 'Duplicate open audit-loop/v3') {
        throw "validator accepted duplicate resumable open audits: $($duplicateOpenAuditResult.Output)"
    }
    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $duplicateOpenAuditName)
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $auditRecordName) -Value ($auditFrontmatter + "`n# Audit") -Encoding UTF8
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

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
status: in-progress
remediation_id: REM-0001
implementer: validator-test
scope: audit:AUD-0001
source_audits: AUD-0001
source_findings: AUD-0001-F001
baseline: git:0000000; worktree:clean
started_at: 2026-07-14T01:00:00+08:00
completed_at: pending
last_updated: 2026-07-14
related_plans: none
---
'@
    $remediationRecordName = 'REM-0001-20260714-validator-audit-validator-fixture.md'
    Set-Content -LiteralPath (Join-Path $remediationRecordsRoot $remediationRecordName) -Value ($remediationFrontmatter + "`n# Remediation") -Encoding UTF8
    $remediationIndexPath = Join-Path $fixtureRoot 'remediations\README.md'
    Set-Content -LiteralPath $remediationIndexPath -Value ("# Remediations`n`n- [REM-0001](./records/$remediationRecordName): ``status=in-progress``; ``verification=not-ready``; fixture.") -Encoding UTF8

$acceptancePlanAuditFrontmatter = @'
---
status: closed
governance_contract: audit-loop/v3
audit_schema: plan-acceptance/v2
audit_id: AUD-0005
auditor: validator-test
execution_context_id: 55555555-5555-4555-8555-555555555555
source_context_ids: legacy-unavailable
audit_type: acceptance
acceptance_type: plan-readiness
acceptance_verdict: ready
plan_status_at_acceptance: active
independence_basis: separate-context
scope: plan:PLN-0001
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
related_plans: PLN-0001
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

## 验证结果

- command: `fixture plan readiness`; result: 通过

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
governance_contract: audit-loop/v3
implementation_schema: implementation/v2
implementation_id: IMP-0001
implementer: validator
execution_context_id: 33333333-3333-4333-8333-333333333333
scope: plan:PLN-0001
related_plans: PLN-0001
plan_acceptance_audits: AUD-0005
plan_evidence_revision: git:0000000000000000000000000000000000000000
baseline: git:0000000000000000000000000000000000000000; worktree:clean
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
governance_contract: audit-loop/v3
audit_schema: implementation-audit/v1
audit_id: AUD-0006
auditor: validator-test
execution_context_id: 66666666-6666-4666-8666-666666666666
audit_type: implementation
scope: implementation:IMP-0001
subject: validator implementation
baseline: git:1111111111111111111111111111111111111111; worktree:clean
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

## 验证结果

- command: `fixture implementation audit`; result: 通过
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAuditName) -Value ($implementationAuditFrontmatter + "`n" + $implementationAuditMatrix) -Encoding UTF8

$implementationAcceptanceFrontmatter = @'
---
status: closed
governance_contract: audit-loop/v3
audit_schema: implementation-acceptance/v2
audit_id: AUD-0007
auditor: validator-test
execution_context_id: 77777777-7777-4777-8777-777777777777
source_context_ids: 55555555-5555-4555-8555-555555555555, 66666666-6666-4666-8666-666666666666, 33333333-3333-4333-8333-333333333333
audit_type: acceptance
acceptance_type: implementation-completion
acceptance_verdict: complete
plan_status_at_acceptance: active
independence_basis: separate-context
scope: plan:PLN-0001
subject: validator implementation completion
baseline: git:1111111111111111111111111111111111111111; worktree:clean
evidence_revision: git:1111111111111111111111111111111111111111; worktree:clean
evidence_run_id: 22222222-2222-4222-8222-222222222222
effective_result_revision: git:1111111111111111111111111111111111111111
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

## 验证结果

- command: `fixture completion acceptance`; result: 通过
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

    $sameContextPlanAcceptance = $acceptancePlanAuditFrontmatter.Replace(
        'source_context_ids: legacy-unavailable',
        'source_context_ids: 55555555-5555-4555-8555-555555555555'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($sameContextPlanAcceptance + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $sameContextPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($sameContextPlanAcceptanceResult.ExitCode -eq 0 -or $sameContextPlanAcceptanceResult.Output -notmatch 'execution context must differ') {
        throw "validator accepted self-approved plan acceptance context: $($sameContextPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $placeholderPlanAcceptance = $acceptancePlanMatrix.Replace('| READY_SCOPE | fixture scope | pass | none |', '| READY_SCOPE | 具体证据 | pass | none |')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $placeholderPlanAcceptance) -Encoding UTF8
    $placeholderPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($placeholderPlanAcceptanceResult.ExitCode -eq 0 -or $placeholderPlanAcceptanceResult.Output -notmatch 'evidence is empty') {
        throw "validator accepted placeholder acceptance evidence: $($placeholderPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $blockedPlanAcceptanceFrontmatter = $acceptancePlanAuditFrontmatter.Replace('acceptance_verdict: ready', 'acceptance_verdict: blocked')
    $blockedPlanAcceptanceMatrix = $acceptancePlanMatrix.Replace(
        '| READY_SCOPE | fixture scope | pass | none |',
        '| READY_SCOPE | external fixture blocker | fail | AUD-0005-F001 |'
    ) + @'

### AUD-0005-F001 - External fixture blocker

- Severity: high
- Evidence: fixture permission unavailable
- Impact: implementation cannot start
- Recommendation: obtain authorization
- Owner: fixture-owner
- Disposition: open
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($blockedPlanAcceptanceFrontmatter + "`n" + $blockedPlanAcceptanceMatrix) -Encoding UTF8
    $blockedPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($blockedPlanAcceptanceResult.ExitCode -eq 0 -or $blockedPlanAcceptanceResult.Output -notmatch 'decision-required') {
        throw "validator did not route blocked acceptance to decision-required: $($blockedPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $multiPlanV3Audit = $planAuditFrontmatter.Replace(
        'status: closed',
        "status: closed`ngovernance_contract: audit-loop/v3`nexecution_context_id: 44444444-4444-4444-8444-444444444444"
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $planAuditRecordName) -Value ($multiPlanV3Audit + "`n" + $planAuditMatrix + "`n## 验证结果`n`n- command: ``fixture plan audit``; result: 通过") -Encoding UTF8
    $multiPlanV3AuditResult = Invoke-Validator $fixtureRoot
    if ($multiPlanV3AuditResult.ExitCode -eq 0 -or $multiPlanV3AuditResult.Output -notmatch 'must identify exactly one plan') {
        throw "validator accepted a shared multi-plan v3 audit: $($multiPlanV3AuditResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $planAuditRecordName) -Value ($planAuditFrontmatter + "`n" + $planAuditMatrix) -Encoding UTF8

    $archivedPlanFrontmatter = $planFrontmatter.Replace('status: active', 'status: archived')
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture.md') -Value ($archivedPlanFrontmatter + "`n# Plan") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture-checklist.md') -Value ($archivedPlanFrontmatter + "`n# Checklist") -Encoding UTF8
    $stableArchiveResult = Invoke-Validator $fixtureRoot
    if ($stableArchiveResult.ExitCode -ne 0) {
        throw "validator invalidated historical acceptance after stable-path archival: $($stableArchiveResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture.md') -Value ($planFrontmatter + "`n# Plan") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $plansRoot 'PLN-0001-validator-fixture-checklist.md') -Value ($planFrontmatter + "`n# Checklist") -Encoding UTF8

    $multiPlanAcceptanceFrontmatter = $acceptancePlanAuditFrontmatter.Replace(
        'scope: plan:PLN-0001',
        'scope: plan:PLN-0001,PLN-0005'
    ).Replace(
        'related_plans: PLN-0001',
        'related_plans: PLN-0001,PLN-0005'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($multiPlanAcceptanceFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $multiPlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($multiPlanAcceptanceResult.ExitCode -eq 0 -or $multiPlanAcceptanceResult.Output -notmatch 'exactly one related plan') {
        throw "validator accepted a multi-plan acceptance verdict: $($multiPlanAcceptanceResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $extraAcceptanceMatrix = $acceptancePlanMatrix + "`n" + $acceptancePlanMatrix.Replace('PLN-0001', 'PLN-0005')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $extraAcceptanceMatrix) -Encoding UTF8
    $extraAcceptanceMatrixResult = Invoke-Validator $fixtureRoot
    if ($extraAcceptanceMatrixResult.ExitCode -eq 0 -or $extraAcceptanceMatrixResult.Output -notmatch 'exactly one plan matrix') {
        throw "validator accepted an undeclared extra acceptance matrix: $($extraAcceptanceMatrixResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $wrongPlanVerdict = $acceptancePlanAuditFrontmatter.Replace('acceptance_verdict: ready', 'acceptance_verdict: complete')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($wrongPlanVerdict + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $wrongPlanVerdictResult = Invoke-Validator $fixtureRoot
    if ($wrongPlanVerdictResult.ExitCode -eq 0 -or $wrongPlanVerdictResult.Output -notmatch 'invalid acceptance_verdict') {
        throw "validator accepted an implementation verdict in a plan acceptance: $($wrongPlanVerdictResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($acceptancePlanAuditFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8

    $forgedTransitionIndex = [regex]::Replace($auditIndexContent, '(?m)^(.*\[AUD-0004\].*?)remediation=none', '${1}remediation=continued-by:AUD-9999')
    Set-Content -LiteralPath $auditIndexPath -Value $forgedTransitionIndex -Encoding UTF8
    $forgedTransitionResult = Invoke-Validator $fixtureRoot
    if ($forgedTransitionResult.ExitCode -eq 0 -or $forgedTransitionResult.Output -notmatch 'missing follow-up audit') {
        throw "validator accepted a forged clean audit transition: $($forgedTransitionResult.Output)"
    }
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $mismatchedEffectiveRevision = $implementationAcceptanceFrontmatter.Replace(
        'effective_result_revision: git:1111111111111111111111111111111111111111',
        'effective_result_revision: git:2222222222222222222222222222222222222222'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($mismatchedEffectiveRevision + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $mismatchedEffectiveRevisionResult = Invoke-Validator $fixtureRoot
    if ($mismatchedEffectiveRevisionResult.ExitCode -eq 0 -or $mismatchedEffectiveRevisionResult.Output -notmatch 'effective_result_revision') {
        throw "validator accepted an effective revision outside the IMP/REM chain: $($mismatchedEffectiveRevisionResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationAcceptanceName) -Value ($implementationAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8

    $laterImplementationFrontmatter = $implementationFrontmatter.Replace('implementation_id: IMP-0001', 'implementation_id: IMP-0002').Replace(
        'status: completed',
        'status: partial'
    ).Replace(
        'result_revision: git:1111111111111111111111111111111111111111',
        'result_revision: pending'
    ).Replace(
        'started_at: 2026-07-14T03:30:00+08:00',
        'started_at: 2026-07-14T06:00:00+08:00'
    ).Replace(
        'completed_at: 2026-07-14T04:00:00+08:00',
        'completed_at: 2026-07-14T06:15:00+08:00'
    )
    $laterImplementationName = 'IMP-0002-20260714-validator-plan-pln-0001-later-fixture.md'
    Set-Content -LiteralPath (Join-Path $implementationRecordsRoot $laterImplementationName) -Value ($laterImplementationFrontmatter + "`n# Later implementation") -Encoding UTF8
    $implementationIndexWithLaterAttempt = (Get-Content -Raw -Encoding UTF8 (Join-Path $fixtureRoot 'implementations\README.md')) + "`n- [IMP-0002](./records/$laterImplementationName): ``status=partial``; ``audit=not-ready``; ``acceptance=not-ready``; fixture."
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value $implementationIndexWithLaterAttempt -Encoding UTF8
    $staleImplementationAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($staleImplementationAcceptanceResult.ExitCode -eq 0 -or $staleImplementationAcceptanceResult.Output -notmatch 'latest IMP') {
        throw "validator accepted completion of a stale IMP: $($staleImplementationAcceptanceResult.Output)"
    }
    Remove-Item -LiteralPath (Join-Path $implementationRecordsRoot $laterImplementationName)
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value "# Implementations`n`n- [IMP-0001](./records/IMP-0001-20260714-validator-plan-pln-0001-fixture.md): ``status=completed``; ``audit=audited-by:AUD-0006``; ``acceptance=accepted-by:AUD-0007``; fixture." -Encoding UTF8

    $latePlanAcceptanceName = 'AUD-0008-20260714-validator-plan-validator-late-readiness.md'
    $latePlanAcceptanceFrontmatter = $acceptancePlanAuditFrontmatter.Replace('audit_id: AUD-0005', 'audit_id: AUD-0008').Replace(
        'acceptance_verdict: ready',
        'acceptance_verdict: not-ready'
    ).Replace(
        'evidence_run_id: 11111111-1111-4111-8111-111111111111',
        'evidence_run_id: 88888888-8888-4888-8888-888888888888'
    ).Replace(
        'started_at: 2026-07-14T03:00:00+08:00',
        'started_at: 2026-07-14T03:16:00+08:00'
    ).Replace(
        'completed_at: 2026-07-14T03:15:00+08:00',
        'completed_at: 2026-07-14T03:20:00+08:00'
    )
    $latePlanAcceptanceMatrix = $acceptancePlanMatrix.Replace(
        '| READY_GATES | fixture gates | pass | none |',
        '| READY_GATES | fixture gates | fail | AUD-0008-F001 |'
    ) + "`n### AUD-0008-F001 - Later readiness failure"
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $latePlanAcceptanceName) -Value ($latePlanAcceptanceFrontmatter + "`n" + $latePlanAcceptanceMatrix) -Encoding UTF8
    $auditIndexWithLateAcceptance = $auditIndexContent + "`n- [AUD-0008](./records/$latePlanAcceptanceName): ``status=closed``; ``remediation=required``; fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexWithLateAcceptance -Encoding UTF8
    $stalePlanAcceptanceResult = Invoke-Validator $fixtureRoot
    if ($stalePlanAcceptanceResult.ExitCode -eq 0 -or $stalePlanAcceptanceResult.Output -notmatch 'latest plan acceptance available when it started') {
        throw "validator accepted implementation against a stale ready plan acceptance: $($stalePlanAcceptanceResult.Output)"
    }
    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $latePlanAcceptanceName)
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8

    $remediationImplementationAuditName = 'AUD-0008-20260714-validator-implementation-imp-0001-remediation-fixture.md'
    $remediationImplementationAuditFrontmatter = $implementationAuditFrontmatter.Replace('audit_id: AUD-0006', 'audit_id: AUD-0008').Replace(
        'execution_context_id: 66666666-6666-4666-8666-666666666666',
        'execution_context_id: 88888888-8888-4888-8888-888888888888'
    ).Replace(
        'baseline: git:1111111; worktree:clean',
        'baseline: git:1111111111111111111111111111111111111111; worktree:clean'
    ).Replace(
        'started_at: 2026-07-14T04:30:00+08:00',
        'started_at: 2026-07-14T06:20:00+08:00'
    ).Replace(
        'completed_at: 2026-07-14T04:45:00+08:00',
        'completed_at: 2026-07-14T06:30:00+08:00'
    )
    $remediationImplementationAuditMatrix = $implementationAuditMatrix.Replace(
        '| CODE_CONTRACT | fixture contract | pass | none |',
        '| CODE_CONTRACT | fixture contract gap | fail | AUD-0008-F001 |'
    ) + @'

### AUD-0008-F001 - Fixture implementation gap

- Severity: high
- Evidence: fixture contract gap
- Impact: invalid implementation
- Recommendation: remediate fixture
- Owner: validator
- Disposition: open
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $remediationImplementationAuditName) -Value ($remediationImplementationAuditFrontmatter + "`n" + $remediationImplementationAuditMatrix) -Encoding UTF8

    $implementationRemediationName = 'REM-0002-20260714-validator-audit-implementation-gap.md'
    $implementationRemediationFrontmatter = @'
---
status: completed
governance_contract: audit-loop/v3
remediation_schema: remediation/v2
remediation_id: REM-0002
implementer: validator-remediator
execution_context_id: 99999999-9999-4999-8999-999999999999
scope: audit:AUD-0008
source_audits: AUD-0008
source_findings: AUD-0008-F001
baseline: git:1111111111111111111111111111111111111111; worktree:clean
result_revision: git:2222222222222222222222222222222222222222
parent_result_revision: git:1111111111111111111111111111111111111111
affects_implementation: true
related_implementations: IMP-0001
started_at: 2026-07-14T06:31:00+08:00
completed_at: 2026-07-14T06:45:00+08:00
last_updated: 2026-07-14
related_plans: PLN-0001
---
'@
    Set-Content -LiteralPath (Join-Path $remediationRecordsRoot $implementationRemediationName) -Value ($implementationRemediationFrontmatter + "`n# Implementation remediation") -Encoding UTF8

    $implementationFollowUpName = 'AUD-0009-20260714-validator-follow-up-rem-0002-fixture.md'
    $implementationFollowUpFrontmatter = @'
---
status: closed
governance_contract: audit-loop/v3
audit_id: AUD-0009
auditor: validator-follow-up
execution_context_id: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
source_context_ids: 88888888-8888-4888-8888-888888888888, 99999999-9999-4999-8999-999999999999, 33333333-3333-4333-8333-333333333333
audit_type: follow-up
independence_basis: separate-context
evidence_run_id: 99999999-aaaa-4aaa-8aaa-999999999999
scope: follow-up:REM-0002
subject: implementation remediation follow-up
baseline: git:2222222222222222222222222222222222222222; worktree:clean
started_at: 2026-07-14T06:50:00+08:00
completed_at: 2026-07-14T07:00:00+08:00
last_updated: 2026-07-14
related_audits: AUD-0008
related_remediations: REM-0002
related_implementations: IMP-0001
supersedes: none
related_plans: PLN-0001
---
'@
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationFollowUpName) -Value ($implementationFollowUpFrontmatter + "`n# Follow-up`n`n## 验证结果`n`n- command: ``fixture follow-up``; result: 通过") -Encoding UTF8

    $effectiveAcceptanceName = 'AUD-0010-20260714-validator-plan-pln-0001-effective-completion.md'
    $effectiveAcceptanceFrontmatter = $implementationAcceptanceFrontmatter.Replace('audit_id: AUD-0007', 'audit_id: AUD-0010').Replace(
        'execution_context_id: 77777777-7777-4777-8777-777777777777',
        'execution_context_id: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    ).Replace(
        'source_context_ids: 55555555-5555-4555-8555-555555555555, 66666666-6666-4666-8666-666666666666, 33333333-3333-4333-8333-333333333333',
        'source_context_ids: 55555555-5555-4555-8555-555555555555, 88888888-8888-4888-8888-888888888888, aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa, 99999999-9999-4999-8999-999999999999, 33333333-3333-4333-8333-333333333333'
    ).Replace(
        'baseline: git:1111111111111111111111111111111111111111; worktree:clean',
        'baseline: git:2222222222222222222222222222222222222222; worktree:clean'
    ).Replace(
        'evidence_revision: git:1111111111111111111111111111111111111111; worktree:clean',
        'evidence_revision: git:2222222222222222222222222222222222222222; worktree:clean'
    ).Replace(
        'evidence_run_id: 22222222-2222-4222-8222-222222222222',
        'evidence_run_id: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
    ).Replace(
        'effective_result_revision: git:1111111111111111111111111111111111111111',
        'effective_result_revision: git:2222222222222222222222222222222222222222'
    ).Replace(
        'started_at: 2026-07-14T05:00:00+08:00',
        'started_at: 2026-07-14T07:10:00+08:00'
    ).Replace(
        'completed_at: 2026-07-14T05:15:00+08:00',
        'completed_at: 2026-07-14T07:20:00+08:00'
    ).Replace(
        'related_audits: AUD-0005, AUD-0006',
        'related_audits: AUD-0005, AUD-0008, AUD-0009'
    ).Replace(
        'related_remediations: none',
        'related_remediations: REM-0002'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $effectiveAcceptanceName) -Value ($effectiveAcceptanceFrontmatter + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8

    $auditIndexWithEffectiveChain = [regex]::Replace($auditIndexContent, '(?m)^(.*\[AUD-0006\].*?)remediation=none', '${1}remediation=none')
    $auditIndexWithEffectiveChain += "`n- [AUD-0008](./records/$remediationImplementationAuditName): ``status=closed``; ``remediation=verified-by:AUD-0009``; fixture."
    $auditIndexWithEffectiveChain += "`n- [AUD-0009](./records/$implementationFollowUpName): ``status=closed``; ``remediation=none``; fixture."
    $auditIndexWithEffectiveChain += "`n- [AUD-0010](./records/$effectiveAcceptanceName): ``status=closed``; ``remediation=none``; fixture."
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexWithEffectiveChain -Encoding UTF8
    $remediationIndexWithEffectiveChain = (Get-Content -Raw -Encoding UTF8 $remediationIndexPath) + "`n- [REM-0002](./records/$implementationRemediationName): ``status=completed``; ``verification=verified-by:AUD-0009``; fixture."
    Set-Content -LiteralPath $remediationIndexPath -Value $remediationIndexWithEffectiveChain -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value "# Implementations`n`n- [IMP-0001](./records/IMP-0001-20260714-validator-plan-pln-0001-fixture.md): ``status=completed``; ``audit=audited-by:AUD-0008``; ``acceptance=accepted-by:AUD-0010``; fixture." -Encoding UTF8

    $effectiveChainResult = Invoke-Validator $fixtureRoot
    if ($effectiveChainResult.ExitCode -ne 0) {
        throw "validator rejected a valid IMP/REM effective revision chain: $($effectiveChainResult.Output)"
    }

    $disconnectedImplementationRemediation = $implementationRemediationFrontmatter.Replace(
        'parent_result_revision: git:1111111111111111111111111111111111111111',
        'parent_result_revision: git:3333333333333333333333333333333333333333'
    )
    Set-Content -LiteralPath (Join-Path $remediationRecordsRoot $implementationRemediationName) -Value ($disconnectedImplementationRemediation + "`n# Implementation remediation") -Encoding UTF8
    $disconnectedChainResult = Invoke-Validator $fixtureRoot
    if ($disconnectedChainResult.ExitCode -eq 0 -or $disconnectedChainResult.Output -notmatch 'chain is disconnected') {
        throw "validator accepted a disconnected implementation remediation chain: $($disconnectedChainResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $remediationRecordsRoot $implementationRemediationName) -Value ($implementationRemediationFrontmatter + "`n# Implementation remediation") -Encoding UTF8

    $selfReviewedFollowUp = $implementationFollowUpFrontmatter.Replace(
        'execution_context_id: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
        'execution_context_id: 99999999-9999-4999-8999-999999999999'
    )
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationFollowUpName) -Value ($selfReviewedFollowUp + "`n# Follow-up`n`n## 验证结果`n`n- command: ``fixture follow-up``; result: 通过") -Encoding UTF8
    $selfReviewedFollowUpResult = Invoke-Validator $fixtureRoot
    if ($selfReviewedFollowUpResult.ExitCode -eq 0 -or $selfReviewedFollowUpResult.Output -notmatch 'execution context must differ') {
        throw "validator accepted a self-reviewed follow-up context: $($selfReviewedFollowUpResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $implementationFollowUpName) -Value ($implementationFollowUpFrontmatter + "`n# Follow-up`n`n## 验证结果`n`n- command: ``fixture follow-up``; result: 通过") -Encoding UTF8

    $missingFindingSeverity = ($remediationImplementationAuditFrontmatter + "`n" + $remediationImplementationAuditMatrix).Replace('- Severity: high', '')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $remediationImplementationAuditName) -Value $missingFindingSeverity -Encoding UTF8
    $missingFindingSeverityResult = Invoke-Validator $fixtureRoot
    if ($missingFindingSeverityResult.ExitCode -eq 0 -or $missingFindingSeverityResult.Output -notmatch 'must record Severity') {
        throw "validator accepted an incomplete v3 finding: $($missingFindingSeverityResult.Output)"
    }
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $remediationImplementationAuditName) -Value ($remediationImplementationAuditFrontmatter + "`n" + $remediationImplementationAuditMatrix) -Encoding UTF8

    $omittedEffectiveRemediation = $effectiveAcceptanceFrontmatter.Replace('related_remediations: REM-0002', 'related_remediations: none')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $effectiveAcceptanceName) -Value ($omittedEffectiveRemediation + "`n" + $implementationAcceptanceMatrix) -Encoding UTF8
    $omittedEffectiveRemediationResult = Invoke-Validator $fixtureRoot
    if ($omittedEffectiveRemediationResult.ExitCode -eq 0 -or $omittedEffectiveRemediationResult.Output -notmatch 'include every verified implementation remediation') {
        throw "validator accepted completion while omitting an effective REM: $($omittedEffectiveRemediationResult.Output)"
    }

    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $remediationImplementationAuditName)
    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $implementationFollowUpName)
    Remove-Item -LiteralPath (Join-Path $auditRecordsRoot $effectiveAcceptanceName)
    Remove-Item -LiteralPath (Join-Path $remediationRecordsRoot $implementationRemediationName)
    Set-Content -LiteralPath $auditIndexPath -Value $auditIndexContent -Encoding UTF8
    Set-Content -LiteralPath $remediationIndexPath -Value ("# Remediations`n`n- [REM-0001](./records/$remediationRecordName): ``status=in-progress``; ``verification=not-ready``; fixture.") -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $fixtureRoot 'implementations\README.md') -Value "# Implementations`n`n- [IMP-0001](./records/IMP-0001-20260714-validator-plan-pln-0001-fixture.md): ``status=completed``; ``audit=audited-by:AUD-0006``; ``acceptance=accepted-by:AUD-0007``; fixture." -Encoding UTF8

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

    $invalidIndependenceFrontmatter = $acceptancePlanAuditFrontmatter.Replace('independence_basis: separate-context', 'independence_basis: self-approved')
    Set-Content -LiteralPath (Join-Path $auditRecordsRoot $acceptancePlanAuditName) -Value ($invalidIndependenceFrontmatter + "`n" + $acceptancePlanMatrix) -Encoding UTF8
    $invalidIndependenceResult = Invoke-Validator $fixtureRoot
    if ($invalidIndependenceResult.ExitCode -eq 0 -or $invalidIndependenceResult.Output -notmatch 'must use independence_basis=separate-context') {
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
    Write-Output 'Validator self-test passed: valid audit-loop/v3 context and linear IMP/REM chains accepted; self-review contexts, shared multi-plan AUDs, duplicate open work, placeholder evidence, blocked-state misrouting, disconnected revisions, forged transitions, dirty chains, and incomplete records rejected.'
} finally {
    if (Test-Path -LiteralPath $allocatorRoot) {
        $resolvedAllocatorRoot = (Resolve-Path $allocatorRoot).Path
        $allowedAllocatorPrefix = (Resolve-Path $fixtureBase).Path + [System.IO.Path]::DirectorySeparatorChar + '.allocator-fixture-'
        if (-not $resolvedAllocatorRoot.StartsWith($allowedAllocatorPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected allocator fixture path: $resolvedAllocatorRoot"
        }
        Remove-Item -LiteralPath $resolvedAllocatorRoot -Recurse -Force
    }
    if (Test-Path -LiteralPath $fixtureRoot) {
        $resolvedFixtureRoot = (Resolve-Path $fixtureRoot).Path
        $allowedPrefix = (Resolve-Path $fixtureBase).Path + [System.IO.Path]::DirectorySeparatorChar + '.validate-fixture-'
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
