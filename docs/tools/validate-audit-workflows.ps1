param(
    [string]$RepositoryRoot
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}

$failures = New-Object System.Collections.Generic.List[string]

function Read-WorkflowAsset([string]$RelativePath) {
    $path = Join-Path $repoRoot $RelativePath
    if (-not (Test-Path -LiteralPath $path)) {
        $failures.Add("Missing audit workflow asset: $RelativePath")
        return $null
    }
    return Get-Content -Raw -Encoding UTF8 $path
}

function Require-Pattern([string]$Content, [string]$Pattern, [string]$Message) {
    if ([string]::IsNullOrWhiteSpace($Content) -or $Content -notmatch $Pattern) {
        $failures.Add($Message)
    }
}

$recordCreators = @(
    'backend-plan-audit',
    'backend-plan-acceptance-audit',
    'backend-implement-plan',
    'backend-implementation-audit',
    'backend-implementation-acceptance-audit',
    'backend-fix-audit-findings',
    'backend-follow-up-audit'
)

$transactionHelper = Read-WorkflowAsset 'docs/tools/invoke-governance-transaction.ps1'
Require-Pattern $transactionHelper 'refs/allinme/governance-head' 'governance transaction helper must CAS the shared governance ref'
Require-Pattern $transactionHelper 'update-ref --stdin' 'governance transaction helper must update refs atomically'
Require-Pattern $transactionHelper 'git-common-dir' 'governance transaction helper must use the Git common directory lock domain'

$loopStateHelper = Read-WorkflowAsset 'docs/tools/update-loop-run-state.ps1'
Require-Pattern $loopStateHelper 'allinme-governance-loop-runs' 'loop state helper must persist state in the Git common directory'
Require-Pattern $loopStateHelper 'ExpectedGeneration' 'loop state helper must enforce generation CAS'
Require-Pattern $loopStateHelper 'ExpectedPreviousGovernanceRevision' 'loop state helper must enforce previous governance revision CAS'

$loopStateSchema = Read-WorkflowAsset 'docs/tools/governance-loop-run.schema.json'
Require-Pattern $loopStateSchema 'governance-loop-run/v1' 'loop state schema must identify governance-loop-run/v1'
Require-Pattern $loopStateSchema 'previous_governance_revision' 'loop state schema must persist the previous governance revision'
Require-Pattern $loopStateSchema 'stagnant_count' 'loop state schema must persist per-plan stagnation state'

$historyValidator = Read-WorkflowAsset 'docs/tools/validate-governance-history.ps1'
Require-Pattern $historyValidator 'HistoryBase' 'governance history validator must require an explicit history base'
Require-Pattern $historyValidator 'committed open checkpoint' 'governance history validator must require committed open checkpoints'
Require-Pattern $historyValidator 'evidence_revision' 'governance history validator must validate audit evidence ancestry'
Require-Pattern $historyValidator 'result_revision' 'governance history validator must validate implementation and remediation result ancestry'
Require-Pattern $historyValidator 'runtime_context_attestation' 'governance history validator must atomically bind open runtime attestations'
Require-Pattern $historyValidator 'evidence_attestation' 'governance history validator must atomically bind terminal evidence attestations'
Require-Pattern $historyValidator 'Terminal governance record changed after HistoryBase' 'governance history validator must inspect pre-existing records changed after HistoryBase'
Require-Pattern $historyValidator 'Refusing repository-local git executable' 'governance history validator must reject repository-local git executable hijacking'

$runtimeAttestationValidator = Read-WorkflowAsset 'docs/tools/validate-runtime-attestations.ps1'
Require-Pattern $runtimeAttestationValidator 'AUDIT_RUNTIME_TRUSTED_KEY_SHA256' 'runtime attestation validator must use an external trust anchor'
Require-Pattern $runtimeAttestationValidator 'openssl.*dgst.*sha256.*verify' 'runtime attestation validator must cryptographically verify signatures'
Require-Pattern $runtimeAttestationValidator 'source_context_attestations' 'runtime attestation validator must verify the exact signed source set'
Require-Pattern $runtimeAttestationValidator 'New audit-loop/v3 records must reference an externally signed' 'runtime attestation validator must reject new unattested records'
Require-Pattern $runtimeAttestationValidator 'unique execution_context_id' 'runtime attestation validator must reject duplicate execution context IDs for new records'

