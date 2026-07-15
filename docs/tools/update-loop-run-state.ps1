param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Initialize', 'Update', 'Read')]
    [string]$Operation,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-z0-9][a-z0-9-]{7,127}$')]
    [string]$RunId,

    [ValidateSet('backend-plan-audit-until-ready', 'backend-implement-audit-until-complete')]
    [string]$Workflow,

    [string[]]$Target,
    [string[]]$PeerSet,
    [string[]]$AdvanceSet,

    [ValidateSet('standalone', 'child')]
    [string]$GoalMode = 'standalone',

    [ValidateSet('loop', 'single-transition')]
    [string]$StepMode = 'loop',

    [ValidateRange(1, 20)]
    [int]$MaxCycles = 8,

    [ValidateRange(1, 3)]
    [int]$MaxStagnantCycles = 2,

    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$PreviousGovernanceRevision,

    [int]$ExpectedGeneration = -1,

    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$ExpectedPreviousGovernanceRevision,

    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$NewPreviousGovernanceRevision,

    [ValidateRange(1, 20)]
    [int]$NewCycle,

    [string]$PlanStatesJson,
    [string]$RepositoryRoot
)

$ErrorActionPreference = 'Stop'

function Invoke-GitProbe([string[]]$Arguments) {
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& git -C $repoRoot @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    return [pscustomobject]@{ ExitCode = $exitCode; Output = $output }
}

function Get-NormalizedPlanSet([string[]]$Values, [string]$Label) {
    $normalized = @($Values | ForEach-Object { $_.Trim().ToUpperInvariant() } | Where-Object { $_ } | Sort-Object -Unique)
    if ($normalized.Count -eq 0) { throw "$Label must not be empty" }
    foreach ($value in $normalized) {
        if ($value -notmatch '^PLN-\d{4}$') { throw "Invalid $Label plan identifier: $value" }
    }
    return $normalized
}

function Test-SameSet([object[]]$Left, [object[]]$Right) {
    return (@($Left | Sort-Object -Unique) -join ',') -eq (@($Right | Sort-Object -Unique) -join ',')
}

function Write-StateAtomically([object]$State) {
    $json = $State | ConvertTo-Json -Depth 8
    $tempPath = "$statePath.$([guid]::NewGuid().ToString('N')).tmp"
    [IO.File]::WriteAllText($tempPath, $json + [Environment]::NewLine, (New-Object Text.UTF8Encoding($false)))
    Move-Item -LiteralPath $tempPath -Destination $statePath -Force
}

function Assert-GitRevision([string]$Revision, [switch]$MustBeHead) {
    $revisionResult = Invoke-GitProbe @('cat-file', '-e', "$Revision`^{commit}")
    if ($revisionResult.ExitCode -ne 0) { throw "Loop state revision does not exist: $Revision" }
    if ($MustBeHead) {
        $headResult = Invoke-GitProbe @('rev-parse', 'HEAD')
        $head = @($headResult.Output | Select-Object -First 1)[0].Trim()
        if ($headResult.ExitCode -ne 0 -or $head -ne $Revision.ToLowerInvariant()) {
            throw "Loop state revision must equal current HEAD: expected=$Revision actual=$head"
        }
    }
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}
$gitCommand = Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath($gitCommand.Source)
$repoPathPrefix = $repoRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
if ($gitExecutable.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-local Git binary: $gitExecutable"
}

function git {
    & $gitExecutable @args
}
$commonDirectoryResult = Invoke-GitProbe @('rev-parse', '--git-common-dir')
$commonDirectoryValue = @($commonDirectoryResult.Output | Select-Object -First 1)[0].Trim()
if ($commonDirectoryResult.ExitCode -ne 0) { throw 'Unable to resolve Git common directory for loop state' }
$commonDirectory = if ([IO.Path]::IsPathRooted($commonDirectoryValue)) { [IO.Path]::GetFullPath($commonDirectoryValue) } else { [IO.Path]::GetFullPath((Join-Path $repoRoot $commonDirectoryValue)) }
$stateDirectory = Join-Path $commonDirectory 'allinme-governance-loop-runs'
[IO.Directory]::CreateDirectory($stateDirectory) | Out-Null
$statePath = Join-Path $stateDirectory "$RunId.json"
$sha256 = [Security.Cryptography.SHA256]::Create()
try {
    $hashBytes = $sha256.ComputeHash([Text.Encoding]::UTF8.GetBytes("$($commonDirectory.ToLowerInvariant())|$RunId"))
} finally {
    $sha256.Dispose()
}
$lockHash = ([BitConverter]::ToString($hashBytes)).Replace('-', '').Substring(0, 16)
$mutex = New-Object Threading.Mutex($false, "allinme-loop-state-$lockHash")
$lockTaken = $false

