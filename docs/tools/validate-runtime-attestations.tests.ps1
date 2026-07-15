$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$validator = Join-Path $PSScriptRoot 'validate-runtime-attestations.ps1'
$fixtureBase = Join-Path $repoRoot '.tmp'
$fixtureRoot = Join-Path $fixtureBase ('.validate-runtime-attestations-' + [Guid]::NewGuid().ToString('N'))
$git = Get-Command git -ErrorAction Stop
$openssl = Get-Command openssl -ErrorAction Stop
$shell = Get-Command pwsh -ErrorAction SilentlyContinue
if ($null -eq $shell) {
    $shell = Get-Command powershell.exe -ErrorAction SilentlyContinue
}
if ($null -eq $shell) {
    $shell = Get-Command powershell -ErrorAction Stop
}

$originUrl = 'https://example.invalid/allinme/runtime-attestation-fixture.git'
$privateKeyPath = Join-Path $fixtureRoot 'runtime-private.pem'
$publicKeyPath = Join-Path $fixtureRoot 'runtime-public.pem'
$trustedKeySha256 = $null
$historyBase = $null

$sourceRecordRelativePath = 'docs/audits/records/AUD-0002-20260715-runtime-source.md'
$acceptanceRecordRelativePath = 'docs/audits/records/AUD-0003-20260715-runtime-acceptance.md'
$sourceRecordPath = Join-Path $fixtureRoot $sourceRecordRelativePath
$acceptanceRecordPath = Join-Path $fixtureRoot $acceptanceRecordRelativePath
$sourceExecutionContextId = '30000000-0000-4000-8000-000000000001'
$acceptanceExecutionContextId = '30000000-0000-4000-8000-000000000002'
$wrongExecutionContextId = '30000000-0000-4000-8000-000000000003'
$sourceRuntimeContextRef = 'runtime://source-task'
$acceptanceRuntimeContextRef = 'runtime://acceptance-task'
$sourceAttestationId = '20000000-0000-4000-8000-000000000001'
$acceptanceAttestationId = '20000000-0000-4000-8000-000000000002'
$sourceAttestationPath = "docs/evidence/runtime-attestations/$sourceAttestationId.json"
$acceptanceAttestationPath = "docs/evidence/runtime-attestations/$acceptanceAttestationId.json"
$sourceScope = 'plan:PLN-0001'
$acceptanceScope = 'plan:PLN-0001'
$sourceTaskId = 'task-source'
$acceptanceTaskId = 'task-acceptance'
$testIssuedAt = [DateTimeOffset]::UtcNow.AddMinutes(-5).ToString('o')
$testExpiresAt = [DateTimeOffset]::UtcNow.AddMinutes(55).ToString('o')

function Set-Utf8File([string]$Path, [string]$Content) {
    $parent = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    $normalized = $Content.Replace("`r`n", "`n").Replace("`r", "`n").Replace("`n", [Environment]::NewLine)
    [IO.File]::WriteAllText($Path, $normalized, [Text.UTF8Encoding]::new($false))
}

function Invoke-Git([string[]]$Arguments) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $git.Source -C $fixtureRoot @Arguments 2>&1)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    if ($exitCode -ne 0) {
        throw "git $($Arguments -join ' ') failed: $($output -join [Environment]::NewLine)"
    }
    return $output
}

function Invoke-OpenSsl([string[]]$Arguments) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = @(& $openssl.Source @Arguments 2>&1)
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    if ($exitCode -ne 0) {
        throw "openssl $($Arguments -join ' ') failed: $($output -join [Environment]::NewLine)"
    }
    return $output
}

function ConvertTo-ProcessArgument([string]$Value) {
    if ($Value -notmatch '[\s"]') {
        return $Value
    }
    return '"' + $Value.Replace('"', '\"') + '"'
}