$evidenceAttestationValidator = Read-WorkflowAsset 'docs/tools/validate-evidence-attestations.ps1'
Require-Pattern $evidenceAttestationValidator 'revision-evidence-attestation/v1' 'evidence attestation validator must require the signed evidence envelope contract'
Require-Pattern $evidenceAttestationValidator 'evidence_artifact_sha256' 'evidence attestation validator must bind the exact artifact bytes'
Require-Pattern $evidenceAttestationValidator 'AUDIT_RUNTIME_TRUSTED_KEY_SHA256' 'evidence attestation validator must use an external trust anchor'
Require-Pattern $evidenceAttestationValidator 'approved fixed image digest' 'evidence attestation validator must enforce the approved image digest'

$evidenceRunner = Read-WorkflowAsset 'docs/tools/invoke-revision-evidence.ps1'
Require-Pattern $evidenceRunner 'host_repository_mounted\s*=\s*\$false' 'evidence runner must not mount the host repository'
Require-Pattern $evidenceRunner 'git-archive-tar\+manifest/v1' 'evidence runner must use a sanitized exact-revision snapshot'
Require-Pattern $evidenceRunner "'--entrypoint',\s*'/usr/bin/env'" 'evidence runner must override the image entrypoint'
Require-Pattern $evidenceRunner 'streaming-bounded' 'evidence runner must use bounded streaming output capture'
Require-Pattern $evidenceRunner 'wall-clock-timeout' 'evidence runner must enforce a wall-clock timeout'

foreach ($name in $recordCreators) {
    $content = Read-WorkflowAsset ".github/prompts/$name.prompt.md"
    Require-Pattern $content 'governance-transaction-contract:\s*invoke-governance-transaction;\s*exact-path-stage;\s*common-git-dir-lock;\s*head-and-shared-ref-cas' "$name must declare the atomic governance transaction contract"
    Require-Pattern $content 'invoke-governance-transaction\.ps1' "$name must use the governance transaction helper"
    Require-Pattern $content 'governance-handoff-contract:\s*open-checkpoint-commit;\s*reuse-existing-checkpoint;\s*no-empty-commit;(?:\s*subject-commit;)?\s*terminal-governance-commit;\s*clean-revision-return' "$name must require reusable open checkpoints, terminal governance commits, and clean revision handoff"
    Require-Pattern $content 'runtime-attestation-contract:\s*external-signed-context;\s*repository-no-signing-key;\s*scope-baseline-bound;\s*missing-attestation-stops' "$name must require externally signed runtime context"
    Require-Pattern $content 'runtime_context_attestation' "$name must persist its signed runtime attestation"
    Require-Pattern $content 'validate-runtime-attestations\.ps1' "$name must verify the runtime attestation before progress"
    Require-Pattern $content 'record-context-contract:\s*one-real-child-per-record;\s*globally-unique-execution-context-id;\s*no-batch-context-reuse' "$name must allocate one real, unique execution context per governance record"
}

$evidenceAuditCreators = @(
    'backend-plan-audit',
    'backend-plan-acceptance-audit',
    'backend-implementation-audit',
    'backend-implementation-acceptance-audit',
    'backend-follow-up-audit'
)
foreach ($name in $evidenceAuditCreators) {
    $content = Read-WorkflowAsset ".github/prompts/$name.prompt.md"
    Require-Pattern $content 'evidence-attestation-contract:\s*external-signed-artifact;\s*exact-run-revision-command-result-image;\s*missing-trust-stops' "$name must declare the external evidence attestation contract"
    Require-Pattern $content 'evidence_attestation' "$name must persist the external evidence attestation"
    Require-Pattern $content 'validate-evidence-attestations\.ps1' "$name must verify signed evidence before closing"
    Require-Pattern $content 'docs/evidence/runs/<(?:run-id|evidence_run_id)>/attestation\.json' "$name must use the canonical evidence attestation path"
}

