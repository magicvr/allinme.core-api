$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$validator = Join-Path $PSScriptRoot 'validate-evidence-attestations.ps1'
$fixtureBase = Join-Path $repoRoot '.tmp'
$testRoot = Join-Path $fixtureBase ('.validate-evidence-attestations-' + [Guid]::NewGuid().ToString('N'))
$fixtureRoot = Join-Path $testRoot 'repository'
$keyRoot = Join-Path $testRoot 'external-keys'
$privateKeyPath = Join-Path $keyRoot 'evidence-private.pem'
$publicKeyPath = Join-Path $keyRoot 'evidence-public.pem'
$git = @(Get-Command git -CommandType Application -ErrorAction Stop) | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath([string]$git.Source)
$repoRootPrefix = $repoRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if ([string]::Equals($gitExecutable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
    $gitExecutable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-controlled Git binary: $gitExecutable"
}
$openssl = Get-Command openssl -ErrorAction Stop
$shell = Get-Command pwsh -ErrorAction SilentlyContinue
if ($null -eq $shell) {
    $shell = Get-Command powershell.exe -ErrorAction SilentlyContinue
}
if ($null -eq $shell) {
    $shell = Get-Command powershell -ErrorAction Stop
}

$originUrl = 'https://example.invalid/allinme/evidence-attestation-fixture.git'
$approvedImageDigest = 'docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a'
$approvedImageId = 'sha256:dd2d88d0c7034f9e48bb74156ea562e66d3064971aed54ccbb23554637580f1c'
$runId = '10000000-0000-4000-8000-000000000001'
$attestationId = '20000000-0000-4000-8000-000000000001'
$auditId = 'AUD-0101'
$recordRelativePath = 'docs/audits/records/AUD-0101-20260715-evidence-attestation.md'
$recordPath = Join-Path $fixtureRoot $recordRelativePath.Replace('/', '\')
$artifactRelativePath = "docs/evidence/runs/$runId/evidence.json"
$artifactPath = Join-Path $fixtureRoot $artifactRelativePath.Replace('/', '\')
$attestationRelativePath = "docs/evidence/runs/$runId/attestation.json"
$attestationPath = Join-Path $fixtureRoot $attestationRelativePath.Replace('/', '\')
$historyBase = $null
$trustedKeySha256 = $null

function ConvertTo-NativeArgumentString([string[]]$Arguments) {
    return (($Arguments | ForEach-Object {
        if ($_.Length -eq 0) {
            '""'
        } elseif ($_ -notmatch '[\s"]') {
            $_
        } else {
            $quoted = New-Object Text.StringBuilder
            [void]$quoted.Append([char]34)
            $backslashCount = 0
            foreach ($character in $_.ToCharArray()) {
                if ($character -eq [char]92) {
                    $backslashCount++
                    continue
                }
                if ($character -eq [char]34) {
                    [void]$quoted.Append([char]92, (2 * $backslashCount) + 1)
                    [void]$quoted.Append([char]34)
                    $backslashCount = 0
                    continue
                }
                if ($backslashCount -gt 0) {
                    [void]$quoted.Append([char]92, $backslashCount)
                    $backslashCount = 0
                }
                [void]$quoted.Append($character)
            }
            if ($backslashCount -gt 0) {
                [void]$quoted.Append([char]92, 2 * $backslashCount)
            }
            [void]$quoted.Append([char]34)
            $quoted.ToString()
        }
    }) -join ' ')
}

function Invoke-NativeCapture([string]$FilePath, [string[]]$Arguments) {
    $startInfo = New-Object Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.Arguments = ConvertTo-NativeArgumentString $Arguments
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $startInfo
    if (-not $process.Start()) { throw "Unable to start $FilePath" }
    try {
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        $process.WaitForExit()
        return [pscustomobject]@{
            ExitCode = $process.ExitCode
            Stdout = $stdoutTask.GetAwaiter().GetResult()
            Stderr = $stderrTask.GetAwaiter().GetResult()
        }
    } finally {
        $process.Dispose()
    }
}

function Invoke-NativeOrThrow([string]$FilePath, [string[]]$Arguments) {
    $result = Invoke-NativeCapture $FilePath $Arguments
    if ($result.ExitCode -ne 0) {
        throw "$FilePath $($Arguments -join ' ') failed: $($result.Stdout) $($result.Stderr)"
    }
    return $result.Stdout
}

function Invoke-Git([string[]]$Arguments) {
    return Invoke-NativeOrThrow $gitExecutable (@('-C', $fixtureRoot) + $Arguments)
}

function Invoke-OpenSsl([string[]]$Arguments) {
    return Invoke-NativeOrThrow $openssl.Source $Arguments
}

function Set-Utf8File([string]$Path, [string]$Content) {
    $parent = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $parent -PathType Container)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    $normalized = $Content.Replace("`r`n", "`n").Replace("`r", "`n").Replace("`n", [Environment]::NewLine)
    [IO.File]::WriteAllText($Path, $normalized, [Text.UTF8Encoding]::new($false))
}

function Get-Sha256Hex([byte[]]$Bytes) {
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha256.ComputeHash($Bytes))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha256.Dispose()
    }
}

function Get-AuditRecord(
    [string]$Id,
    [string]$Status,
    [string]$EvidenceRunId,
    [string]$EvidenceRevision,
    [string]$EvidenceArtifact,
    [string]$EvidenceAttestation,
    [string]$EvidenceArgvJson = '["git","rev-parse","HEAD"]'
) {
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add('---')
    $lines.Add("status: $Status")
    $lines.Add('governance_contract: audit-loop/v3')
    $lines.Add('workflow_contract_revision: audit-runtime/v1')
    $lines.Add('audit_schema: plan-audit/v2')
    $lines.Add("audit_id: $Id")
    $lines.Add('auditor: evidence-attestation-validator-test')
    $lines.Add('audit_type: targeted')
    $lines.Add('scope: plan:PLN-0001')
    $lines.Add('subject: evidence attestation fixture')
    $lines.Add("baseline: git:$EvidenceRevision; worktree:clean")
    $lines.Add("evidence_revision: git:$EvidenceRevision; worktree:clean")
    $lines.Add("evidence_worktree_revision: git:$EvidenceRevision")
    $lines.Add('evidence_runner: docs/tools/invoke-revision-evidence.ps1')
    if (-not [string]::IsNullOrWhiteSpace($EvidenceArgvJson)) {
        $lines.Add("evidence_argv_json: $EvidenceArgvJson")
    }
    $lines.Add("evidence_run_id: $EvidenceRunId")
    $lines.Add("evidence_artifact: $EvidenceArtifact")
    if (-not [string]::IsNullOrWhiteSpace($EvidenceAttestation)) {
        $lines.Add("evidence_attestation: $EvidenceAttestation")
    }
    $lines.Add('started_at: 2026-07-15T00:00:00Z')
    $lines.Add($(if ($Status -eq 'closed') { 'completed_at: 2026-07-15T00:05:00Z' } else { 'completed_at: pending' }))
    $lines.Add('last_updated: 2026-07-15')
    $lines.Add('related_audits: none')
    $lines.Add('related_plans: PLN-0001')
    $lines.Add('---')
    $lines.Add('')
    $lines.Add("# $Id evidence attestation fixture")
    return ($lines -join "`n") + "`n"
}

function Write-EvidenceArtifact(
    [string]$ImageDigest = $approvedImageDigest,
    [string[]]$Argv = @('git', 'rev-parse', 'HEAD')
) {
    $tree = (Invoke-Git @('rev-parse', "$historyBase`^{tree}")).Trim()
    $artifact = [ordered]@{
        schema = 'revision-evidence/v1'
        evidence_run_id = $runId
        evidence_revision = $historyBase
        evidence_tree = $tree
        evidence_worktree = 'detached'
        argv = $Argv
        exit_code = 0
        isolation = [ordered]@{
            engine = 'docker'
            image = $ImageDigest
            image_id = $approvedImageId
            network = 'none'
            repository_mount = 'read-only'
            root_filesystem = 'read-only'
            capabilities = 'none'
            no_new_privileges = $true
            user = '65534:65534'
            memory_megabytes = 1024
            cpus = 1
            pids_limit = 256
            sanitized_environment = $true
            preflight_passed = $true
            workspace = '/tmp/workspace (tmpfs)'
            workspace_mount_options = 'rw,exec,nosuid,nodev,size=512m'
        }
        output = [ordered]@{
            stdout_sha256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
            stderr_sha256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
            combined_sha256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
            stdout_bytes = 0
            stderr_bytes = 0
        }
        clean_status = [ordered]@{
            host_tracked_clean_before_run = $true
            host_tracked_clean_after_run = $true
            host_tracked_state_unchanged = $true
            subject_tracked_clean_before_run = $true
            subject_tracked_clean_after_run = $true
            subject_workspace_discarded = $true
        }
        tracked_worktree_clean_after_run = $true
        started_at = '2026-07-15T00:00:00Z'
        completed_at = '2026-07-15T00:00:01Z'
    }
    Set-Utf8File $artifactPath (($artifact | ConvertTo-Json -Depth 8) + "`n")
}

function Write-SignedAttestation(
    [hashtable]$PayloadOverrides,
    [switch]$TamperPayloadAfterSigning,
    [switch]$OmitSignature
) {
    $artifactBytes = [IO.File]::ReadAllBytes($artifactPath)
    $artifact = [Text.UTF8Encoding]::new($false, $true).GetString($artifactBytes) | ConvertFrom-Json
    $payload = [ordered]@{
        schema = 'revision-evidence-binding/v1'
        attestation_id = $attestationId
        repository = $originUrl
        audit_id = $auditId
        audit_record_path = $recordRelativePath
        evidence_run_id = $runId
        evidence_artifact_path = $artifactRelativePath
        evidence_artifact_sha256 = Get-Sha256Hex $artifactBytes
        evidence_revision = $artifact.evidence_revision
        evidence_tree = $artifact.evidence_tree
        argv = @($artifact.argv)
        exit_code = [int]$artifact.exit_code
        image_digest = $artifact.isolation.image
        image_id = $artifact.isolation.image_id
        issued_at = '2026-07-15T00:01:00Z'
        expires_at = '2026-07-15T01:01:00Z'
    }
    if ($null -ne $PayloadOverrides) {
        foreach ($key in $PayloadOverrides.Keys) {
            $payload[$key] = $PayloadOverrides[$key]
        }
    }

    $payloadPath = Join-Path $testRoot ('payload-' + [Guid]::NewGuid().ToString('N') + '.json')
    $signaturePath = Join-Path $testRoot ('signature-' + [Guid]::NewGuid().ToString('N') + '.bin')
    try {
        $payloadText = $payload | ConvertTo-Json -Depth 8 -Compress
        [IO.File]::WriteAllText($payloadPath, $payloadText, [Text.UTF8Encoding]::new($false))
        Invoke-OpenSsl @('dgst', '-sha256', '-sign', $privateKeyPath, '-out', $signaturePath, $payloadPath) | Out-Null
        $payloadBytes = [IO.File]::ReadAllBytes($payloadPath)
        if ($TamperPayloadAfterSigning) {
            $tampered = ([Text.Encoding]::UTF8.GetString($payloadBytes)).Replace($auditId, 'AUD-9999')
            $payloadBytes = [Text.Encoding]::UTF8.GetBytes($tampered)
        }
        $envelope = [ordered]@{
            schema = 'revision-evidence-attestation/v1'
            attestation_id = $attestationId
            algorithm = 'rsa-sha256'
            key_id = "sha256:$trustedKeySha256"
            payload_base64 = [Convert]::ToBase64String($payloadBytes)
        }
        if (-not $OmitSignature) {
            $envelope.signature_base64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes($signaturePath))
        }
        Set-Utf8File $attestationPath (($envelope | ConvertTo-Json -Depth 8 -Compress) + "`n")
    } finally {
        Remove-Item -LiteralPath $payloadPath, $signaturePath -Force -ErrorAction SilentlyContinue
    }
}

