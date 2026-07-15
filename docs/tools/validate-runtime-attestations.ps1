param(
    [string]$RepositoryRoot,
    [string]$HistoryBase,
    [string]$PublicKeyPath,
    [string]$TrustedKeySha256
)

$ErrorActionPreference = 'Stop'

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

if ([string]::IsNullOrWhiteSpace($HistoryBase)) {
    $HistoryBase = $env:AUDIT_HISTORY_BASE
}
if ([string]::IsNullOrWhiteSpace($TrustedKeySha256)) {
    $TrustedKeySha256 = $env:AUDIT_RUNTIME_TRUSTED_KEY_SHA256
}

$failures = New-Object System.Collections.Generic.List[string]
$uuidV4Pattern = '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$'
$maxAttestationBytes = 1048576
$maxPayloadBytes = 262144
$maxSignatureBytes = 16384

function Get-RepoRelativePath([string]$Path) {
    return $Path.Substring($repoRoot.Length + 1).Replace('\', '/')
}

function Get-Frontmatter([string]$Content) {
    $match = [regex]::Match($Content, '(?s)\A(?:\uFEFF)?---\r?\n(?<frontmatter>.*?)\r?\n---\r?\n')
    if (-not $match.Success) { return $null }
    return $match.Groups['frontmatter'].Value
}

function Get-FrontmatterValue([string]$Frontmatter, [string]$Field) {
    $match = [regex]::Match($Frontmatter, "(?m)^${Field}:\s*(?<value>.+?)\s*$")
    if (-not $match.Success) { return $null }
    return $match.Groups['value'].Value.Trim()
}

function Get-ListValues([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -eq 'none' -or $Value -eq 'legacy-unavailable') {
        return @()
    }
    return @($Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Get-GitRevision([string]$Value) {
    if ($Value -match '^git:(?<sha>[0-9a-fA-F]{40})(?:;\s*worktree:clean)?$') {
        return $Matches['sha'].ToLowerInvariant()
    }
    return $null
}

function Test-GitCommit([string]$Revision) {
    if ([string]::IsNullOrWhiteSpace($Revision)) { return $false }
    $previousPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        & git -C $repoRoot cat-file -e "$Revision`^{commit}" 2>$null
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    return $exitCode -eq 0
}

function Test-GitPathAtRevision([string]$Revision, [string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Revision)) { return $false }
    $previousPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        & git -C $repoRoot cat-file -e "$Revision`:$Path" 2>$null
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousPreference
    }
    return $exitCode -eq 0
}

function Get-RecordType([string]$Frontmatter) {
    foreach ($field in @('audit_schema', 'remediation_schema', 'implementation_schema')) {
        $value = Get-FrontmatterValue $Frontmatter $field
        if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    }
    if ((Get-FrontmatterValue $Frontmatter 'audit_type') -eq 'follow-up') { return 'follow-up-audit/v1' }
    return $null
}

function ConvertTo-DateTimeOffsetOrNull([string]$Value) {
    $parsed = [DateTimeOffset]::MinValue
    if (-not [string]::IsNullOrWhiteSpace($Value) -and [DateTimeOffset]::TryParse($Value, [ref]$parsed)) {
        return $parsed
    }
    return $null
}

$recordRoots = @(
    'docs/audits/records',
    'docs/remediations/records',
    'docs/implementations/records'
)
$recordFiles = New-Object System.Collections.Generic.List[IO.FileInfo]
foreach ($recordRoot in $recordRoots) {
    $fullRoot = Join-Path $repoRoot $recordRoot
    if (Test-Path -LiteralPath $fullRoot -PathType Container) {
        Get-ChildItem -LiteralPath $fullRoot -Filter '*.md' -File | ForEach-Object { $recordFiles.Add($_) }
    }
}

$historyRevision = $null
if (-not [string]::IsNullOrWhiteSpace($HistoryBase)) {
    $candidateOutput = @(& git -C $repoRoot rev-parse --verify "$HistoryBase^{commit}" 2>$null)
    $candidateExitCode = $LASTEXITCODE
    $candidate = $candidateOutput | Select-Object -First 1
    if ($candidateExitCode -eq 0 -and $candidate -match '^[0-9a-fA-F]{40}$') {
        $historyRevision = $candidate.ToLowerInvariant()
        $previousPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = 'Continue'
            & git -C $repoRoot merge-base --is-ancestor $historyRevision HEAD 2>$null
            $historyAncestorExitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousPreference
        }
        if ($historyAncestorExitCode -ne 0) {
            $failures.Add("Runtime attestation HistoryBase must be an ancestor of HEAD: $HistoryBase")
        }
    } else {
        $failures.Add("Runtime attestation validation requires a valid HistoryBase: $HistoryBase")
    }
}