foreach ($name in @(
    'backend-plan-acceptance-audit',
    'backend-implementation-audit',
    'backend-implementation-acceptance-audit',
    'backend-follow-up-audit'
)) {
    $content = Read-WorkflowAsset ".github/prompts/$name.prompt.md"
    Require-Pattern $content 'context-dispatch-contract:\s*runtime-provided-new-task-context;\s*runtime-ref-required;\s*correlation-uuid-not-identity' "$name must distinguish runtime identity from correlation UUIDs"
    Require-Pattern $content 'CONTEXT_REF' "$name must bind the runtime-provided context reference"
    Require-Pattern $content 'context-resume-contract:\s*same-runtime-ref-or-supersede-context-loss;\s*never-rebind-open-audit' "$name must prevent a new task from rebinding an open independent audit"
    Require-Pattern $content 'invoke-revision-evidence\.ps1' "$name must execute subject evidence at the exact revision"
    Require-Pattern $content 'runtime-independence-attestation-contract:\s*exact-signed-source-set;\s*current-task-differs-from-sources' "$name must cryptographically bind independent source contexts"
    Require-Pattern $content 'source_context_attestations' "$name must persist the exact signed source set"
}

$planAudit = Read-WorkflowAsset '.github/prompts/backend-plan-audit.prompt.md'
Require-Pattern $planAudit 'PEER_SET' 'backend-plan-audit must accept PEER_SET'
Require-Pattern $planAudit 'peer-set-contract:\s*target-subset-of-peer-set;\s*audit-target-only;\s*inspect-complete-peer-set;\s*persist-peer-snapshot' 'backend-plan-audit must persist the complete peer snapshot'
Require-Pattern $planAudit 'audited_peer_plans' 'backend-plan-audit must record the complete peer set'
Require-Pattern $planAudit 'invoke-revision-evidence\.ps1' 'backend-plan-audit must run evidence at the exact revision'

$planLoop = Read-WorkflowAsset '.github/prompts/backend-plan-audit-until-ready.prompt.md'
Require-Pattern $planLoop 'plan-loop-contract:\s*immutable-target-set;\s*separate-peer-set;\s*separate-advance-set;\s*set-aware-plan-audit;\s*verification-before-remediation;\s*per-plan-terminal-state' 'plan loop must separate goal, peer and advance sets'
Require-Pattern $planLoop 'argument-hint:.*MAX_CYCLES=8' 'plan loop default MAX_CYCLES must cover the normal audit/remediation/follow-up/acceptance path'
Require-Pattern $planLoop 'STEP_MODE=loop\|single-transition' 'plan loop must expose single-transition child mode'
Require-Pattern $planLoop 'single-transition.*MAX_CYCLES=1' 'plan loop single-transition mode must force one cycle'
Require-Pattern $planLoop 'peer-routing-contract:\s*peer-set-is-complete-active-set;\s*target-is-goal-set;\s*advance-set-is-subset;\s*plan-audit-target-is-drifted-subset' 'plan loop must preserve complete peers without expanding the goal set'
Require-Pattern $planLoop 'peer-drift-contract:\s*active-peer-set-change-requires-safe-restart;\s*no-stale-peer-progress' 'plan loop must safely restart when the active peer set changes'
Require-Pattern $planLoop 'ADVANCE_SET' 'plan loop must use an explicit advance subset'
Require-Pattern $planLoop 'PEER_SET' 'plan loop must use an explicit complete peer set'
Require-Pattern $planLoop 'standalone-goal-contract:\s*advance-set-equals-target;\s*complete-full-target-only' 'standalone plan loop must not complete a strict advance subset as the full goal'
Require-Pattern $planLoop 'context-dispatch-contract:\s*independent-stages-require-new-runtime-task;\s*runtime-ref-required;\s*uuid-is-not-isolation' 'plan loop must require real runtime task isolation'
Require-Pattern $planLoop 'governance-handoff-contract:\s*child-must-return-clean-terminal-governance-revision' 'plan loop must require clean child governance handoff'
Require-Pattern $planLoop 'stable-state-fingerprint' 'plan loop must define deterministic per-plan stagnation state'
Require-Pattern $planLoop 'terminal-reentry-contract:\s*blocked-rem-requires-changed-recovery-evidence;\s*no-automatic-retry-storm' 'plan loop must not automatically retry blocked remediation'
Require-Pattern $planLoop 'reuse-current-ready' 'plan loop must reuse a current closed ready acceptance'
Require-Pattern $planLoop 'peer_reaudit_required' 'plan loop must persistently route affected peers back to audit'
Require-Pattern $planLoop 'context-loss' 'plan loop must safely replace independent open audits whose runtime task is lost'
Require-Pattern $planLoop 'persistent-loop-state-contract:\s*governance-loop-run-v1;\s*immutable-sets;\s*generation-cas;\s*previous-governance-sha;\s*per-plan-fingerprint' 'plan loop must declare the persistent state contract'
Require-Pattern $planLoop 'update-loop-run-state\.ps1' 'plan loop must persist and resume loop state'
Require-Pattern $planLoop '(?s)update-loop-run-state\.ps1.*generation.*previous governance SHA.*child.*CAS.*child' 'plan loop must run children strictly serially around state CAS updates'
Require-Pattern $planLoop 'runtime-attestation-dispatch-contract:\s*external-signed-child;\s*exact-signed-source-set;\s*missing-trust-stops' 'plan loop must require signed child dispatch'