function Invoke-FixtureValidator([switch]$OmitTrustAnchor) {
    $environmentNames = @(
        'AUDIT_HISTORY_BASE',
        'AUDIT_RUNTIME_PUBLIC_KEY_PATH',
        'AUDIT_RUNTIME_PUBLIC_KEY_BASE64',
        'AUDIT_RUNTIME_TRUSTED_KEY_SHA256'
    )
    $savedEnvironment = @{}
    foreach ($name in $environmentNames) {
        $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
        [Environment]::SetEnvironmentVariable($name, $null, 'Process')
    }

    try {
        $arguments = @(
            '-NoProfile',
            '-ExecutionPolicy', 'Bypass',
            '-File', $validator,
            '-RepositoryRoot', $fixtureRoot,
            '-HistoryBase', $historyBase,
            '-PublicKeyPath', $publicKeyPath
        )
        if (-not $OmitTrustAnchor) {
            $arguments += @('-TrustedKeySha256', $trustedKeySha256)
        }

        $startInfo = New-Object Diagnostics.ProcessStartInfo
        $startInfo.FileName = $shell.Source
        $startInfo.Arguments = (($arguments | ForEach-Object { ConvertTo-ProcessArgument ([string]$_) }) -join ' ')
        $startInfo.UseShellExecute = $false
        $startInfo.CreateNoWindow = $true
        $startInfo.RedirectStandardOutput = $true
        $startInfo.RedirectStandardError = $true
        $process = New-Object Diagnostics.Process
        $process.StartInfo = $startInfo
        if (-not $process.Start()) {
            throw "Could not start PowerShell validator process: $($shell.Source)"
        }
        try {
            $stdoutTask = $process.StandardOutput.ReadToEndAsync()
            $stderrTask = $process.StandardError.ReadToEndAsync()
            $process.WaitForExit()
            $standardOutput = $stdoutTask.GetAwaiter().GetResult()
            $standardError = $stderrTask.GetAwaiter().GetResult()
            $exitCode = $process.ExitCode
        } finally {
            $process.Dispose()
        }
        $output = @($standardOutput, $standardError) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
        return @{
            ExitCode = $exitCode
            Output = ($output | Out-String).Trim()
        }
    } finally {
        foreach ($name in $environmentNames) {
            [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process')
        }
    }
}

function Assert-Pass([string]$Label) {
    $result = Invoke-FixtureValidator
    if ($result.ExitCode -ne 0) {
        throw "$Label unexpectedly failed: $($result.Output)"
    }
}

function Assert-Fail([string]$Label, [string]$Pattern, [switch]$OmitTrustAnchor) {
    $result = Invoke-FixtureValidator -OmitTrustAnchor:$OmitTrustAnchor
    $compactOutput = $result.Output -replace '\s+', ''
    $compactPattern = $Pattern -replace '\s+', ''
    if ($result.ExitCode -eq 0 -or $compactOutput -notmatch $compactPattern) {
        throw "$Label was not rejected with /$Pattern/: $($result.Output)"
    }
}

function Get-AuditRecord(
    [string]$AuditId,
    [string]$Schema,
    [string]$AuditType,
    [string]$ExecutionContextId,
    [string]$RuntimeContextRef,
    [string]$RuntimeAttestation,
    [string]$Scope,
    [string]$BaselineRevision,
    [string]$IndependenceBasis,
    [string]$SourceContextIds,
    [string]$SourceContextRefs,
    [string]$SourceContextAttestations
) {
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add('---')
    $lines.Add('status: closed')
    $lines.Add('governance_contract: audit-loop/v3')
    $lines.Add('workflow_contract_revision: audit-runtime/v1')
    $lines.Add("audit_schema: $Schema")
    $lines.Add("audit_id: $AuditId")
    $lines.Add('auditor: runtime-validator-test')
    $lines.Add("execution_context_id: $ExecutionContextId")
    $lines.Add("runtime_context_ref: $RuntimeContextRef")
    if (-not [string]::IsNullOrWhiteSpace($RuntimeAttestation)) {
        $lines.Add("runtime_context_attestation: $RuntimeAttestation")
    }
    if (-not [string]::IsNullOrWhiteSpace($SourceContextIds)) {
        $lines.Add("source_context_ids: $SourceContextIds")
    }
    if (-not [string]::IsNullOrWhiteSpace($SourceContextRefs)) {
        $lines.Add("source_context_refs: $SourceContextRefs")
    }
    if (-not [string]::IsNullOrWhiteSpace($SourceContextAttestations)) {
        $lines.Add("source_context_attestations: $SourceContextAttestations")
    }
    $lines.Add("audit_type: $AuditType")
    $lines.Add("independence_basis: $IndependenceBasis")
    $lines.Add("scope: $Scope")
    $lines.Add('subject: runtime attestation validator fixture')
    $lines.Add("baseline: git:$BaselineRevision; worktree:clean")
    $lines.Add('started_at: 2026-07-15T00:00:00Z')
    $lines.Add('completed_at: 2026-07-15T00:05:00Z')
    $lines.Add('last_updated: 2026-07-15')
    $lines.Add('---')
    $lines.Add('')
    $lines.Add("# $AuditId runtime attestation fixture")
    return ($lines -join "`n") + "`n"
}

function Write-SignedAttestation(
    [string]$AttestationId,
    [string]$ExecutionContextId,
    [string]$RuntimeContextRef,
    [string]$RecordId,
    [string]$RecordPath,
    [string]$Scope,
    [string]$RecordType,
    [string]$BaselineRevision,
    [string]$TaskId,
    [string]$ParentTaskId,
    [switch]$TamperSignature
) {
    $payload = [ordered]@{
        schema = 'runtime-context/v1'
        attestation_id = $AttestationId
        repository = $originUrl
        execution_context_id = $ExecutionContextId
        runtime_context_ref = $RuntimeContextRef
        record_id = $RecordId
        record_path = $RecordPath
        scope = $Scope
        record_type = $RecordType
        baseline_revision = $BaselineRevision
        task_id = $TaskId
        parent_task_id = $ParentTaskId
        issued_at = $testIssuedAt
        expires_at = $testExpiresAt
    }
    $payloadPath = Join-Path $fixtureRoot ("runtime-payload-{0}.json" -f [Guid]::NewGuid().ToString('N'))
    $signaturePath = Join-Path $fixtureRoot ("runtime-signature-{0}.bin" -f [Guid]::NewGuid().ToString('N'))
    try {
        Set-Utf8File $payloadPath ($payload | ConvertTo-Json -Depth 4 -Compress)
        Invoke-OpenSsl @('dgst', '-sha256', '-sign', $privateKeyPath, '-out', $signaturePath, $payloadPath) | Out-Null
        $payloadBytes = [IO.File]::ReadAllBytes($payloadPath)
        $signatureBytes = [IO.File]::ReadAllBytes($signaturePath)
        if ($TamperSignature) {
            $signatureBytes[0] = $signatureBytes[0] -bxor 1
        }
        $envelope = [ordered]@{
            schema = 'runtime-context-attestation/v1'
            attestation_id = $AttestationId
            algorithm = 'rsa-sha256'
            key_id = "sha256:$trustedKeySha256"
            payload_base64 = [Convert]::ToBase64String($payloadBytes)
            signature_base64 = [Convert]::ToBase64String($signatureBytes)
        }
        $attestationFullPath = Join-Path $fixtureRoot ("docs/evidence/runtime-attestations/$AttestationId.json")
        Set-Utf8File $attestationFullPath ($envelope | ConvertTo-Json -Depth 4 -Compress)
    } finally {
        Remove-Item -LiteralPath $payloadPath, $signaturePath -Force -ErrorAction SilentlyContinue
    }
}

function Write-SourceFixture {
    Set-Utf8File $sourceRecordPath (Get-AuditRecord `
        'AUD-0002' `
        'plan-audit/v2' `
        'targeted' `
        $sourceExecutionContextId `
        $sourceRuntimeContextRef `
        $sourceAttestationPath `
        $sourceScope `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null)
    Write-SignedAttestation `
        $sourceAttestationId `
        $sourceExecutionContextId `
        $sourceRuntimeContextRef `
        'AUD-0002' `
        $sourceRecordRelativePath `
        $sourceScope `
        'plan-audit/v2' `
        $historyBase `
        $sourceTaskId `
        'task-parent-source'
}

function Write-AcceptanceFixture(
    [string]$RecordRuntimeContextRef,
    [string]$PayloadRuntimeContextRef,
    [string]$PayloadScope,
    [string]$PayloadBaselineRevision,
    [string]$PayloadExecutionContextId,
    [string]$PayloadTaskId,
    [string]$PayloadRecordId = 'AUD-0003',
    [string]$PayloadRecordPath = $acceptanceRecordRelativePath,
    [switch]$TamperSignature
) {
    Set-Utf8File $acceptanceRecordPath (Get-AuditRecord `
        'AUD-0003' `
        'plan-acceptance/v2' `
        'acceptance' `
        $acceptanceExecutionContextId `
        $RecordRuntimeContextRef `
        $acceptanceAttestationPath `
        $acceptanceScope `
        $historyBase `
        'separate-context' `
        $sourceExecutionContextId `
        $sourceRuntimeContextRef `
        $sourceAttestationPath)
    Write-SignedAttestation `
        $acceptanceAttestationId `
        $PayloadExecutionContextId `
        $PayloadRuntimeContextRef `
        $PayloadRecordId `
        $PayloadRecordPath `
        $PayloadScope `
        'plan-acceptance/v2' `
        $PayloadBaselineRevision `
        $PayloadTaskId `
        'task-parent-acceptance' `
        -TamperSignature:$TamperSignature
}

function Write-ValidPair {
    Write-SourceFixture
    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot -Force | Out-Null
    Invoke-Git @('init', '-q') | Out-Null
    Invoke-Git @('config', 'user.name', 'Runtime Attestation Validator Test') | Out-Null
    Invoke-Git @('config', 'user.email', 'runtime-validator@example.invalid') | Out-Null
    Invoke-Git @('config', 'core.autocrlf', 'false') | Out-Null
    Invoke-Git @('remote', 'add', 'origin', $originUrl) | Out-Null

    $historicalRecordPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0001-20260715-historical-unattested.md'
    Set-Utf8File $historicalRecordPath (Get-AuditRecord `
        'AUD-0001' `
        'plan-audit/v2' `
        'targeted' `
        '10000000-0000-4000-8000-000000000001' `
        'runtime://historical-task' `
        $null `
        'plan:PLN-HISTORICAL' `
        '0000000000000000000000000000000000000000' `
        'direct-context' `
        $null `
        $null `
        $null)
    Invoke-Git @('add', '--', 'docs/audits/records/AUD-0001-20260715-historical-unattested.md') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'historical unattested record') | Out-Null
    $historyBase = (Invoke-Git @('rev-parse', 'HEAD') | Select-Object -First 1).Trim()

    Invoke-OpenSsl @('genpkey', '-algorithm', 'RSA', '-pkeyopt', 'rsa_keygen_bits:2048', '-out', $privateKeyPath) | Out-Null
    Invoke-OpenSsl @('pkey', '-in', $privateKeyPath, '-pubout', '-out', $publicKeyPath) | Out-Null
    $trustedKeySha256 = (Get-FileHash -LiteralPath $publicKeyPath -Algorithm SHA256).Hash.ToLowerInvariant()

    Assert-Pass 'historical unattested record at HistoryBase'

    $newUnattestedPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0004-20260715-new-unattested.md'
    Set-Utf8File $newUnattestedPath (Get-AuditRecord `
        'AUD-0004' `
        'plan-audit/v2' `
        'targeted' `
        '10000000-0000-4000-8000-000000000002' `
        'runtime://new-unattested-task' `
        $null `
        'plan:PLN-NEW' `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null)
    Assert-Fail 'new unattested record' 'externally signed runtime context attestation'
    Remove-Item -LiteralPath $newUnattestedPath -Force

    $missingGovernanceContractPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0006-20260715-missing-governance-contract.md'
    $missingGovernanceContractRecord = Get-AuditRecord `
        'AUD-0006' `
        'plan-audit/v2' `
        'targeted' `
        '10000000-0000-4000-8000-000000000006' `
        'runtime://missing-governance-contract' `
        $null `
        'plan:PLN-CONTRACT-DOWNGRADE' `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null
    Set-Utf8File $missingGovernanceContractPath ($missingGovernanceContractRecord.Replace("governance_contract: audit-loop/v3`n", ''))
    Assert-Fail 'new record missing governance_contract' 'must use governance_contract audit-loop/v3'
    Remove-Item -LiteralPath $missingGovernanceContractPath -Force

    $missingWorkflowContractPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0007-20260715-missing-workflow-contract.md'
    $missingWorkflowContractRecord = Get-AuditRecord `
        'AUD-0007' `
        'plan-audit/v2' `
        'targeted' `
        '10000000-0000-4000-8000-000000000007' `
        'runtime://missing-workflow-contract' `
        $null `
        'plan:PLN-CONTRACT-DOWNGRADE' `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null
    Set-Utf8File $missingWorkflowContractPath ($missingWorkflowContractRecord.Replace("workflow_contract_revision: audit-runtime/v1`n", ''))
    Assert-Fail 'new audit-loop/v3 record missing workflow_contract_revision' 'must use workflow_contract_revision audit-runtime/v1'
    Remove-Item -LiteralPath $missingWorkflowContractPath -Force

    Write-ValidPair
    Assert-Pass 'valid signed source and independently signed acceptance'

    $acceptanceAttestationFullPath = Join-Path $fixtureRoot $acceptanceAttestationPath
    $oversizedEnvelope = Get-Content -LiteralPath $acceptanceAttestationFullPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $oversizedEnvelope.payload_base64 = [Convert]::ToBase64String((New-Object byte[] 262145))
    Set-Utf8File $acceptanceAttestationFullPath ($oversizedEnvelope | ConvertTo-Json -Depth 6 -Compress)
    Assert-Fail 'oversized runtime payload' 'payload/signature exceeds the permitted size'
    Write-ValidPair

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId `
        -TamperSignature
    Assert-Fail 'tampered signature' 'Runtime attestation signature is invalid'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        'plan:PLN-WRONG-SCOPE' `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId
    Assert-Fail 'wrong signed scope' 'signed payload does not bind the record context, scope, baseline'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        '0000000000000000000000000000000000000000' `
        $acceptanceExecutionContextId `
        $acceptanceTaskId
    Assert-Fail 'wrong signed baseline' 'signed payload does not bind the record context, scope, baseline'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $wrongExecutionContextId `
        $acceptanceTaskId
    Assert-Fail 'wrong signed execution context' 'signed payload does not bind the record context, scope, baseline'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId `
        -PayloadRecordId 'AUD-9999'
    Assert-Fail 'wrong signed record id' 'signed payload does not bind the record context, scope, baseline'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId `
        -PayloadRecordPath 'docs/audits/records/AUD-9999-forged.md'
    Assert-Fail 'wrong signed record path' 'signed payload does not bind the record context, scope, baseline'

    Write-AcceptanceFixture `
        $acceptanceRuntimeContextRef `
        $acceptanceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $sourceTaskId
    Assert-Fail 'same task as signed source' 'Independent task must differ from every signed source task'

    Write-AcceptanceFixture `
        $sourceRuntimeContextRef `
        $sourceRuntimeContextRef `
        $acceptanceScope `
        $historyBase `
        $acceptanceExecutionContextId `
        $acceptanceTaskId
    Assert-Fail 'same runtime reference as signed source' 'Independent task must differ from every signed source task'

    Write-ValidPair
    $forgedSourceRefRecord = (Get-Content -LiteralPath $acceptanceRecordPath -Raw -Encoding UTF8).Replace(
        "source_context_refs: $sourceRuntimeContextRef",
        'source_context_refs: runtime://forged-source-task'
    )
    Set-Utf8File $acceptanceRecordPath $forgedSourceRefRecord
    Assert-Fail 'forged source runtime reference' 'exact signed source runtime refs'

    Write-ValidPair
    Assert-Fail 'missing external trust anchor' 'require AUDIT_RUNTIME_TRUSTED_KEY_SHA256 as an external trust anchor' -OmitTrustAnchor

    $duplicateRecordPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0005-20260715-duplicate-attestation.md'
    Set-Utf8File $duplicateRecordPath (Get-AuditRecord `
        'AUD-0005' `
        'plan-audit/v2' `
        'targeted' `
        '30000000-0000-4000-8000-000000000005' `
        $sourceRuntimeContextRef `
        $sourceAttestationPath `
        $sourceScope `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null)
    Assert-Fail 'single-use attestation' 'Runtime attestation must be single-use'
    Remove-Item -LiteralPath $duplicateRecordPath -Force

    $duplicateContextAttestationId = '20000000-0000-4000-8000-000000000005'
    $duplicateContextAttestationPath = "docs/evidence/runtime-attestations/$duplicateContextAttestationId.json"
    Set-Utf8File $duplicateRecordPath (Get-AuditRecord `
        'AUD-0005' `
        'plan-audit/v2' `
        'targeted' `
        $sourceExecutionContextId `
        'runtime://distinct-task-reusing-context-id' `
        $duplicateContextAttestationPath `
        'plan:PLN-DUPLICATE-CONTEXT' `
        $historyBase `
        'direct-context' `
        $null `
        $null `
        $null)
    Write-SignedAttestation `
        $duplicateContextAttestationId `
        $sourceExecutionContextId `
        'runtime://distinct-task-reusing-context-id' `
        'AUD-0005' `
        'docs/audits/records/AUD-0005-20260715-duplicate-attestation.md' `
        'plan:PLN-DUPLICATE-CONTEXT' `
        'plan-audit/v2' `
        $historyBase `
        'task-distinct-reusing-context-id' `
        'task-parent-duplicate-context'
    Assert-Fail 'duplicate execution context id' 'unique execution_context_id'
    Remove-Item -LiteralPath $duplicateRecordPath -Force
    Remove-Item -LiteralPath (Join-Path $fixtureRoot $duplicateContextAttestationPath) -Force

    Assert-Pass 'restored valid signed pair after negative cases'
    Write-Output 'Runtime attestation validator tests passed: HistoryBase grandfathering, fail-closed signing, globally unique execution contexts, record binding, independent source identity, external trust anchoring, and single-use attestations are enforced.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot) {
        $resolvedFixtureRoot = (Resolve-Path $fixtureRoot).Path
        $resolvedFixtureBase = if (Test-Path -LiteralPath $fixtureBase) {
            (Resolve-Path $fixtureBase).Path
        } else {
            [IO.Path]::GetFullPath($fixtureBase)
        }
        $allowedPrefix = $resolvedFixtureBase.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar + '.validate-runtime-attestations-'
        if (-not $resolvedFixtureRoot.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected runtime attestation fixture path: $resolvedFixtureRoot"
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
