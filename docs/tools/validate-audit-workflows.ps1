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

foreach ($name in $recordCreators) {
    $content = Read-WorkflowAsset ".github/prompts/$name.prompt.md"
    Require-Pattern $content 'governance-handoff-contract:' "$name must declare the governance handoff contract"
    Require-Pattern $content 'open checkpoint' "$name must commit an open checkpoint before subject or evidence work"
    Require-Pattern $content 'governance-handoff-contract:.*reuse-existing-checkpoint;.*no-empty-commit;' "$name must reuse committed checkpoints without empty commits"
    Require-Pattern $content 'terminal governance commit' "$name must commit its terminal governance transition"
    Require-Pattern $content 'governance_revision' "$name must return a clean terminal governance_revision"
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
    Require-Pattern $content 'invoke-revision-evidence\.ps1' "$name must execute subject evidence at the exact revision"
}

$planAudit = Read-WorkflowAsset '.github/prompts/backend-plan-audit.prompt.md'
Require-Pattern $planAudit 'PEER_SET' 'backend-plan-audit must accept PEER_SET'
Require-Pattern $planAudit 'peer-set-contract:\s*target-subset-of-peer-set;\s*audit-target-only;\s*inspect-complete-peer-set;\s*persist-peer-snapshot' 'backend-plan-audit must persist the complete peer snapshot'
Require-Pattern $planAudit 'audited_peer_plans' 'backend-plan-audit must record the complete peer set'
Require-Pattern $planAudit 'invoke-revision-evidence\.ps1' 'backend-plan-audit must run evidence at the exact revision'

$planLoop = Read-WorkflowAsset '.github/prompts/backend-plan-audit-until-ready.prompt.md'
Require-Pattern $planLoop 'plan-loop-contract:\s*immutable-target-set;\s*separate-advance-set;\s*set-aware-plan-audit;\s*verification-before-remediation;\s*per-plan-terminal-state' 'plan loop must separate peer and advance sets'
Require-Pattern $planLoop 'argument-hint:.*MAX_CYCLES=8' 'plan loop default MAX_CYCLES must cover the normal audit/remediation/follow-up/acceptance path'
Require-Pattern $planLoop 'STEP_MODE=loop\|single-transition' 'plan loop must expose single-transition child mode'
Require-Pattern $planLoop 'single-transition.*MAX_CYCLES=1' 'plan loop single-transition mode must force one cycle'
Require-Pattern $planLoop 'peer-routing-contract:\s*target-is-complete-peer-set;\s*advance-set-is-subset;\s*plan-audit-target-is-drifted-subset' 'plan loop must preserve complete peers while auditing only the drifted subset'
Require-Pattern $planLoop 'ADVANCE_SET' 'plan loop must use an explicit advance subset'
Require-Pattern $planLoop 'context-dispatch-contract:\s*independent-stages-require-new-runtime-task;\s*runtime-ref-required;\s*uuid-is-not-isolation' 'plan loop must require real runtime task isolation'
Require-Pattern $planLoop 'governance-handoff-contract:\s*child-must-return-clean-terminal-governance-revision' 'plan loop must require clean child governance handoff'
Require-Pattern $planLoop 'stable-state-fingerprint' 'plan loop must define deterministic per-plan stagnation state'
Require-Pattern $planLoop 'reuse-current-ready' 'plan loop must reuse a current closed ready acceptance'
Require-Pattern $planLoop 'peer_reaudit_required' 'plan loop must persistently route affected peers back to audit'

$implementationLoop = Read-WorkflowAsset '.github/prompts/backend-implement-audit-until-complete.prompt.md'
Require-Pattern $implementationLoop 'argument-hint:.*MAX_CYCLES=12' 'implementation loop default MAX_CYCLES must cover readiness, implementation, audit and acceptance'
Require-Pattern $implementationLoop 'orchestration-step-contract:\s*one-durable-transition-per-plan-per-cycle;\s*nested-loop-forbidden' 'implementation loop must forbid nested multi-cycle execution'
Require-Pattern $implementationLoop 'GOAL_MODE=child\s+STEP_MODE=single-transition\s+MAX_CYCLES=1' 'implementation loop must call readiness as a single-transition child'
Require-Pattern $implementationLoop 'peer-routing-contract:\s*target-is-complete-peer-set;\s*readiness-advance-set-is-subset' 'implementation loop must preserve the complete peer set while advancing a readiness subset'
Require-Pattern $implementationLoop 'context-dispatch-contract:\s*independent-stages-require-new-runtime-task;\s*runtime-ref-required;\s*uuid-is-not-isolation' 'implementation loop must require real runtime task isolation'
Require-Pattern $implementationLoop 'governance-handoff-contract:\s*child-must-return-clean-terminal-governance-revision' 'implementation loop must require clean child governance handoff'
Require-Pattern $implementationLoop 'stable-state-fingerprint' 'implementation loop must define deterministic per-plan stagnation state'
Require-Pattern $implementationLoop 'CONTEXT_REF=<child runtime ref>' 'implementation loop must propagate runtime context references to independent children'

$skillRequirements = @{
    'backend-plan-audit-until-ready' = @('ADVANCE_SET', 'PEER_SET', 'STEP_MODE=single-transition', 'CONTEXT_REF', 'governance_revision')
    'backend-implement-audit-until-complete' = @('ADVANCE_SET', 'STEP_MODE=single-transition', 'CONTEXT_REF', 'governance_revision')
    'backend-plan-audit' = @('PEER_SET', 'persist the peer set', 'invoke-revision-evidence.ps1', 'terminal governance')
    'backend-plan-acceptance-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-implementation-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-implementation-acceptance-audit' = @('task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
    'backend-follow-up-audit' = @('new task/agent', 'CONTEXT_REF', 'detached evidence runner', 'governance_revision')
}

foreach ($entry in $skillRequirements.GetEnumerator()) {
    $content = Read-WorkflowAsset ".agents/skills/$($entry.Key)/SKILL.md"
    Require-Pattern $content "(?m)^name:\s*$([regex]::Escape($entry.Key))\s*$" "$($entry.Key) skill metadata must match its folder"
    Require-Pattern $content "\.github/prompts/$([regex]::Escape($entry.Key))\.prompt\.md" "$($entry.Key) skill must bind its canonical prompt"
    foreach ($requiredText in $entry.Value) {
        Require-Pattern $content ([regex]::Escape($requiredText)) "$($entry.Key) skill must preserve canonical workflow requirement: $requiredText"
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output 'Audit workflow contracts passed: durable governance handoffs, runtime context isolation, adequate cycle budgets, single-transition child routing, and peer/advance set separation are present.'