$implementationLoop = Read-WorkflowAsset '.github/prompts/backend-implement-audit-until-complete.prompt.md'
Require-Pattern $implementationLoop 'argument-hint:.*MAX_CYCLES=12' 'implementation loop default MAX_CYCLES must cover readiness, implementation, audit and acceptance'
Require-Pattern $implementationLoop 'orchestration-step-contract:\s*one-durable-transition-per-plan-per-cycle;\s*nested-loop-forbidden' 'implementation loop must forbid nested multi-cycle execution'
Require-Pattern $implementationLoop 'GOAL_MODE=child\s+STEP_MODE=single-transition\s+MAX_CYCLES=1' 'implementation loop must call readiness as a single-transition child'
Require-Pattern $implementationLoop 'peer-routing-contract:\s*peer-set-is-complete-active-set;\s*target-is-goal-set;\s*readiness-advance-set-is-subset' 'implementation loop must preserve complete peers without expanding implementation scope'
Require-Pattern $implementationLoop 'peer-drift-contract:\s*active-peer-set-change-requires-safe-restart;\s*no-stale-peer-progress' 'implementation loop must safely restart when the active peer set changes'
Require-Pattern $implementationLoop 'PEER_SET' 'implementation loop must pass the complete peer set to readiness'
Require-Pattern $implementationLoop 'context-dispatch-contract:\s*independent-stages-require-new-runtime-task;\s*runtime-ref-required;\s*uuid-is-not-isolation' 'implementation loop must require real runtime task isolation'
Require-Pattern $implementationLoop 'governance-handoff-contract:\s*child-must-return-clean-terminal-governance-revision' 'implementation loop must require clean child governance handoff'
Require-Pattern $implementationLoop 'stable-state-fingerprint' 'implementation loop must define deterministic per-plan stagnation state'
Require-Pattern $implementationLoop 'terminal-reentry-contract:\s*blocked-rem-requires-changed-recovery-evidence;\s*partial-or-blocked-imp-requires-current-ready-and-changed-recovery-evidence;\s*consumed-actions-are-not-replayable;\s*no-automatic-retry-storm' 'implementation loop must safely re-enter terminal IMP work without replaying consumed actions'
Require-Pattern $implementationLoop 'implemented-by:IMP-NNNN' 'implementation loop must mark implement actions as consumed'
Require-Pattern $implementationLoop 'CONTEXT_REF=<child runtime ref>' 'implementation loop must propagate runtime context references to independent children'
Require-Pattern $implementationLoop 'context-loss' 'implementation loop must safely replace independent open audits whose runtime task is lost'
Require-Pattern $implementationLoop 'persistent-loop-state-contract:\s*governance-loop-run-v1;\s*immutable-sets;\s*generation-cas;\s*previous-governance-sha;\s*per-plan-fingerprint' 'implementation loop must declare the persistent state contract'
Require-Pattern $implementationLoop 'update-loop-run-state\.ps1' 'implementation loop must persist and resume loop state'
Require-Pattern $implementationLoop '(?s)update-loop-run-state\.ps1.*generation.*previous governance SHA.*child.*CAS.*child' 'implementation loop must run children strictly serially around state CAS updates'
Require-Pattern $implementationLoop 'runtime-attestation-dispatch-contract:\s*external-signed-child;\s*exact-signed-source-set;\s*missing-trust-stops' 'implementation loop must require signed child dispatch'