function Write-ValidFixture {
    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath)
    Write-EvidenceArtifact
    Write-SignedAttestation
}

function Invoke-FixtureValidator(
    [switch]$OmitHistoryBase,
    [switch]$OmitPublicKey,
    [switch]$OmitTrustAnchor
) {
    $environmentNames = @(
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
            '-RepositoryRoot', $fixtureRoot
        )
        if (-not $OmitHistoryBase) { $arguments += @('-HistoryBase', $historyBase) }
        if (-not $OmitPublicKey) { $arguments += @('-PublicKeyPath', $publicKeyPath) }
        if (-not $OmitTrustAnchor) { $arguments += @('-TrustedKeySha256', $trustedKeySha256) }
        return Invoke-NativeCapture $shell.Source $arguments
    } finally {
        foreach ($name in $environmentNames) {
            [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process')
        }
    }
}

function Assert-Pass(
    [string]$Label,
    [switch]$OmitHistoryBase,
    [switch]$OmitPublicKey,
    [switch]$OmitTrustAnchor
) {
    $result = Invoke-FixtureValidator @PSBoundParameters
    if ($result.ExitCode -ne 0) {
        throw "$Label unexpectedly failed: $($result.Stdout) $($result.Stderr)"
    }
}

function Assert-Fail(
    [string]$Label,
    [string]$Pattern,
    [switch]$OmitHistoryBase,
    [switch]$OmitPublicKey,
    [switch]$OmitTrustAnchor
) {
    $validatorParameters = @{}
    foreach ($name in @('OmitHistoryBase', 'OmitPublicKey', 'OmitTrustAnchor')) {
        if ($PSBoundParameters.ContainsKey($name)) { $validatorParameters[$name] = $PSBoundParameters[$name] }
    }
    $result = Invoke-FixtureValidator @validatorParameters
    $output = "$($result.Stdout)`n$($result.Stderr)"
    if ($result.ExitCode -eq 0 -or $output -notmatch $Pattern) {
        throw "$Label was not rejected with /$Pattern/: $output"
    }
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot, $keyRoot -Force | Out-Null
    Invoke-Git @('init', '-q') | Out-Null
    Invoke-Git @('config', 'user.name', 'Evidence Attestation Validator Test') | Out-Null
    Invoke-Git @('config', 'user.email', 'evidence-validator@example.invalid') | Out-Null
    Invoke-Git @('config', 'core.autocrlf', 'false') | Out-Null
    Invoke-Git @('remote', 'add', 'origin', $originUrl) | Out-Null

    $historicalPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0100-20260715-historical.md'
    Set-Utf8File $historicalPath (Get-AuditRecord `
        'AUD-0100' `
        'closed' `
        '00000000-0000-4000-8000-000000000001' `
        '0000000000000000000000000000000000000000' `
        'docs/evidence/runs/00000000-0000-4000-8000-000000000001/evidence.json' `
        $null `
        $null)
    Invoke-Git @('add', '--', 'docs/audits/records/AUD-0100-20260715-historical.md') | Out-Null
    Invoke-Git @('commit', '-q', '-m', 'historical unattested audit') | Out-Null
    $historyBase = (Invoke-Git @('rev-parse', 'HEAD')).Trim()

    Invoke-OpenSsl @('genpkey', '-algorithm', 'RSA', '-pkeyopt', 'rsa_keygen_bits:2048', '-out', $privateKeyPath) | Out-Null
    Invoke-OpenSsl @('pkey', '-in', $privateKeyPath, '-pubout', '-out', $publicKeyPath) | Out-Null
    $trustedKeySha256 = (Get-FileHash -LiteralPath $publicKeyPath -Algorithm SHA256).Hash.ToLowerInvariant()

    Assert-Pass 'historical unattested audit at HistoryBase' -OmitPublicKey -OmitTrustAnchor

    $repositoryGitPath = Join-Path $fixtureRoot 'git.cmd'
    Set-Utf8File $repositoryGitPath "@echo off`r`nexit /b 99`r`n"
    $savedPath = $env:PATH
    try {
        $env:PATH = "$fixtureRoot$([IO.Path]::PathSeparator)$savedPath"
        Assert-Fail 'repository-controlled Git executable' 'Refusing to execute a repository-controlled Git binary' -OmitPublicKey -OmitTrustAnchor
    } finally {
        $env:PATH = $savedPath
        Remove-Item -LiteralPath $repositoryGitPath -Force
    }

    $openPath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0102-20260715-open.md'
    Set-Utf8File $openPath (Get-AuditRecord `
        'AUD-0102' `
        'open' `
        '30000000-0000-4000-8000-000000000001' `
        $historyBase `
        'docs/evidence/runs/30000000-0000-4000-8000-000000000001/evidence.json' `
        $null)
    Assert-Pass 'new open audit does not require terminal evidence signing' -OmitPublicKey -OmitTrustAnchor
    Remove-Item -LiteralPath $openPath -Force

    Write-ValidFixture
    $repositoryOpenSslPath = Join-Path $fixtureRoot 'openssl.cmd'
    Set-Utf8File $repositoryOpenSslPath "@echo off`r`nexit /b 99`r`n"
    $savedPath = $env:PATH
    try {
        $env:PATH = "$fixtureRoot$([IO.Path]::PathSeparator)$savedPath"
        Assert-Fail 'repository-controlled OpenSSL executable' 'Refusing to execute a repository-controlled OpenSSL binary'
    } finally {
        $env:PATH = $savedPath
        Remove-Item -LiteralPath $repositoryOpenSslPath -Force
    }
    Assert-Pass 'valid externally signed evidence attestation'

    Write-EvidenceArtifact -Argv @('/bin/true')
    Write-SignedAttestation
    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath '["/bin/true"]')
    Assert-Fail 'meaningless subject command' 'must invoke a meaningful subject-specific command'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath $null)
    Assert-Fail 'missing evidence argv declaration' 'must declare evidence_argv_json as strict JSON'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath '["git",]')
    Assert-Fail 'invalid evidence argv JSON' 'must declare evidence_argv_json as strict JSON'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath '[]')
    Assert-Fail 'empty evidence argv JSON array' 'must declare evidence_argv_json as strict JSON'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath '["git",1]')
    Assert-Fail 'non-string evidence argv JSON item' 'must declare evidence_argv_json as strict JSON'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath '["git","HEAD","rev-parse"]')
    Assert-Fail 'mismatched evidence argv declaration' 'must exactly match the ordered evidence artifact argv'

    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $null)
    Assert-Fail 'missing evidence attestation' 'must reference an externally signed evidence_attestation'

    Write-ValidFixture
    [IO.File]::AppendAllText($artifactPath, ' ', [Text.UTF8Encoding]::new($false))
    Assert-Fail 'artifact raw-byte tampering' 'does not exactly bind the repository, audit, artifact bytes'

    Write-ValidFixture
    Write-SignedAttestation -TamperPayloadAfterSigning
    Assert-Fail 'signed payload tampering' 'Evidence attestation signature is invalid'

    Write-ValidFixture
    Assert-Fail 'missing external trust anchor' 'require AUDIT_RUNTIME_TRUSTED_KEY_SHA256 as an external trust anchor' -OmitTrustAnchor

    Write-ValidFixture
    Write-SignedAttestation -OmitSignature
    Assert-Fail 'missing signature field' 'envelope must contain exactly the required fields'

    $wrongImage = 'docker.io/library/golang@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
    Set-Utf8File $recordPath (Get-AuditRecord $auditId 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath)
    Write-EvidenceArtifact -ImageDigest $wrongImage
    Write-SignedAttestation
    Assert-Fail 'correctly signed but unapproved image' 'does not use the approved fixed image digest'

    Write-ValidFixture
    Write-SignedAttestation -PayloadOverrides @{ audit_id = 'AUD-9999' }
    Assert-Fail 'correctly signed payload with wrong audit binding' 'does not exactly bind the repository, audit, artifact bytes'

    Write-ValidFixture
    Write-SignedAttestation -PayloadOverrides @{ expires_at = '2026-07-16T01:01:01Z' }
    Assert-Fail 'attestation lifetime over 24 hours' 'bounded lifetime'

    Write-ValidFixture
    $duplicatePath = Join-Path $fixtureRoot 'docs\audits\records\AUD-0103-20260715-duplicate-attestation.md'
    Set-Utf8File $duplicatePath (Get-AuditRecord 'AUD-0103' 'closed' $runId $historyBase $artifactRelativePath $attestationRelativePath)
    Assert-Fail 'single-use evidence attestation' 'must be single-use and referenced by exactly one audit record'
    Remove-Item -LiteralPath $duplicatePath -Force

    Write-ValidFixture
    Assert-Fail 'missing explicit HistoryBase' 'requires an explicit -HistoryBase' -OmitHistoryBase
    Assert-Pass 'restored valid fixture after negative cases'

    Write-Output 'Evidence attestation validator tests passed: explicit HistoryBase grandfathering, closed-only enforcement, exact declared/artifact/signed argv binding, repository-local Git/OpenSSL rejection, RSA trust anchoring, byte and payload tamper detection, fixed-image policy, bounded lifetime, and single-use binding are enforced.'
} finally {
    if (Test-Path -LiteralPath $testRoot) {
        $resolvedTestRoot = (Resolve-Path $testRoot).Path
        $resolvedFixtureBase = if (Test-Path -LiteralPath $fixtureBase) {
            (Resolve-Path $fixtureBase).Path
        } else {
            [IO.Path]::GetFullPath($fixtureBase)
        }
        $allowedPrefix = $resolvedFixtureBase.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar + '.validate-evidence-attestations-'
        if (-not $resolvedTestRoot.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove unexpected evidence attestation fixture path: $resolvedTestRoot"
        }
        for ($attempt = 1; $attempt -le 5; $attempt++) {
            try {
                Remove-Item -LiteralPath $resolvedTestRoot -Recurse -Force -ErrorAction Stop
                break
            } catch {
                if ($attempt -eq 5) { throw }
                Start-Sleep -Milliseconds 200
            }
        }
    }
}