try {
    $lockTaken = $mutex.WaitOne([TimeSpan]::FromSeconds(30))
    if (-not $lockTaken) { throw 'Timed out waiting for the loop state lock' }

    if ($Operation -eq 'Read') {
        if (-not (Test-Path -LiteralPath $statePath -PathType Leaf)) { throw "Loop state does not exist: $RunId" }
        Get-Content -Raw -Encoding UTF8 $statePath
        return
    }

    $normalizedTarget = Get-NormalizedPlanSet $Target 'TARGET'
    $normalizedPeerSet = Get-NormalizedPlanSet $PeerSet 'PEER_SET'
    $normalizedAdvanceSet = Get-NormalizedPlanSet $AdvanceSet 'ADVANCE_SET'
    if (@($normalizedTarget | Where-Object { $normalizedPeerSet -notcontains $_ }).Count -gt 0) { throw 'TARGET must be a subset of PEER_SET' }
    if (@($normalizedAdvanceSet | Where-Object { $normalizedTarget -notcontains $_ }).Count -gt 0) { throw 'ADVANCE_SET must be a subset of TARGET' }
    if ($GoalMode -eq 'standalone' -and -not (Test-SameSet $normalizedAdvanceSet $normalizedTarget)) { throw 'standalone ADVANCE_SET must equal TARGET' }
    if ($StepMode -eq 'single-transition' -and $MaxCycles -ne 1) { throw 'single-transition requires MaxCycles=1' }

    if ($Operation -eq 'Initialize') {
        if ([string]::IsNullOrWhiteSpace($Workflow) -or [string]::IsNullOrWhiteSpace($PreviousGovernanceRevision)) { throw 'Initialize requires Workflow and PreviousGovernanceRevision' }
        if (Test-Path -LiteralPath $statePath) { throw "Loop state already exists: $RunId" }
        Assert-GitRevision $PreviousGovernanceRevision -MustBeHead
        $plans = @($normalizedTarget | ForEach-Object {
            [ordered]@{ plan = $_; stage = 'pending'; fingerprint = 'pending'; stagnant_count = 0; blocker_code = 'none' }
        })
        $state = [ordered]@{
            schema = 'governance-loop-run/v1'
            run_id = $RunId
            workflow = $Workflow
            goal_mode = $GoalMode
            step_mode = $StepMode
            target = $normalizedTarget
            peer_set = $normalizedPeerSet
            advance_set = $normalizedAdvanceSet
            max_cycles = $MaxCycles
            max_stagnant_cycles = $MaxStagnantCycles
            generation = 0
            cycle = 0
            previous_governance_revision = $PreviousGovernanceRevision.ToLowerInvariant()
            plans = $plans
            updated_at = [DateTimeOffset]::UtcNow.ToString('o')
        }
        Write-StateAtomically $state
        $state | ConvertTo-Json -Compress -Depth 8
        return
    }

    if (-not (Test-Path -LiteralPath $statePath -PathType Leaf)) { throw "Loop state does not exist: $RunId" }
    $current = Get-Content -Raw -Encoding UTF8 $statePath | ConvertFrom-Json
    if ($current.schema -ne 'governance-loop-run/v1' -or $current.workflow -ne $Workflow) { throw 'Loop state workflow/schema mismatch' }
    if (-not (Test-SameSet $current.target $normalizedTarget) -or -not (Test-SameSet $current.peer_set $normalizedPeerSet) -or -not (Test-SameSet $current.advance_set $normalizedAdvanceSet)) { throw 'Loop state immutable set mismatch' }
    if ($current.goal_mode -ne $GoalMode -or $current.step_mode -ne $StepMode -or [int]$current.max_cycles -ne $MaxCycles -or [int]$current.max_stagnant_cycles -ne $MaxStagnantCycles) { throw 'Loop state immutable execution contract mismatch' }
    if ([int]$current.generation -ne $ExpectedGeneration) { throw "Loop state generation CAS failed: expected=$ExpectedGeneration actual=$($current.generation)" }
    if ([string]::IsNullOrWhiteSpace($ExpectedPreviousGovernanceRevision) -or [string]::IsNullOrWhiteSpace($NewPreviousGovernanceRevision) -or $NewCycle -lt 1) { throw 'Update requires both governance revisions and NewCycle' }
    if ($current.previous_governance_revision -ne $ExpectedPreviousGovernanceRevision.ToLowerInvariant()) { throw 'Loop state previous governance revision CAS failed' }
    if ($NewPreviousGovernanceRevision.ToLowerInvariant() -eq $ExpectedPreviousGovernanceRevision.ToLowerInvariant()) { throw 'Loop state governance revision must advance' }
    if ($NewCycle -ne ([int]$current.cycle + 1)) { throw "Loop state cycle must advance exactly once: current=$($current.cycle) new=$NewCycle" }
    if ($NewCycle -gt $MaxCycles) { throw 'Loop state cycle exceeds MaxCycles' }
    Assert-GitRevision $NewPreviousGovernanceRevision -MustBeHead
    $sharedGovernanceResult = Invoke-GitProbe @('rev-parse', '--verify', 'refs/allinme/governance-head')
    $sharedGovernanceRevision = @($sharedGovernanceResult.Output | Select-Object -First 1)
    if ($sharedGovernanceResult.ExitCode -ne 0 -or $sharedGovernanceRevision.Count -eq 0 -or $sharedGovernanceRevision[0].Trim() -ne $NewPreviousGovernanceRevision.ToLowerInvariant()) {
        throw 'Loop state update requires a governance revision produced by the shared transaction helper'
    }
    $ancestorResult = Invoke-GitProbe @('merge-base', '--is-ancestor', $ExpectedPreviousGovernanceRevision, $NewPreviousGovernanceRevision)
    if ($ancestorResult.ExitCode -ne 0) { throw 'New governance revision must descend from the previous governance revision' }
    if ([string]::IsNullOrWhiteSpace($PlanStatesJson)) { throw 'Update requires PlanStatesJson' }

    $incomingStates = @($PlanStatesJson | ConvertFrom-Json)
    if ($incomingStates.Count -ne $normalizedTarget.Count) { throw 'PlanStatesJson must contain every TARGET plan exactly once' }
    $currentByPlan = @{}
    foreach ($entry in @($current.plans)) { $currentByPlan[$entry.plan] = $entry }
    $seen = @{}
    $nextPlans = New-Object Collections.Generic.List[object]
    foreach ($entry in $incomingStates) {
        $plan = ([string]$entry.plan).ToUpperInvariant()
        if ($normalizedTarget -notcontains $plan -or $seen.ContainsKey($plan)) { throw "Invalid or duplicate plan state: $plan" }
        $seen[$plan] = $true
        $stage = [string]$entry.stage
        $fingerprint = [string]$entry.fingerprint
        $blockerCode = [string]$entry.blocker_code
        if ($stage -notmatch '^[A-Za-z0-9._:-]+$' -or [string]::IsNullOrWhiteSpace($fingerprint) -or $fingerprint.Contains("`n") -or $blockerCode -notmatch '^[A-Za-z0-9._:-]+$') { throw "Invalid canonical plan state for $plan" }
        $prior = $currentByPlan[$plan]
        if ($normalizedAdvanceSet -contains $plan) {
            $expectedStagnant = if ($prior.fingerprint -eq $fingerprint) { [int]$prior.stagnant_count + 1 } else { 0 }
            if ([int]$entry.stagnant_count -ne $expectedStagnant) { throw "Invalid stagnant_count for ${plan}: expected=$expectedStagnant" }
        } else {
            if ($stage -ne $prior.stage -or $fingerprint -ne $prior.fingerprint -or $blockerCode -ne $prior.blocker_code -or [int]$entry.stagnant_count -ne [int]$prior.stagnant_count) { throw "Non-ADVANCE_SET plan state changed: $plan" }
        }
        if ([int]$entry.stagnant_count -gt $MaxStagnantCycles) { throw "stagnant_count exceeds MaxStagnantCycles for $plan" }
        $nextPlans.Add([ordered]@{ plan = $plan; stage = $stage; fingerprint = $fingerprint; stagnant_count = [int]$entry.stagnant_count; blocker_code = $blockerCode })
    }

    $nextState = [ordered]@{
        schema = $current.schema
        run_id = $current.run_id
        workflow = $current.workflow
        goal_mode = $current.goal_mode
        step_mode = $current.step_mode
        target = @($current.target)
        peer_set = @($current.peer_set)
        advance_set = @($current.advance_set)
        max_cycles = [int]$current.max_cycles
        max_stagnant_cycles = [int]$current.max_stagnant_cycles
        generation = [int]$current.generation + 1
        cycle = $NewCycle
        previous_governance_revision = $NewPreviousGovernanceRevision.ToLowerInvariant()
        plans = $nextPlans.ToArray()
        updated_at = [DateTimeOffset]::UtcNow.ToString('o')
    }
    Write-StateAtomically $nextState
    $nextState | ConvertTo-Json -Compress -Depth 8
} finally {
    if ($lockTaken) { $mutex.ReleaseMutex() }
    $mutex.Dispose()
}