$implementPlan = Read-WorkflowAsset '.github/prompts/backend-implement-plan.prompt.md'
Require-Pattern $implementPlan 'mutable-context-resume-contract:\s*same-runtime-ref-and-recoverable-task-or-context-loss-supersede' 'implement plan must only resume an IMP in its original recoverable runtime context'
Require-Pattern $implementPlan 'context-loss.*supersed' 'implement plan must replace a lost in-progress IMP instead of taking it over'
Require-Pattern $implementPlan '(?s)invoke-governance-transaction\.ps1.*status: partial.*governance_revision' 'implement plan partial branch must use subject/result and terminal transactions'
Require-Pattern $implementPlan '(?s)invoke-governance-transaction\.ps1.*status: blocked.*governance_revision' 'implement plan blocked branch must use a terminal governance transaction'

$fixFindings = Read-WorkflowAsset '.github/prompts/backend-fix-audit-findings.prompt.md'
Require-Pattern $fixFindings 'mutable-context-resume-contract:\s*same-runtime-ref-and-recoverable-task-or-context-loss-supersede' 'fix findings must only resume a REM in its original recoverable runtime context'
Require-Pattern $fixFindings 'context-loss.*supersed' 'fix findings must replace a lost in-progress REM instead of taking it over'
Require-Pattern $fixFindings '(?s)invoke-governance-transaction\.ps1.*status: partial.*governance_revision' 'fix findings partial branch must use subject/result and terminal transactions'
Require-Pattern $fixFindings '(?s)invoke-governance-transaction\.ps1.*status: blocked.*governance_revision' 'fix findings blocked branch must use a terminal governance transaction'

$auditReadme = Read-WorkflowAsset 'docs/audits/README.md'
Require-Pattern $auditReadme 'TARGET.*goal' 'audit README must define TARGET as the non-expanding goal subset'
Require-Pattern $auditReadme 'PEER_SET.*peer' 'audit README must define PEER_SET as the complete active peer set'
Require-Pattern $auditReadme 'accepted-risk.*decision-required' 'audit README must forbid repository-authored accepted-risk closure'
Require-Pattern $planAudit 'accepted-risk.*decision-required' 'plan audit must route risk acceptance to external decision instead of self-approval'
$followUpAudit = Read-WorkflowAsset '.github/prompts/backend-follow-up-audit.prompt.md'
Require-Pattern $followUpAudit 'accepted-risk.*decision-required' 'follow-up audit must route risk acceptance to external decision instead of self-approval'