$records = New-Object System.Collections.Generic.List[object]
foreach ($file in $recordFiles) {
    $relativePath = Get-RepoRelativePath $file.FullName
    $content = Get-Content -LiteralPath $file.FullName -Raw -Encoding UTF8
    $frontmatter = Get-Frontmatter $content
    if ($null -eq $frontmatter) { continue }

    $governanceContract = Get-FrontmatterValue $frontmatter 'governance_contract'
    $contract = Get-FrontmatterValue $frontmatter 'workflow_contract_revision'
    $isNew = if ($null -ne $historyRevision) {
        -not (Test-GitPathAtRevision $historyRevision $relativePath)
    } else {
        -not (Test-GitPathAtRevision 'HEAD' $relativePath)
    }
    if ($isNew -and $governanceContract -ne 'audit-loop/v3') {
        $failures.Add("New governance records must use governance_contract audit-loop/v3: $relativePath")
    }
    if ($isNew -and $contract -ne 'audit-runtime/v1') {
        $failures.Add("New governance records must use workflow_contract_revision audit-runtime/v1: $relativePath")
    }
    if ($governanceContract -ne 'audit-loop/v3') { continue }

    $recordId = Get-FrontmatterValue $frontmatter 'audit_id'
    if ([string]::IsNullOrWhiteSpace($recordId)) { $recordId = Get-FrontmatterValue $frontmatter 'remediation_id' }
    if ([string]::IsNullOrWhiteSpace($recordId)) { $recordId = Get-FrontmatterValue $frontmatter 'implementation_id' }

    $attestationPath = Get-FrontmatterValue $frontmatter 'runtime_context_attestation'
    if ($isNew -and [string]::IsNullOrWhiteSpace($attestationPath)) {
        $failures.Add("New audit-loop/v3 records must reference an externally signed runtime context attestation: $relativePath")
    }

    $records.Add([pscustomobject]@{
        Id = $recordId
        Path = $relativePath
        Content = $content
        Frontmatter = $frontmatter
        Contract = $contract
        IsNew = $isNew
        ExecutionContextId = Get-FrontmatterValue $frontmatter 'execution_context_id'
        RuntimeContextRef = Get-FrontmatterValue $frontmatter 'runtime_context_ref'
        AttestationPath = $attestationPath
        SourceContextIds = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_context_ids'))
        SourceContextRefs = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_context_refs'))
        SourceAttestations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_context_attestations'))
        IndependenceBasis = Get-FrontmatterValue $frontmatter 'independence_basis'
        Scope = Get-FrontmatterValue $frontmatter 'scope'
        BaselineRevision = Get-GitRevision (Get-FrontmatterValue $frontmatter 'baseline')
        RecordType = Get-RecordType $frontmatter
        Payload = $null
    })
}

$attestedRecords = @($records | Where-Object { -not [string]::IsNullOrWhiteSpace($_.AttestationPath) })
$contextOwners = @{}
foreach ($record in @($records | Where-Object { -not [string]::IsNullOrWhiteSpace($_.ExecutionContextId) })) {
    $contextKey = $record.ExecutionContextId.ToLowerInvariant()
    if (-not $contextOwners.ContainsKey($contextKey)) { $contextOwners[$contextKey] = @() }
    $contextOwners[$contextKey] += $record
}
foreach ($record in @($records | Where-Object { $_.IsNew -and -not [string]::IsNullOrWhiteSpace($_.ExecutionContextId) })) {
    $contextKey = $record.ExecutionContextId.ToLowerInvariant()
    if (@($contextOwners[$contextKey]).Count -gt 1) {
        $failures.Add("Every new governance record must use a unique execution_context_id: $($record.Path) ($(@($contextOwners[$contextKey]).Path -join ', '))")
    }
}
if ($attestedRecords.Count -eq 0) {
    if ($failures.Count -gt 0) {
        $failures | ForEach-Object { Write-Error $_ }
        exit 1
    }
    Write-Output 'Runtime attestation validation passed: no new or attested governance records require signature verification.'
    exit 0
}

