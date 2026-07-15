$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$validator = Join-Path $PSScriptRoot 'validate.ps1'
$fixtureRoot = Join-Path ([IO.Path]::GetTempPath()) ("allinme-docs-validator-" + [guid]::NewGuid().ToString('N'))
$powershellPath = (Get-Process -Id $PID).Path

function Invoke-Validator([string]$DocsRoot) {
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = & $powershellPath -NoProfile -ExecutionPolicy Bypass -File $validator -DocsRoot $DocsRoot 2>&1
    $code = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return @{ Code = $code; Output = ($output -join "`n") }
}

function Assert-Pass([string]$Label, [string]$DocsRoot) {
    $result = Invoke-Validator $DocsRoot
    if ($result.Code -ne 0) { throw "$Label should pass:`n$($result.Output)" }
}

function Assert-Fail([string]$Label, [string]$DocsRoot, [string]$Pattern) {
    $result = Invoke-Validator $DocsRoot
    if ($result.Code -eq 0 -or $result.Output -notmatch $Pattern) { throw "$Label should fail with '$Pattern':`n$($result.Output)" }
}

try {
    Copy-Item -LiteralPath (Join-Path $repoRoot 'docs') -Destination $fixtureRoot -Recurse
    Assert-Pass 'current documentation fixture' $fixtureRoot

    $brokenLinkRoot = "$fixtureRoot-broken-link"; Copy-Item $fixtureRoot $brokenLinkRoot -Recurse
    Add-Content -LiteralPath (Join-Path $brokenLinkRoot '00-overview.md') -Value "`n[broken](./missing.md)"
    Assert-Fail 'broken relative link' $brokenLinkRoot 'Broken relative link'

    $missingChecklistRoot = "$fixtureRoot-missing-checklist"; Copy-Item $fixtureRoot $missingChecklistRoot -Recurse
    Remove-Item -LiteralPath (Join-Path $missingChecklistRoot 'plans\PLN-0005-phase-05-attachment-lifecycle-checklist.md')
    Assert-Fail 'missing checklist' $missingChecklistRoot 'Plan is missing checklist'

    $missingFieldRoot = "$fixtureRoot-missing-field"; Copy-Item $fixtureRoot $missingFieldRoot -Recurse
    $auditPath = Join-Path $missingFieldRoot 'audits\records\AUD-0001-20260714-codex-repository-docs-governance.md'
    (Get-Content -Raw -Encoding UTF8 $auditPath) -replace '(?m)^audit_type:.*\r?\n', '' | Set-Content -LiteralPath $auditPath -Encoding UTF8
    Assert-Fail 'missing audit field' $missingFieldRoot 'missing frontmatter field: audit_type'

    $duplicateIndexRoot = "$fixtureRoot-duplicate-index"; Copy-Item $fixtureRoot $duplicateIndexRoot -Recurse
    $auditIndexPath = Join-Path $duplicateIndexRoot 'audits\README.md'
    Add-Content -LiteralPath $auditIndexPath -Value "`n- [duplicate](./records/AUD-0001-20260714-codex-repository-docs-governance.md)"
    Assert-Fail 'duplicate audit index' $duplicateIndexRoot 'indexed exactly once'

    Write-Output 'Documentation validator tests passed: valid structure, link failure, plan pairing, required audit fields, and unique indexes.'
    exit 0
} finally {
    Get-ChildItem ([IO.Path]::GetTempPath()) -Directory -Filter 'allinme-docs-validator-*' -ErrorAction SilentlyContinue | Remove-Item -Recurse -Force
}
