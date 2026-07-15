$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$transactionScript = Join-Path $PSScriptRoot 'invoke-governance-transaction.ps1'
$reservationScript = Join-Path $PSScriptRoot 'reserve-governance-record.ps1'
$loopStateScript = Join-Path $PSScriptRoot 'update-loop-run-state.ps1'
$fixtureBase = Join-Path $repoRoot '.tmp'
$fixtureRoot = Join-Path $fixtureBase ('.governance-helpers-' + [Guid]::NewGuid().ToString('N'))
$gitExecutable = (Get-Command git -CommandType Application | Select-Object -First 1).Source

function Assert-True([bool]$Condition, [string]$Label) {
    if (-not $Condition) { throw "Assertion failed: $Label" }
}

function Assert-Equal([object]$Expected, [object]$Actual, [string]$Label) {
    if (-not [object]::Equals($Expected, $Actual)) {
        throw "Assertion failed: $Label; expected=<$Expected> actual=<$Actual>"
    }
}

function Assert-Throws([string]$Label, [string]$Pattern, [scriptblock]$Action) {
    try {
        & $Action | Out-Null
    } catch {
        $message = $_.Exception.Message
        if ($message -notmatch $Pattern) {
            throw "$Label failed with an unexpected error; expected=/$Pattern/ actual=<$message>"
        }
        return
    }
    throw "$Label unexpectedly succeeded"
}

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

function Invoke-GitProbe([string]$Repository, [string[]]$Arguments) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $gitExecutable -C $Repository @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    return [pscustomobject]@{ ExitCode = $exitCode; Output = $output }
}

function Get-GitValue([string]$Repository, [string[]]$Arguments) {
    return ([string](Invoke-Git $Repository $Arguments | Select-Object -First 1)).Trim()
}

function Get-Head([string]$Repository) {
    return Get-GitValue $Repository @('rev-parse', 'HEAD')
}

function Get-CommitCount([string]$Repository) {
    return [int](Get-GitValue $Repository @('rev-list', '--count', 'HEAD'))
}

function Assert-IndexEmpty([string]$Repository, [string]$Label) {
    $staged = @(Invoke-Git $Repository @('diff', '--cached', '--name-only', 'HEAD', '--'))
    Assert-Equal 0 $staged.Count $Label
}

function Assert-RefMissing([string]$Repository, [string]$Ref, [string]$Label) {
    $probe = Invoke-GitProbe $Repository @('rev-parse', '--verify', '--quiet', $Ref)
    Assert-True ($probe.ExitCode -ne 0) $Label
}

function New-TestRepository([string]$Name) {
    $path = Join-Path $fixtureRoot $Name
    [IO.Directory]::CreateDirectory($path) | Out-Null
    Invoke-Git $path @('init', '-q') | Out-Null
    Invoke-Git $path @('config', 'user.name', 'Governance Helpers Test') | Out-Null
    Invoke-Git $path @('config', 'user.email', 'governance-helpers@example.invalid') | Out-Null
    Invoke-Git $path @('config', 'commit.gpgSign', 'false') | Out-Null
    Invoke-Git $path @('config', 'core.autocrlf', 'false') | Out-Null
    foreach ($relativeDirectory in @(
        'docs\audits\records',
        'docs\remediations\records',
        'docs\implementations\records'
    )) {
        [IO.Directory]::CreateDirectory((Join-Path $path $relativeDirectory)) | Out-Null
    }
    Set-Utf8File (Join-Path $path 'README.md') "# governance helper fixture`n"
    Set-Utf8File (Join-Path $path 'docs\governance.txt') "base`n"
    Invoke-Git $path @('add', '--', 'README.md', 'docs/governance.txt') | Out-Null
    Invoke-Git $path @('commit', '-q', '-m', 'fixture base') | Out-Null
    return $path
}

function Invoke-GovernanceCommit([string]$Repository, [string]$ExpectedHead, [string[]]$Paths, [string]$Message) {
    $output = @(& $transactionScript `
        -RepositoryRoot $Repository `
        -ExpectedHead $ExpectedHead `
        -Paths $Paths `
        -Message $Message)
    return (($output | Select-Object -Last 1) | ConvertFrom-Json)
}