$temporaryKeyPath = $null
try {
    if ([string]::IsNullOrWhiteSpace($PublicKeyPath)) {
        $PublicKeyPath = $env:AUDIT_RUNTIME_PUBLIC_KEY_PATH
    }
    if ([string]::IsNullOrWhiteSpace($PublicKeyPath) -and -not [string]::IsNullOrWhiteSpace($env:AUDIT_RUNTIME_PUBLIC_KEY_BASE64)) {
        $temporaryKeyPath = Join-Path ([IO.Path]::GetTempPath()) ("audit-runtime-key-{0}.pem" -f [guid]::NewGuid().ToString('N'))
        try {
            [IO.File]::WriteAllBytes($temporaryKeyPath, [Convert]::FromBase64String($env:AUDIT_RUNTIME_PUBLIC_KEY_BASE64))
            $PublicKeyPath = $temporaryKeyPath
        } catch {
            $failures.Add('AUDIT_RUNTIME_PUBLIC_KEY_BASE64 must contain a valid base64-encoded PEM public key.')
        }
    }

    if ([string]::IsNullOrWhiteSpace($PublicKeyPath) -or -not (Test-Path -LiteralPath $PublicKeyPath -PathType Leaf)) {
        $failures.Add('Signed governance records require an externally supplied runtime attestation public key.')
    }
    if ($TrustedKeySha256 -notmatch '^[0-9a-fA-F]{64}$') {
        $failures.Add('Signed governance records require AUDIT_RUNTIME_TRUSTED_KEY_SHA256 as an external trust anchor.')
    }

    $openssl = @(Get-Command openssl -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
    if ($null -eq $openssl) {
        $failures.Add('OpenSSL is required to verify runtime attestation signatures.')
    } else {
        $opensslPath = [IO.Path]::GetFullPath([string]$openssl.Source)
        if ($opensslPath.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            $failures.Add('OpenSSL must not resolve to a repository-local executable.')
            $openssl = $null
        } else {
            $openssl = [pscustomobject]@{ Source = $opensslPath }
        }
    }

    $trustedKeyId = $null
    if ($failures.Count -eq 0) {
        $actualKeyHash = (Get-FileHash -LiteralPath $PublicKeyPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actualKeyHash -ne $TrustedKeySha256.ToLowerInvariant()) {
            $failures.Add("Runtime attestation public key does not match the external trust anchor: expected=$($TrustedKeySha256.ToLowerInvariant()) actual=$actualKeyHash")
        } else {
            $trustedKeyId = "sha256:$actualKeyHash"
        }
    }

    $originOutput = @(& git -C $repoRoot remote get-url origin 2>$null)
    $originExitCode = $LASTEXITCODE
    $originUrl = $originOutput | Select-Object -First 1
    if ($originExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($originUrl)) {
        $failures.Add('Runtime attestation validation requires a canonical origin URL.')
    }

    $usedAttestations = @{}
    $usedAttestationIds = @{}
    foreach ($record in $attestedRecords) {
        if ($record.ExecutionContextId -notmatch $uuidV4Pattern) {
            $failures.Add("Attested governance record must use a UUIDv4 execution_context_id: $($record.Path)")
        }
        if ([string]::IsNullOrWhiteSpace($record.RuntimeContextRef) -or $record.RuntimeContextRef -eq 'runtime-unavailable') {
            $failures.Add("Attested governance record must declare a real runtime_context_ref: $($record.Path)")
        }
        $attestationPathMatch = [regex]::Match(
            $record.AttestationPath,
            '^docs/evidence/runtime-attestations/(?<id>[0-9a-fA-F-]{36})\.json$'
        )
        if (-not $attestationPathMatch.Success -or $attestationPathMatch.Groups['id'].Value -notmatch $uuidV4Pattern) {
            $failures.Add("Attested governance record must reference a stable runtime attestation path: $($record.Path)")
            continue
        }
        $attestationId = $attestationPathMatch.Groups['id'].Value.ToLowerInvariant()
        if ($usedAttestations.ContainsKey($record.AttestationPath)) {
            $failures.Add("Runtime attestation must be single-use: $($record.AttestationPath) ($($usedAttestations[$record.AttestationPath]), $($record.Path))")
            continue
        }
        $usedAttestations[$record.AttestationPath] = $record.Path
        $attestationFullPath = Join-Path $repoRoot $record.AttestationPath
        if (-not (Test-Path -LiteralPath $attestationFullPath -PathType Leaf)) {
            $failures.Add("Runtime attestation file is missing: $($record.Path) ($($record.AttestationPath))")
            continue
        }

        try {
            $attestationItem = Get-Item -LiteralPath $attestationFullPath -Force
            if (($attestationItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or
                -not [string]::IsNullOrWhiteSpace($attestationItem.LinkType) -or
                $attestationItem.Length -gt $maxAttestationBytes) {
                throw 'runtime attestation file is not a bounded regular file'
            }
            $attestationBytes = [IO.File]::ReadAllBytes($attestationFullPath)
            $attestationText = [Text.UTF8Encoding]::new($false, $true).GetString($attestationBytes)
            $envelope = $attestationText | ConvertFrom-Json
        } catch {
            $failures.Add("Runtime attestation must be a bounded regular UTF-8 JSON file: $($record.AttestationPath)")
            continue
        }
        if ($envelope.schema -ne 'runtime-context-attestation/v1' -or
            $envelope.algorithm -ne 'rsa-sha256' -or
            $envelope.key_id -ne $trustedKeyId -or
            $envelope.attestation_id -ne $attestationId) {
            $failures.Add("Runtime attestation envelope does not match the trusted contract: $($record.AttestationPath)")
            continue
        }
        if ($usedAttestationIds.ContainsKey($envelope.attestation_id)) {
            $failures.Add("Runtime attestation_id must be single-use: $($record.AttestationPath) ($($usedAttestationIds[$envelope.attestation_id]), $($record.Path))")
            continue
        }
        $usedAttestationIds[$envelope.attestation_id] = $record.Path

        try {
            $payloadBytes = [Convert]::FromBase64String([string]$envelope.payload_base64)
            $signatureBytes = [Convert]::FromBase64String([string]$envelope.signature_base64)
        } catch {
            $failures.Add("Runtime attestation payload/signature must be base64: $($record.AttestationPath)")
            continue
        }
        if ($payloadBytes.Length -gt $maxPayloadBytes -or $signatureBytes.Length -gt $maxSignatureBytes) {
            $failures.Add("Runtime attestation payload/signature exceeds the permitted size: $($record.AttestationPath)")
            continue
        }

        $payloadPath = Join-Path ([IO.Path]::GetTempPath()) ("audit-runtime-payload-{0}.json" -f [guid]::NewGuid().ToString('N'))
        $signaturePath = Join-Path ([IO.Path]::GetTempPath()) ("audit-runtime-signature-{0}.bin" -f [guid]::NewGuid().ToString('N'))
        try {
            [IO.File]::WriteAllBytes($payloadPath, $payloadBytes)
            [IO.File]::WriteAllBytes($signaturePath, $signatureBytes)
            if ($null -ne $openssl -and (Test-Path -LiteralPath $PublicKeyPath -PathType Leaf)) {
                $previousPreference = $ErrorActionPreference
                try {
                    $ErrorActionPreference = 'Continue'
                    & $openssl.Source dgst -sha256 -verify $PublicKeyPath -signature $signaturePath $payloadPath *> $null
                    $opensslExitCode = $LASTEXITCODE
                } finally {
                    $ErrorActionPreference = $previousPreference
                }
                if ($opensslExitCode -ne 0) {
                    $failures.Add("Runtime attestation signature is invalid: $($record.AttestationPath)")
                    continue
                }
            } else {
                continue
            }
            try {
                $payloadText = [Text.Encoding]::UTF8.GetString($payloadBytes)
                $payload = $payloadText | ConvertFrom-Json
            } catch {
                $failures.Add("Runtime attestation signed payload must be valid UTF-8 JSON: $($record.AttestationPath)")
                continue
            }
        } finally {
            Remove-Item -LiteralPath $payloadPath, $signaturePath -Force -ErrorAction SilentlyContinue
        }

        $issuedAt = ConvertTo-DateTimeOffsetOrNull ([string]$payload.issued_at)
        $expiresAt = ConvertTo-DateTimeOffsetOrNull ([string]$payload.expires_at)
        if ($payload.schema -ne 'runtime-context/v1' -or
            $payload.attestation_id -ne $attestationId -or
            $payload.repository -ne $originUrl.Trim() -or
            $payload.execution_context_id -ne $record.ExecutionContextId -or
            $payload.runtime_context_ref -ne $record.RuntimeContextRef -or
            $payload.record_id -ne $record.Id -or
            $payload.record_path -ne $record.Path -or
            $payload.scope -ne $record.Scope -or
            $payload.record_type -ne $record.RecordType -or
            $payload.baseline_revision -ne $record.BaselineRevision -or
            [string]::IsNullOrWhiteSpace([string]$payload.task_id) -or
            [string]::IsNullOrWhiteSpace([string]$payload.parent_task_id) -or
            $payload.task_id -eq $payload.parent_task_id -or
            $null -eq $issuedAt -or $null -eq $expiresAt -or
            $issuedAt -ge $expiresAt -or ($expiresAt - $issuedAt).TotalHours -gt 24) {
            $failures.Add("Runtime attestation signed payload does not bind the record context, scope, baseline, and bounded lifetime: $($record.AttestationPath)")
            continue
        }
        $record.Payload = $payload
    }

    $attestedByContextId = @{}
    foreach ($record in $attestedRecords) {
        if (-not [string]::IsNullOrWhiteSpace($record.ExecutionContextId)) {
            if (-not $attestedByContextId.ContainsKey($record.ExecutionContextId)) { $attestedByContextId[$record.ExecutionContextId] = @() }
            $attestedByContextId[$record.ExecutionContextId] += $record
        }
    }

    foreach ($record in @($attestedRecords | Where-Object { $_.IndependenceBasis -eq 'separate-context' })) {
        if ($null -eq $record.Payload) { continue }
        if ($record.SourceContextIds.Count -eq 0 -or $record.SourceContextRefs.Count -eq 0) {
            $failures.Add("Independent attested record must identify signed source contexts: $($record.Path)")
            continue
        }
        $expectedSourceAttestations = New-Object System.Collections.Generic.List[string]
        $expectedSourceRefs = New-Object System.Collections.Generic.List[string]
        foreach ($sourceContextId in $record.SourceContextIds) {
            if (-not $attestedByContextId.ContainsKey($sourceContextId) -or $attestedByContextId[$sourceContextId].Count -ne 1) {
                $failures.Add("Independent attested record must resolve each source context to one signed record: $($record.Path) ($sourceContextId)")
                continue
            }
            $sourceRecord = $attestedByContextId[$sourceContextId][0]
            if ($null -eq $sourceRecord.Payload) {
                $failures.Add("Independent source attestation is not valid: $($record.Path) ($sourceContextId)")
                continue
            }
            $expectedSourceAttestations.Add($sourceRecord.AttestationPath)
            $expectedSourceRefs.Add($sourceRecord.RuntimeContextRef)
            if ($sourceRecord.Payload.task_id -eq $record.Payload.task_id -or
                $sourceRecord.RuntimeContextRef -eq $record.RuntimeContextRef) {
                $failures.Add("Independent task must differ from every signed source task: $($record.Path) ($sourceContextId)")
            }
        }
        $actualSet = @($record.SourceAttestations | Sort-Object -Unique)
        $expectedSet = @($expectedSourceAttestations | Sort-Object -Unique)
        if (($actualSet -join ',') -ne ($expectedSet -join ',')) {
            $failures.Add("Independent record must list the exact signed source attestations: $($record.Path)")
        }
        $actualRefSet = @($record.SourceContextRefs | Sort-Object -Unique)
        $expectedRefSet = @($expectedSourceRefs | Sort-Object -Unique)
        if (($actualRefSet -join ',') -ne ($expectedRefSet -join ',')) {
            $failures.Add("Independent record must list the exact signed source runtime refs: $($record.Path)")
        }
    }
} finally {
    if ($null -ne $temporaryKeyPath) {
        Remove-Item -LiteralPath $temporaryKeyPath -Force -ErrorAction SilentlyContinue
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { Write-Error $_ }
    exit 1
}

Write-Output "Runtime attestation validation passed: $($attestedRecords.Count) governance record(s) are signed, scoped, revision-bound, and independently sourced."