$skillRequirements = @{
    'backend-plan-audit-until-ready' = @('ADVANCE_SET', 'PEER_SET', 'STEP_MODE=single-transition', 'CONTEXT_REF', 'governance_revision')
    'backend-implement-audit-until-complete' = @('PEER_SET', 'ADVANCE_SET', 'STEP_MODE=single-transition', 'CONTEXT_REF', 'governance_revision')
    'backend-plan-audit' = @('PEER_SET', 'persist the peer set', 'invoke-revision-evidence.ps1', 'terminal governance')
    'backend-plan-acceptance-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-implementation-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-implementation-acceptance-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-follow-up-audit' = @('new task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-implement-plan' = @('governance_revision')
    'backend-fix-audit-findings' = @('governance_revision')
}

foreach ($entry in $skillRequirements.GetEnumerator()) {
    $content = Read-WorkflowAsset ".agents/skills/$($entry.Key)/SKILL.md"
    Require-Pattern $content "(?m)^name:\s*$([regex]::Escape($entry.Key))\s*$" "$($entry.Key) skill metadata must match its folder"
    Require-Pattern $content "\.github/prompts/$([regex]::Escape($entry.Key))\.prompt\.md" "$($entry.Key) skill must bind its canonical prompt"
    foreach ($requiredText in $entry.Value) {
        Require-Pattern $content ([regex]::Escape($requiredText)) "$($entry.Key) skill must preserve canonical workflow requirement: $requiredText"
    }
}

foreach ($name in $recordCreators) {
    $content = Read-WorkflowAsset ".agents/skills/$name/SKILL.md"
    Require-Pattern $content 'invoke-governance-transaction\.ps1' "$name skill must require the governance transaction helper"
    Require-Pattern $content 'runtime_context_attestation' "$name skill must preserve the signed runtime context requirement"
    Require-Pattern $content 'trust anchor' "$name skill must require the external runtime trust anchor"
}

foreach ($name in @('backend-plan-acceptance-audit', 'backend-implementation-audit', 'backend-implementation-acceptance-audit', 'backend-follow-up-audit')) {
    $content = Read-WorkflowAsset ".agents/skills/$name/SKILL.md"
    Require-Pattern $content 'source_context_attestations' "$name skill must preserve exact signed source contexts"
}

foreach ($name in $evidenceAuditCreators) {
    $content = Read-WorkflowAsset ".agents/skills/$name/SKILL.md"
    Require-Pattern $content 'evidence_attestation' "$name skill must preserve externally signed evidence"
    Require-Pattern $content 'attestation\.json' "$name skill must preserve the canonical evidence attestation path"
}

foreach ($name in @('backend-plan-audit-until-ready', 'backend-implement-audit-until-complete')) {
    $content = Read-WorkflowAsset ".agents/skills/$name/SKILL.md"
    Require-Pattern $content 'runtime_context_attestation' "$name orchestrator skill must require signed child records"
    Require-Pattern $content 'source_context_attestations' "$name orchestrator skill must require signed independent sources"
    Require-Pattern $content 'evidence_attestation' "$name orchestrator skill must require signed child evidence"
}

$genericAuditTemplate = Read-WorkflowAsset 'docs/audits/templates/audit-record.md'
Require-Pattern $genericAuditTemplate 'governance_contract:\s*audit-loop/v3' 'generic audit template must use the current governance contract'
Require-Pattern $genericAuditTemplate 'workflow_contract_revision:\s*audit-runtime/v1' 'generic audit template must use the current workflow contract'
Require-Pattern $genericAuditTemplate 'runtime_context_attestation' 'generic audit template must bind a signed runtime context'
Require-Pattern $genericAuditTemplate 'evidence_attestation' 'generic audit template must bind externally signed evidence'

foreach ($name in @('backend-plan-audit-until-ready', 'backend-implement-audit-until-complete')) {
    $content = Read-WorkflowAsset ".agents/skills/$name/SKILL.md"
    Require-Pattern $content 'update-loop-run-state\.ps1' "$name skill must require persistent loop state"
    Require-Pattern $content 'generation' "$name skill must preserve generation CAS"
    Require-Pattern $content 'previous governance SHA' "$name skill must preserve previous governance revision CAS"
    Require-Pattern $content '(?s)update-loop-run-state\.ps1.*generation.*previous governance SHA.*child' "$name skill must require strictly serial children around persistent state updates"
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output 'Audit workflow contracts passed: atomic governance transactions, persistent CAS loop state, strict child serialization, durable handoffs, runtime/evidence attestations, isolated evidence execution, and governance history validation are present.'