function Get-Reservation([string]$Repository, [string]$Kind, [string]$Suffix) {
    $line = [string](@(& $reservationScript -RepositoryRoot $Repository -Kind $Kind -Suffix $Suffix) | Select-Object -Last 1)
    $parts = @($line -split "`t", 2)
    if ($parts.Count -ne 2) { throw "Reservation helper returned malformed output: $line" }
    return [pscustomobject]@{ Id = $parts[0]; Path = $parts[1] }
}

function Read-LoopState([string]$Repository, [string]$RunId) {
    return [string](@(& $loopStateScript -Operation Read -RepositoryRoot $Repository -RunId $RunId) | Select-Object -Last 1)
}

function Assert-RejectedWithoutLoopMutation(
    [string]$Label,
    [string]$Pattern,
    [string]$Repository,
    [string]$RunId,
    [scriptblock]$Action
) {
    $before = Read-LoopState $Repository $RunId
    Assert-Throws $Label $Pattern $Action
    $after = Read-LoopState $Repository $RunId
    Assert-Equal $before $after "$Label must not mutate persisted state"
}

try {
    [IO.Directory]::CreateDirectory($fixtureBase) | Out-Null
    [IO.Directory]::CreateDirectory($fixtureRoot) | Out-Null

    # Exact-path transaction success and the shared governance head.
    $transactionRepo = New-TestRepository 'transaction-success'
    $baseHead = Get-Head $transactionRepo
    Set-Utf8File (Join-Path $transactionRepo 'docs\governance.txt') "base`nexact transaction`n"
    $transaction = Invoke-GovernanceCommit $transactionRepo $baseHead @('docs/governance.txt') 'test exact governance transaction'
    $newHead = Get-Head $transactionRepo
    Assert-True ($newHead -ne $baseHead) 'exact transaction must advance HEAD'
    Assert-Equal $newHead ([string]$transaction.governance_revision) 'transaction JSON revision'
    Assert-Equal $baseHead ([string]$transaction.parent_revision) 'transaction JSON parent'
    Assert-Equal 'docs/governance.txt' ([string](@($transaction.paths) -join ',')) 'transaction JSON exact paths'
    Assert-Equal $baseHead (Get-GitValue $transactionRepo @('rev-parse', 'HEAD^')) 'transaction commit parent'
    Assert-Equal 'docs/governance.txt' ([string](@(Invoke-Git $transactionRepo @('diff-tree', '--no-commit-id', '--name-only', '-r', 'HEAD')) -join ',')) 'committed exact path set'
    Assert-Equal $newHead (Get-GitValue $transactionRepo @('rev-parse', 'refs/allinme/governance-head')) 'shared governance ref advances'
    Assert-Equal '' ([string](@(Invoke-Git $transactionRepo @('status', '--porcelain=v1', '--untracked-files=all')) -join "`n")) 'successful transaction leaves clean worktree'

    # An unrelated dirty path must reject the whole transaction without a commit.
    $dirtyRepo = New-TestRepository 'transaction-dirty'
    $dirtyHead = Get-Head $dirtyRepo
    $dirtyCount = Get-CommitCount $dirtyRepo
    Set-Utf8File (Join-Path $dirtyRepo 'docs\governance.txt') "base`nallowed`n"
    Set-Utf8File (Join-Path $dirtyRepo 'README.md') "# unrelated dirty change`n"
    Assert-Throws 'unrelated dirty path' 'refuses unrelated user or parallel changes' {
        & $transactionScript -RepositoryRoot $dirtyRepo -ExpectedHead $dirtyHead -Paths @('docs/governance.txt') -Message 'must reject dirty path'
    }
    Assert-Equal $dirtyHead (Get-Head $dirtyRepo) 'dirty rejection preserves HEAD'
    Assert-Equal $dirtyCount (Get-CommitCount $dirtyRepo) 'dirty rejection creates no reachable commit'
    Assert-IndexEmpty $dirtyRepo 'dirty rejection preserves empty index'
    Assert-RefMissing $dirtyRepo 'refs/allinme/governance-head' 'dirty rejection must not create governance ref'

    # Any pre-staged content must reject before the helper rewrites the index.
    $stagedRepo = New-TestRepository 'transaction-staged'
    $stagedHead = Get-Head $stagedRepo
    $stagedCount = Get-CommitCount $stagedRepo
    Set-Utf8File (Join-Path $stagedRepo 'docs\governance.txt') "base`nstaged`n"
    Invoke-Git $stagedRepo @('add', '--', 'docs/governance.txt') | Out-Null
    Assert-Throws 'pre-staged transaction' 'requires an empty Git index' {
        & $transactionScript -RepositoryRoot $stagedRepo -ExpectedHead $stagedHead -Paths @('docs/governance.txt') -Message 'must reject staged content'
    }
    Assert-Equal $stagedHead (Get-Head $stagedRepo) 'staged rejection preserves HEAD'
    Assert-Equal $stagedCount (Get-CommitCount $stagedRepo) 'staged rejection creates no reachable commit'
    Assert-Equal 'docs/governance.txt' ([string](@(Invoke-Git $stagedRepo @('diff', '--cached', '--name-only', 'HEAD', '--')) -join ',')) 'staged rejection preserves caller index'
    Assert-RefMissing $stagedRepo 'refs/allinme/governance-head' 'staged rejection must not create governance ref'

    # A stale caller HEAD must fail its compare-and-swap before creating a governance ref.
    $staleRepo = New-TestRepository 'transaction-stale-head'
    $staleHead = Get-Head $staleRepo
    Set-Utf8File (Join-Path $staleRepo 'README.md') "# committed after caller snapshot`n"
    Invoke-Git $staleRepo @('add', '--', 'README.md') | Out-Null
    Invoke-Git $staleRepo @('commit', '-q', '-m', 'advance outside helper') | Out-Null
    $actualHead = Get-Head $staleRepo
    $staleCount = Get-CommitCount $staleRepo
    Set-Utf8File (Join-Path $staleRepo 'docs\governance.txt') "base`nstale caller`n"
    Assert-Throws 'stale ExpectedHead' 'HEAD CAS failed' {
        & $transactionScript -RepositoryRoot $staleRepo -ExpectedHead $staleHead -Paths @('docs/governance.txt') -Message 'must reject stale head'
    }
    Assert-Equal $actualHead (Get-Head $staleRepo) 'stale HEAD rejection preserves current HEAD'
    Assert-Equal $staleCount (Get-CommitCount $staleRepo) 'stale HEAD rejection creates no reachable commit'
    Assert-IndexEmpty $staleRepo 'stale HEAD rejection preserves empty index'
    Assert-RefMissing $staleRepo 'refs/allinme/governance-head' 'stale HEAD rejection must not create governance ref'

    # A shared governance ref that is ahead of the caller snapshot must not be forked.
    $sharedRepo = New-TestRepository 'transaction-shared-ref'
    $sharedHead = Get-Head $sharedRepo
    $sharedTree = Get-GitValue $sharedRepo @('rev-parse', 'HEAD^{tree}')
    $sharedChild = Get-GitValue $sharedRepo @('commit-tree', $sharedTree, '-p', $sharedHead, '-m', 'concurrent governance child')
    Invoke-Git $sharedRepo @('update-ref', 'refs/allinme/governance-head', $sharedChild) | Out-Null
    Set-Utf8File (Join-Path $sharedRepo 'docs\governance.txt') "base`nshared ref stale`n"
    Assert-Throws 'stale shared governance ref' 'would fork the shared governance chain' {
        & $transactionScript -RepositoryRoot $sharedRepo -ExpectedHead $sharedHead -Paths @('docs/governance.txt') -Message 'must reject shared ref fork'
    }
    Assert-Equal $sharedHead (Get-Head $sharedRepo) 'shared ref rejection preserves HEAD'
    Assert-Equal $sharedChild (Get-GitValue $sharedRepo @('rev-parse', 'refs/allinme/governance-head')) 'shared ref rejection preserves concurrent value'
    Assert-IndexEmpty $sharedRepo 'shared ref rejection restores empty index'

    # Ref lock contention exercises the atomic update-ref CAS failure path.
    $casRepo = New-TestRepository 'transaction-ref-cas'
    $casHead = Get-Head $casRepo
    Invoke-Git $casRepo @('update-ref', 'refs/allinme/governance-head', $casHead) | Out-Null
    $gitDirectoryValue = Get-GitValue $casRepo @('rev-parse', '--git-dir')
    $gitDirectory = if ([IO.Path]::IsPathRooted($gitDirectoryValue)) { [IO.Path]::GetFullPath($gitDirectoryValue) } else { [IO.Path]::GetFullPath((Join-Path $casRepo $gitDirectoryValue)) }
    $sharedRefLock = Join-Path $gitDirectory 'refs\allinme\governance-head.lock'
    Set-Utf8File $sharedRefLock "held by governance helper test`n"
    Set-Utf8File (Join-Path $casRepo 'docs\governance.txt') "base`nCAS contention`n"
    try {
        Assert-Throws 'shared ref CAS lock contention' 'Governance transaction ref CAS failed' {
            & $transactionScript -RepositoryRoot $casRepo -ExpectedHead $casHead -Paths @('docs/governance.txt') -Message 'must reject shared ref CAS contention'
        }
    } finally {
        if (Test-Path -LiteralPath $sharedRefLock -PathType Leaf) { Remove-Item -LiteralPath $sharedRefLock -Force }
    }
    Assert-Equal $casHead (Get-Head $casRepo) 'shared ref CAS failure preserves HEAD atomically'
    Assert-Equal $casHead (Get-GitValue $casRepo @('rev-parse', 'refs/allinme/governance-head')) 'shared ref CAS failure preserves governance ref'
    Assert-IndexEmpty $casRepo 'shared ref CAS failure restores empty index'

    # Record allocation is monotonic within one Git common directory and isolated across repositories.
    $allocatorA = New-TestRepository 'allocator-a'
    $allocatorB = New-TestRepository 'allocator-b'
    $reservationA1 = Get-Reservation $allocatorA 'AUD' '20260715-first'
    $reservationA2 = Get-Reservation $allocatorA 'AUD' '20260715-second'
    $reservationB1 = Get-Reservation $allocatorB 'AUD' '20260715-isolated'
    Assert-Equal 'AUD-0001' $reservationA1.Id 'first allocator value'
    Assert-Equal 'AUD-0002' $reservationA2.Id 'allocator monotonic increment'
    Assert-Equal 'AUD-0001' $reservationB1.Id 'allocator repository isolation'
    Assert-True (Test-Path -LiteralPath (Join-Path $allocatorA $reservationA1.Path) -PathType Leaf) 'first reservation creates exact record file'
    Assert-True (Test-Path -LiteralPath (Join-Path $allocatorA $reservationA2.Path) -PathType Leaf) 'second reservation creates exact record file'
    Assert-True (Test-Path -LiteralPath (Join-Path $allocatorB $reservationB1.Path) -PathType Leaf) 'isolated reservation creates exact record file'
    $allocatorStatus = [string](@(Invoke-Git $allocatorA @('status', '--porcelain=v1', '--untracked-files=all')) -join "`n")
    Assert-True ($allocatorStatus -notmatch 'allinme-governance-reservations') 'allocator lock metadata stays outside the worktree'

    $nestedRoot = Join-Path $allocatorA 'nested-root'
    [IO.Directory]::CreateDirectory((Join-Path $nestedRoot 'docs\audits\records')) | Out-Null
    Assert-Throws 'non-top-level allocator root' 'RepositoryRoot must be the Git top-level directory' {
        & $reservationScript -RepositoryRoot $nestedRoot -Kind AUD -Suffix '20260715-nested-root'
    }

    # Loop state Initialize/Read/Update persistence and CAS/invariant rejection.
    $loopRepo = New-TestRepository 'loop-state'
    $runId = 'loop-run-' + [Guid]::NewGuid().ToString('N')
    $loopHead0 = Get-Head $loopRepo
    $loopContract = @{
        RepositoryRoot = $loopRepo
        RunId = $runId
        Workflow = 'backend-plan-audit-until-ready'
        Target = @('PLN-0001')
        PeerSet = @('PLN-0001')
        AdvanceSet = @('PLN-0001')
        GoalMode = 'standalone'
        StepMode = 'loop'
        MaxCycles = 4
        MaxStagnantCycles = 1
    }
    $initializeOutput = @(& $loopStateScript -Operation Initialize @loopContract -PreviousGovernanceRevision $loopHead0)
    $initialized = ($initializeOutput | Select-Object -Last 1) | ConvertFrom-Json
    Assert-Equal 0 ([int]$initialized.generation) 'initialize generation'
    Assert-Equal 0 ([int]$initialized.cycle) 'initialize cycle'
    Assert-Equal $loopHead0 ([string]$initialized.previous_governance_revision) 'initialize previous governance revision'
    $readInitialized = (Read-LoopState $loopRepo $runId) | ConvertFrom-Json
    Assert-Equal $runId ([string]$readInitialized.run_id) 'Read returns initialized state'
    Assert-Equal 'PLN-0001' ([string](@($readInitialized.target) -join ',')) 'Read preserves normalized target'

    Set-Utf8File (Join-Path $loopRepo 'docs\governance.txt') "base`nloop governance 1`n"
    $loopTransaction1 = Invoke-GovernanceCommit $loopRepo $loopHead0 @('docs/governance.txt') 'loop governance revision 1'
    $loopHead1 = [string]$loopTransaction1.governance_revision
    $planState1 = ([ordered]@{
        plan = 'PLN-0001'
        stage = 'plan-audit'
        fingerprint = 'fingerprint-1'
        stagnant_count = 0
        blocker_code = 'none'
    } | ConvertTo-Json -Compress)
    $updateOutput1 = @(& $loopStateScript `
        -Operation Update `
        @loopContract `
        -ExpectedGeneration 0 `
        -ExpectedPreviousGovernanceRevision $loopHead0 `
        -NewPreviousGovernanceRevision $loopHead1 `
        -NewCycle 1 `
        -PlanStatesJson $planState1)
    $updated1 = ($updateOutput1 | Select-Object -Last 1) | ConvertFrom-Json
    Assert-Equal 1 ([int]$updated1.generation) 'Update increments generation'
    Assert-Equal 1 ([int]$updated1.cycle) 'Update advances cycle exactly once'
    Assert-Equal $loopHead1 ([string]$updated1.previous_governance_revision) 'Update persists new governance revision'
    $persisted1 = (Read-LoopState $loopRepo $runId) | ConvertFrom-Json
    Assert-Equal 1 ([int]$persisted1.generation) 'Read observes persisted generation'
    Assert-Equal 'fingerprint-1' ([string]@($persisted1.plans)[0].fingerprint) 'Read observes persisted plan state'

    Assert-RejectedWithoutLoopMutation 'stale loop generation' 'generation CAS failed' $loopRepo $runId {
        & $loopStateScript -Operation Update @loopContract -ExpectedGeneration 0 -ExpectedPreviousGovernanceRevision $loopHead1 -NewPreviousGovernanceRevision $loopHead0 -NewCycle 2 -PlanStatesJson $planState1
    }
    Assert-RejectedWithoutLoopMutation 'stale loop previous SHA' 'previous governance revision CAS failed' $loopRepo $runId {
        & $loopStateScript -Operation Update @loopContract -ExpectedGeneration 1 -ExpectedPreviousGovernanceRevision $loopHead0 -NewPreviousGovernanceRevision $loopHead1 -NewCycle 2 -PlanStatesJson $planState1
    }
    $changedLoopContract = @{
        RepositoryRoot = $loopRepo
        RunId = $runId
        Workflow = 'backend-plan-audit-until-ready'
        Target = @('PLN-0001', 'PLN-0002')
        PeerSet = @('PLN-0001', 'PLN-0002')
        AdvanceSet = @('PLN-0001', 'PLN-0002')
        GoalMode = 'standalone'
        StepMode = 'loop'
        MaxCycles = 4
        MaxStagnantCycles = 1
    }
    Assert-RejectedWithoutLoopMutation 'loop immutable set drift' 'immutable set mismatch' $loopRepo $runId {
        & $loopStateScript -Operation Update @changedLoopContract -ExpectedGeneration 1 -ExpectedPreviousGovernanceRevision $loopHead1 -NewPreviousGovernanceRevision $loopHead0 -NewCycle 2 -PlanStatesJson $planState1
    }

    Set-Utf8File (Join-Path $loopRepo 'docs\governance.txt') "base`nloop governance 1`nloop governance 2`n"
    $loopTransaction2 = Invoke-GovernanceCommit $loopRepo $loopHead1 @('docs/governance.txt') 'loop governance revision 2'
    $loopHead2 = [string]$loopTransaction2.governance_revision
    $invalidStagnation = ([ordered]@{
        plan = 'PLN-0001'
        stage = 'plan-audit'
        fingerprint = 'fingerprint-1'
        stagnant_count = 0
        blocker_code = 'none'
    } | ConvertTo-Json -Compress)
    Assert-RejectedWithoutLoopMutation 'forged loop stagnation count' 'Invalid stagnant_count' $loopRepo $runId {
        & $loopStateScript -Operation Update @loopContract -ExpectedGeneration 1 -ExpectedPreviousGovernanceRevision $loopHead1 -NewPreviousGovernanceRevision $loopHead2 -NewCycle 2 -PlanStatesJson $invalidStagnation
    }

    $validStagnation = ([ordered]@{
        plan = 'PLN-0001'
        stage = 'plan-audit'
        fingerprint = 'fingerprint-1'
        stagnant_count = 1
        blocker_code = 'none'
    } | ConvertTo-Json -Compress)
    & $loopStateScript -Operation Update @loopContract -ExpectedGeneration 1 -ExpectedPreviousGovernanceRevision $loopHead1 -NewPreviousGovernanceRevision $loopHead2 -NewCycle 2 -PlanStatesJson $validStagnation | Out-Null
    $persisted2 = (Read-LoopState $loopRepo $runId) | ConvertFrom-Json
    Assert-Equal 2 ([int]$persisted2.generation) 'valid post-rejection update increments generation'
    Assert-Equal $loopHead2 ([string]$persisted2.previous_governance_revision) 'valid post-rejection update persists previous SHA'
    Assert-Equal 1 ([int]@($persisted2.plans)[0].stagnant_count) 'valid stagnation count persists'

    Set-Utf8File (Join-Path $loopRepo 'docs\governance.txt') "base`nloop governance 1`nloop governance 2`nloop governance 3`n"
    $loopTransaction3 = Invoke-GovernanceCommit $loopRepo $loopHead2 @('docs/governance.txt') 'loop governance revision 3'
    $loopHead3 = [string]$loopTransaction3.governance_revision
    $stagnationOverLimit = ([ordered]@{
        plan = 'PLN-0001'
        stage = 'plan-audit'
        fingerprint = 'fingerprint-1'
        stagnant_count = 2
        blocker_code = 'none'
    } | ConvertTo-Json -Compress)
    Assert-RejectedWithoutLoopMutation 'loop stagnation limit' 'stagnant_count exceeds MaxStagnantCycles' $loopRepo $runId {
        & $loopStateScript -Operation Update @loopContract -ExpectedGeneration 2 -ExpectedPreviousGovernanceRevision $loopHead2 -NewPreviousGovernanceRevision $loopHead3 -NewCycle 3 -PlanStatesJson $stagnationOverLimit
    }

    Write-Output 'Governance helper tests passed: exact transactions and CAS, isolated monotonic allocation, and persistent loop-state invariants.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot) {
        $resolvedFixtureRoot = (Resolve-Path $fixtureRoot).Path
        $resolvedFixtureBase = (Resolve-Path $fixtureBase).Path
        $allowedPrefix = $resolvedFixtureBase.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar + '.governance-helpers-'
        if (-not $resolvedFixtureRoot.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected governance helper fixture path: $resolvedFixtureRoot"
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
