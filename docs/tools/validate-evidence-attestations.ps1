param(
    [string]$RepositoryRoot,
    [string]$HistoryBase,
    [string]$PublicKeyPath,
    [string]$TrustedKeySha256
)

$ErrorActionPreference = 'Stop'

$approvedImageDigest = 'docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a'
$uuidV4Pattern = '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
$sha256Pattern = '^[0-9a-f]{64}$'
$gitShaPattern = '^[0-9a-f]{40}$'
$maxArtifactBytes = 1MB
$maxAttestationBytes = 1MB
$maxPayloadBytes = 256KB
$maxSignatureBytes = 16KB
$failures = New-Object System.Collections.Generic.List[string]

function Invoke-NativeCapture {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $startInfo = New-Object Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.CreateNoWindow = $true
    $startInfo.StandardOutputEncoding = [Text.Encoding]::UTF8
    $startInfo.StandardErrorEncoding = [Text.Encoding]::UTF8
    $startInfo.Arguments = (($Arguments | ForEach-Object {
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

    $process = New-Object Diagnostics.Process
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw "Unable to start native command: $FilePath"
    }
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

function Get-Frontmatter([string]$Content) {
    $match = [regex]::Match($Content, '(?s)\A(?:\uFEFF)?---\r?\n(?<frontmatter>.*?)\r?\n---\r?\n')
    if (-not $match.Success) { return $null }
    return $match.Groups['frontmatter'].Value
}

function Get-FrontmatterValue([string]$Frontmatter, [string]$Field) {
    $match = [regex]::Match($Frontmatter, "(?m)^$([regex]::Escape($Field)):\s*(?<value>.*?)\s*$")
    if (-not $match.Success) { return $null }
    return $match.Groups['value'].Value.Trim()
}

function Get-GitRevision([string]$Value) {
    $match = [regex]::Match($Value, '^git:(?<sha>[0-9a-f]{40});\s*worktree:clean$')
    if (-not $match.Success) { return $null }
    return $match.Groups['sha'].Value
}

function Get-RepoRelativePath([string]$Path) {
    return $Path.Substring($repoRoot.Length + 1).Replace('\', '/')
}

function Get-Sha256Hex([byte[]]$Bytes) {
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha256.ComputeHash($Bytes))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha256.Dispose()
    }
}

function Test-ExactProperties([object]$Object, [string[]]$ExpectedNames) {
    if ($Object -isnot [pscustomobject]) { return $false }
    $actualNames = @($Object.PSObject.Properties | ForEach-Object { $_.Name } | Sort-Object)
    $expected = @($ExpectedNames | Sort-Object)
    if ($actualNames.Count -ne $expected.Count) { return $false }
    for ($index = 0; $index -lt $expected.Count; $index++) {
        if ($actualNames[$index] -cne $expected[$index]) { return $false }
    }
    return $true
}

function Test-StringArray([object]$Value) {
    if ($Value -isnot [System.Array]) { return $false }
    $items = @($Value)
    if ($items.Count -eq 0) { return $false }
    return @($items | Where-Object { $_ -isnot [string] -or [string]::IsNullOrWhiteSpace($_) }).Count -eq 0
}

function Test-MeaningfulEvidenceArgv([object[]]$Argv) {
    if ($null -eq $Argv -or $Argv.Count -eq 0) { return $false }
    $command = [IO.Path]::GetFileName(([string]$Argv[0]).Trim()).ToLowerInvariant()
    if ($command -in @('true', 'true.exe', 'false', 'false.exe', 'echo', 'echo.exe', 'printf', 'printf.exe', ':')) {
        return $false
    }
    if ($command -in @('sh', 'sh.exe', 'bash', 'bash.exe', 'cmd', 'cmd.exe', 'pwsh', 'pwsh.exe', 'powershell', 'powershell.exe') -and
        (($Argv -join ' ') -match '(?i)(?:^|\s)(?:-c|/c)\s*["'']?(?:true|false|echo|printf|:)(?:["'']?\s|$)')) {
        return $false
    }
    return $true
}

function Test-StringArraysEqual([object]$Left, [object]$Right) {
    if (-not (Test-StringArray $Left) -or -not (Test-StringArray $Right)) { return $false }
    $leftItems = @($Left)
    $rightItems = @($Right)
    if ($leftItems.Count -ne $rightItems.Count) { return $false }
    for ($index = 0; $index -lt $leftItems.Count; $index++) {
        if ([string]$leftItems[$index] -cne [string]$rightItems[$index]) { return $false }
    }
    return $true
}

function ConvertFrom-StrictEvidenceArgvJson([string]$Value) {
    $invalid = [pscustomobject]@{ IsValid = $false; Items = @() }
    if ([string]::IsNullOrWhiteSpace($Value)) { return $invalid }
    try {
        $parsed = ConvertFrom-Json -InputObject $Value -ErrorAction Stop
    } catch {
        return $invalid
    }
    if ($parsed -isnot [System.Array]) { return $invalid }
    $items = @($parsed)
    if ($items.Count -eq 0 -or
        @($items | Where-Object {
            $_ -isnot [string] -or [string]::IsNullOrWhiteSpace([string]$_) -or ([string]$_).IndexOf([char]0) -ge 0
        }).Count -gt 0) {
        return $invalid
    }
    return [pscustomobject]@{ IsValid = $true; Items = $items }
}

function ConvertTo-DateTimeOffsetOrNull([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or
        $Value -notmatch '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$') {
        return $null
    }
    $parsed = [DateTimeOffset]::MinValue
    if (-not [DateTimeOffset]::TryParse(
        $Value,
        [Globalization.CultureInfo]::InvariantCulture,
        [Globalization.DateTimeStyles]::RoundtripKind,
        [ref]$parsed
    )) {
        return $null
    }
    return $parsed
}

function Test-PathChainIsRegular([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    $current = Get-Item -LiteralPath $Path -Force
    while ($null -ne $current -and $current.FullName -ne $repoRoot) {
        if (($current.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { return $false }
        if ($current.PSObject.Properties.Name -contains 'LinkType' -and
            -not [string]::IsNullOrWhiteSpace([string]$current.LinkType)) {
            return $false
        }
        $parentPath = Split-Path -Parent $current.FullName
        if ([string]::IsNullOrWhiteSpace($parentPath) -or
            -not $parentPath.StartsWith($repoRoot, [StringComparison]::OrdinalIgnoreCase)) {
            return $false
        }
        if ($parentPath -eq $repoRoot) { break }
        $current = Get-Item -LiteralPath $parentPath -Force
    }
    return $true
}

function Test-GitPathAtRevision([string]$Revision, [string]$Path) {
    $result = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @('-C', $repoRoot, 'cat-file', '-e', "$Revision`:$Path")
    return $result.ExitCode -eq 0
}

function Get-GitTreeOrNull([string]$Revision) {
    $result = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @('-C', $repoRoot, 'rev-parse', "$Revision`^{tree}")
    if ($result.ExitCode -ne 0) { return $null }
    $tree = $result.Stdout.Trim()
    if ($tree -notmatch $gitShaPattern) { return $null }
    return $tree
}

function Add-Failure([string]$Message) {
    $failures.Add($Message)
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}

if ([string]::IsNullOrWhiteSpace($HistoryBase)) {
    Add-Failure 'Evidence attestation validation requires an explicit -HistoryBase; environment fallback is not accepted.'
}

$git = @(Get-Command git -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
if ($null -eq $git) {
    Add-Failure 'Git is required to validate evidence attestations.'
    $gitExecutable = $null
} else {
    $gitExecutable = [IO.Path]::GetFullPath([string]$git.Source)
    $repoRootPrefix = $repoRoot.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if ([string]::Equals($gitExecutable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
        $gitExecutable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        Add-Failure "Refusing to execute a repository-controlled Git binary: $gitExecutable"
        $gitExecutable = $null
    }
}

$historyRevision = $null
if ($failures.Count -eq 0) {
    $historyResult = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @('-C', $repoRoot, 'rev-parse', '--verify', "$HistoryBase`^{commit}")
    $candidate = $historyResult.Stdout.Trim()
    if ($historyResult.ExitCode -ne 0 -or $candidate -notmatch $gitShaPattern) {
        Add-Failure "Evidence attestation validation requires a valid HistoryBase commit: $HistoryBase"
    } else {
        $historyRevision = $candidate
        $ancestorResult = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @(
            '-C', $repoRoot, 'merge-base', '--is-ancestor', $historyRevision, 'HEAD'
        )
        if ($ancestorResult.ExitCode -ne 0) {
            Add-Failure "Evidence attestation HistoryBase must be an ancestor of HEAD: $HistoryBase"
        }
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

$recordRoot = Join-Path $repoRoot 'docs\audits\records'
$records = New-Object System.Collections.Generic.List[object]
if (Test-Path -LiteralPath $recordRoot -PathType Container) {
    foreach ($file in Get-ChildItem -LiteralPath $recordRoot -Filter '*.md' -File) {
        $relativePath = Get-RepoRelativePath $file.FullName
        try {
            $content = Get-Content -LiteralPath $file.FullName -Raw -Encoding UTF8
        } catch {
            Add-Failure "Audit record cannot be read as UTF-8: $relativePath"
            continue
        }
        $frontmatter = Get-Frontmatter $content
        if ($null -eq $frontmatter) { continue }
        $record = [pscustomobject]@{
            Path = $relativePath
            IsNew = -not (Test-GitPathAtRevision $historyRevision $relativePath)
            Status = Get-FrontmatterValue $frontmatter 'status'
            GovernanceContract = Get-FrontmatterValue $frontmatter 'governance_contract'
            WorkflowContract = Get-FrontmatterValue $frontmatter 'workflow_contract_revision'
            AuditId = Get-FrontmatterValue $frontmatter 'audit_id'
            EvidenceRunId = Get-FrontmatterValue $frontmatter 'evidence_run_id'
            EvidenceArtifact = Get-FrontmatterValue $frontmatter 'evidence_artifact'
            EvidenceArgvJson = Get-FrontmatterValue $frontmatter 'evidence_argv_json'
            EvidenceRevision = Get-GitRevision (Get-FrontmatterValue $frontmatter 'evidence_revision')
            EvidenceAttestation = Get-FrontmatterValue $frontmatter 'evidence_attestation'
        }
        $records.Add($record)
    }
}

$requiredRecords = @($records | Where-Object {
    $_.IsNew -and
    $_.Status -eq 'closed' -and
    $_.GovernanceContract -eq 'audit-loop/v3' -and
    $_.WorkflowContract -eq 'audit-runtime/v1'
})

if ($requiredRecords.Count -eq 0) {
    Write-Output "Evidence attestation validation passed: no new closed audit-loop/v3 + audit-runtime/v1 records exist after HistoryBase $historyRevision."
    exit 0
}

$attestationUsage = @{}
foreach ($record in $records) {
    if ([string]::IsNullOrWhiteSpace($record.EvidenceAttestation)) { continue }
    if (-not $attestationUsage.ContainsKey($record.EvidenceAttestation)) {
        $attestationUsage[$record.EvidenceAttestation] = New-Object System.Collections.Generic.List[string]
    }
    $attestationUsage[$record.EvidenceAttestation].Add($record.Path)
}

if ([string]::IsNullOrWhiteSpace($PublicKeyPath)) {
    $PublicKeyPath = $env:AUDIT_RUNTIME_PUBLIC_KEY_PATH
}
if ([string]::IsNullOrWhiteSpace($TrustedKeySha256)) {
    $TrustedKeySha256 = $env:AUDIT_RUNTIME_TRUSTED_KEY_SHA256
}

$temporaryKeyPath = $null
$openssl = @(Get-Command openssl -CommandType Application -ErrorAction SilentlyContinue) | Select-Object -First 1
$opensslExecutable = $null
$trustedKeyId = $null
$canVerifySignatures = $true

try {
    if ([string]::IsNullOrWhiteSpace($PublicKeyPath) -and
        -not [string]::IsNullOrWhiteSpace($env:AUDIT_RUNTIME_PUBLIC_KEY_BASE64)) {
        $temporaryKeyPath = Join-Path ([IO.Path]::GetTempPath()) ("audit-evidence-key-{0}.pem" -f [Guid]::NewGuid().ToString('N'))
        try {
            [IO.File]::WriteAllBytes($temporaryKeyPath, [Convert]::FromBase64String($env:AUDIT_RUNTIME_PUBLIC_KEY_BASE64))
            $PublicKeyPath = $temporaryKeyPath
        } catch {
            Add-Failure 'AUDIT_RUNTIME_PUBLIC_KEY_BASE64 must contain a valid base64-encoded PEM public key.'
            $canVerifySignatures = $false
        }
    }

    if ([string]::IsNullOrWhiteSpace($PublicKeyPath) -or
        -not (Test-Path -LiteralPath $PublicKeyPath -PathType Leaf)) {
        Add-Failure 'Signed evidence records require the externally supplied runtime attestation public key.'
        $canVerifySignatures = $false
    }
    if ($TrustedKeySha256 -notmatch $sha256Pattern) {
        Add-Failure 'Signed evidence records require AUDIT_RUNTIME_TRUSTED_KEY_SHA256 as an external trust anchor.'
        $canVerifySignatures = $false
    }
    if ($null -eq $openssl -or [string]::IsNullOrWhiteSpace([string]$openssl.Source)) {
        Add-Failure 'OpenSSL is required to verify evidence attestation signatures.'
        $canVerifySignatures = $false
    } else {
        $opensslExecutable = [IO.Path]::GetFullPath([string]$openssl.Source)
        if ([string]::Equals($opensslExecutable, $repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
            $opensslExecutable.StartsWith($repoRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            Add-Failure "Refusing to execute a repository-controlled OpenSSL binary: $opensslExecutable"
            $opensslExecutable = $null
            $canVerifySignatures = $false
        }
    }

    if ($canVerifySignatures) {
        $actualKeyHash = (Get-FileHash -LiteralPath $PublicKeyPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actualKeyHash -cne $TrustedKeySha256.ToLowerInvariant()) {
            Add-Failure "Evidence attestation public key does not match the external trust anchor: expected=$($TrustedKeySha256.ToLowerInvariant()) actual=$actualKeyHash"
            $canVerifySignatures = $false
        } else {
            $trustedKeyId = "sha256:$actualKeyHash"
        }
    }

    $originResult = Invoke-NativeCapture -FilePath $gitExecutable -Arguments @('-C', $repoRoot, 'remote', 'get-url', 'origin')
    $originUrl = $originResult.Stdout.Trim()
    if ($originResult.ExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($originUrl)) {
        Add-Failure 'Evidence attestation validation requires a canonical repository origin URL.'
        $originUrl = $null
    }

    $usedAttestationIds = @{}
    foreach ($record in $requiredRecords) {
        if ($record.AuditId -notmatch '^AUD-[0-9]{4,}$') {
            Add-Failure "Closed evidence-bearing audit must declare a valid audit_id: $($record.Path)"
        }
        if ($record.EvidenceRunId -notmatch $uuidV4Pattern) {
            Add-Failure "Closed evidence-bearing audit must declare a lowercase UUIDv4 evidence_run_id: $($record.Path)"
            continue
        }
        $declaredArgvResult = ConvertFrom-StrictEvidenceArgvJson ([string]$record.EvidenceArgvJson)
        if (-not $declaredArgvResult.IsValid) {
            Add-Failure "New closed audit-loop/v3 + audit-runtime/v1 record must declare evidence_argv_json as strict JSON containing a non-empty array of non-empty strings: $($record.Path)"
            continue
        }

        $runId = $record.EvidenceRunId
        $expectedArtifactPath = "docs/evidence/runs/$runId/evidence.json"
        $expectedAttestationPath = "docs/evidence/runs/$runId/attestation.json"

        if ($record.EvidenceArtifact -cne $expectedArtifactPath) {
            Add-Failure "Audit evidence_artifact must exactly match its evidence_run_id: $($record.Path) (expected=$expectedArtifactPath; actual=$($record.EvidenceArtifact))"
        }
        if ([string]::IsNullOrWhiteSpace($record.EvidenceAttestation)) {
            Add-Failure "New closed audit-loop/v3 + audit-runtime/v1 record must reference an externally signed evidence_attestation: $($record.Path)"
            continue
        }
        if ($record.EvidenceAttestation -cne $expectedAttestationPath) {
            Add-Failure "Audit evidence_attestation must exactly match docs/evidence/runs/<evidence_run_id>/attestation.json: $($record.Path) (expected=$expectedAttestationPath; actual=$($record.EvidenceAttestation))"
            continue
        }
        if (-not $attestationUsage.ContainsKey($expectedAttestationPath) -or
            $attestationUsage[$expectedAttestationPath].Count -ne 1) {
            $users = if ($attestationUsage.ContainsKey($expectedAttestationPath)) {
                $attestationUsage[$expectedAttestationPath] -join ', '
            } else {
                'none'
            }
            Add-Failure "Evidence attestation must be single-use and referenced by exactly one audit record: $expectedAttestationPath ($users)"
            continue
        }

        $artifactFullPath = Join-Path $repoRoot $expectedArtifactPath.Replace('/', '\')
        $attestationFullPath = Join-Path $repoRoot $expectedAttestationPath.Replace('/', '\')
        if (-not (Test-PathChainIsRegular $artifactFullPath)) {
            Add-Failure "Evidence artifact is missing or uses a reparse path: $($record.Path) ($expectedArtifactPath)"
            continue
        }
        if (-not (Test-PathChainIsRegular $attestationFullPath)) {
            Add-Failure "Evidence attestation is missing or uses a reparse path: $($record.Path) ($expectedAttestationPath)"
            continue
        }

        if ((Get-Item -LiteralPath $artifactFullPath -Force).Length -gt $maxArtifactBytes) {
            Add-Failure "Evidence artifact exceeds the validation size limit: $($record.Path) ($expectedArtifactPath)"
            continue
        }
        if ((Get-Item -LiteralPath $attestationFullPath -Force).Length -gt $maxAttestationBytes) {
            Add-Failure "Evidence attestation exceeds the validation size limit: $($record.Path) ($expectedAttestationPath)"
            continue
        }

        $artifactBytes = [IO.File]::ReadAllBytes($artifactFullPath)
        $artifactSha256 = Get-Sha256Hex $artifactBytes
        try {
            $artifactText = [Text.UTF8Encoding]::new($false, $true).GetString($artifactBytes)
            $artifact = $artifactText | ConvertFrom-Json
        } catch {
            Add-Failure "Evidence artifact must be valid UTF-8 JSON: $($record.Path) ($expectedArtifactPath)"
            continue
        }
        if ($artifact -isnot [pscustomobject] -or
            $artifact.schema -isnot [string] -or $artifact.schema -cne 'revision-evidence/v1' -or
            $artifact.evidence_run_id -isnot [string] -or $artifact.evidence_run_id -cne $runId -or
            $artifact.evidence_revision -isnot [string] -or $artifact.evidence_revision -notmatch $gitShaPattern -or
            $artifact.evidence_tree -isnot [string] -or $artifact.evidence_tree -notmatch $gitShaPattern -or
            -not (Test-StringArray $artifact.argv) -or
            ($artifact.exit_code -isnot [int] -and $artifact.exit_code -isnot [long]) -or
            [long]$artifact.exit_code -lt [int]::MinValue -or [long]$artifact.exit_code -gt [int]::MaxValue -or
            $artifact.isolation -isnot [pscustomobject] -or
            $artifact.isolation.image -isnot [string] -or
            $artifact.isolation.image_id -isnot [string] -or
            $artifact.isolation.image_id -notmatch '^sha256:[0-9a-f]{64}$') {
            Add-Failure "Evidence artifact is missing typed fields required by the signed binding: $($record.Path) ($expectedArtifactPath)"
            continue
        }

        $expectedTree = if ($null -ne $record.EvidenceRevision) { Get-GitTreeOrNull $record.EvidenceRevision } else { $null }
        if ($null -eq $record.EvidenceRevision -or
            $artifact.evidence_revision -cne $record.EvidenceRevision -or
            $null -eq $expectedTree -or
            $artifact.evidence_tree -cne $expectedTree) {
            Add-Failure "Evidence artifact revision/tree does not exactly match the audit and Git object: $($record.Path) ($expectedArtifactPath)"
            continue
        }
        if ($artifact.isolation.image -cne $approvedImageDigest) {
            Add-Failure "Evidence artifact does not use the approved fixed image digest: $($record.Path) ($($artifact.isolation.image))"
            continue
        }
        if (-not (Test-StringArraysEqual $declaredArgvResult.Items $artifact.argv)) {
            Add-Failure "Audit evidence_argv_json must exactly match the ordered evidence artifact argv: $($record.Path) ($expectedArtifactPath)"
            continue
        }
        if (-not (Test-MeaningfulEvidenceArgv $artifact.argv)) {
            Add-Failure "Evidence artifact argv must invoke a meaningful subject-specific command: $($record.Path) ($expectedArtifactPath)"
            continue
        }

        try {
            $envelopeText = [Text.UTF8Encoding]::new($false, $true).GetString([IO.File]::ReadAllBytes($attestationFullPath))
            $envelope = $envelopeText | ConvertFrom-Json
        } catch {
            Add-Failure "Evidence attestation must be valid UTF-8 JSON: $expectedAttestationPath"
            continue
        }
        $envelopeProperties = @('schema', 'attestation_id', 'algorithm', 'key_id', 'payload_base64', 'signature_base64')
        if (-not (Test-ExactProperties $envelope $envelopeProperties)) {
            Add-Failure "Evidence attestation envelope must contain exactly the required fields: $expectedAttestationPath"
            continue
        }
        if ($envelope.schema -isnot [string] -or $envelope.schema -cne 'revision-evidence-attestation/v1' -or
            $envelope.attestation_id -isnot [string] -or $envelope.attestation_id -notmatch $uuidV4Pattern -or
            $envelope.algorithm -isnot [string] -or $envelope.algorithm -cne 'rsa-sha256' -or
            $envelope.key_id -isnot [string] -or $envelope.key_id -notmatch '^sha256:[0-9a-f]{64}$' -or
            $envelope.payload_base64 -isnot [string] -or [string]::IsNullOrWhiteSpace($envelope.payload_base64) -or
            $envelope.signature_base64 -isnot [string] -or [string]::IsNullOrWhiteSpace($envelope.signature_base64)) {
            Add-Failure "Evidence attestation envelope does not match revision-evidence-attestation/v1: $expectedAttestationPath"
            continue
        }
        if ($null -ne $trustedKeyId -and $envelope.key_id -cne $trustedKeyId) {
            Add-Failure "Evidence attestation key_id does not match the external trust anchor: $expectedAttestationPath"
            continue
        }
        if ($usedAttestationIds.ContainsKey($envelope.attestation_id)) {
            Add-Failure "Evidence attestation_id must be single-use: $($envelope.attestation_id) ($($usedAttestationIds[$envelope.attestation_id]), $expectedAttestationPath)"
            continue
        }
        $usedAttestationIds[$envelope.attestation_id] = $expectedAttestationPath

        try {
            $payloadBytes = [Convert]::FromBase64String($envelope.payload_base64)
            $signatureBytes = [Convert]::FromBase64String($envelope.signature_base64)
        } catch {
            Add-Failure "Evidence attestation payload/signature must be valid base64: $expectedAttestationPath"
            continue
        }
        if ($payloadBytes.Length -eq 0 -or $signatureBytes.Length -eq 0) {
            Add-Failure "Evidence attestation payload/signature cannot be empty: $expectedAttestationPath"
            continue
        }
        if ($payloadBytes.Length -gt $maxPayloadBytes -or $signatureBytes.Length -gt $maxSignatureBytes) {
            Add-Failure "Evidence attestation payload/signature exceeds the validation size limit: $expectedAttestationPath"
            continue
        }
        if (-not $canVerifySignatures) { continue }

        $payloadPath = Join-Path ([IO.Path]::GetTempPath()) ("audit-evidence-payload-{0}.json" -f [Guid]::NewGuid().ToString('N'))
        $signaturePath = Join-Path ([IO.Path]::GetTempPath()) ("audit-evidence-signature-{0}.bin" -f [Guid]::NewGuid().ToString('N'))
        try {
            [IO.File]::WriteAllBytes($payloadPath, $payloadBytes)
            [IO.File]::WriteAllBytes($signaturePath, $signatureBytes)
            $verifyResult = Invoke-NativeCapture -FilePath $opensslExecutable -Arguments @(
                'dgst', '-sha256', '-verify', $PublicKeyPath, '-signature', $signaturePath, $payloadPath
            )
            if ($verifyResult.ExitCode -ne 0) {
                Add-Failure "Evidence attestation signature is invalid: $expectedAttestationPath"
                continue
            }
            try {
                $payloadText = [Text.UTF8Encoding]::new($false, $true).GetString($payloadBytes)
                $payload = $payloadText | ConvertFrom-Json
            } catch {
                Add-Failure "Signed evidence attestation payload must be valid UTF-8 JSON: $expectedAttestationPath"
                continue
            }
        } finally {
            Remove-Item -LiteralPath $payloadPath, $signaturePath -Force -ErrorAction SilentlyContinue
        }

        $payloadProperties = @(
            'schema',
            'attestation_id',
            'repository',
            'audit_id',
            'audit_record_path',
            'evidence_run_id',
            'evidence_artifact_path',
            'evidence_artifact_sha256',
            'evidence_revision',
            'evidence_tree',
            'argv',
            'exit_code',
            'image_digest',
            'image_id',
            'issued_at',
            'expires_at'
        )
        if (-not (Test-ExactProperties $payload $payloadProperties)) {
            Add-Failure "Signed evidence attestation payload must contain exactly the required fields: $expectedAttestationPath"
            continue
        }

        $issuedAt = ConvertTo-DateTimeOffsetOrNull ([string]$payload.issued_at)
        $expiresAt = ConvertTo-DateTimeOffsetOrNull ([string]$payload.expires_at)
        $payloadExitCodeIsInteger = $payload.exit_code -is [int] -or $payload.exit_code -is [long]
        if ($payload.schema -isnot [string] -or $payload.schema -cne 'revision-evidence-binding/v1' -or
            $payload.attestation_id -isnot [string] -or $payload.attestation_id -cne $envelope.attestation_id -or
            $payload.repository -isnot [string] -or $null -eq $originUrl -or $payload.repository -cne $originUrl -or
            $payload.audit_id -isnot [string] -or $payload.audit_id -cne $record.AuditId -or
            $payload.audit_record_path -isnot [string] -or $payload.audit_record_path -cne $record.Path -or
            $payload.evidence_run_id -isnot [string] -or $payload.evidence_run_id -cne $runId -or
            $payload.evidence_artifact_path -isnot [string] -or $payload.evidence_artifact_path -cne $expectedArtifactPath -or
            $payload.evidence_artifact_sha256 -isnot [string] -or $payload.evidence_artifact_sha256 -notmatch $sha256Pattern -or
            $payload.evidence_artifact_sha256 -cne $artifactSha256 -or
            $payload.evidence_revision -isnot [string] -or $payload.evidence_revision -cne $artifact.evidence_revision -or
            $payload.evidence_tree -isnot [string] -or $payload.evidence_tree -cne $artifact.evidence_tree -or
            -not (Test-StringArraysEqual $payload.argv $artifact.argv) -or
            -not (Test-StringArraysEqual $payload.argv $declaredArgvResult.Items) -or
            -not $payloadExitCodeIsInteger -or [long]$payload.exit_code -ne [long]$artifact.exit_code -or
            $payload.image_digest -isnot [string] -or $payload.image_digest -cne $approvedImageDigest -or
            $payload.image_digest -cne $artifact.isolation.image -or
            $payload.image_id -isnot [string] -or $payload.image_id -cne $artifact.isolation.image_id -or
            $null -eq $issuedAt -or $null -eq $expiresAt -or
            $issuedAt -ge $expiresAt -or ($expiresAt - $issuedAt).TotalHours -gt 24) {
            Add-Failure "Signed evidence attestation payload does not exactly bind the repository, audit, artifact bytes, execution fields, approved image, and bounded lifetime: $expectedAttestationPath"
            continue
        }
    }
} finally {
    if ($null -ne $temporaryKeyPath) {
        Remove-Item -LiteralPath $temporaryKeyPath -Force -ErrorAction SilentlyContinue
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output "Evidence attestation validation passed: $($requiredRecords.Count) new closed audit record(s) have externally signed, byte-bound, single-use revision evidence."
