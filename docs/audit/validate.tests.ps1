$ErrorActionPreference = 'Stop'

$fixtureRoot = Join-Path $PSScriptRoot ('.validate-fixture-' + [Guid]::NewGuid().ToString('N'))
$validator = Join-Path $PSScriptRoot 'validate.ps1'

function Invoke-Validator([string]$DocsRoot) {
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $shell = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($null -eq $shell) {
        $shell = Get-Command powershell -ErrorAction Stop
    }
    $output = & $shell.Source -NoProfile -ExecutionPolicy Bypass -File $validator -DocsRoot $DocsRoot 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    return @{
        ExitCode = $exitCode
        Output = ($output | Out-String).Trim()
    }
}

try {
    New-Item -ItemType Directory -Path $fixtureRoot | Out-Null
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

    $validResult = Invoke-Validator $fixtureRoot
    if ($validResult.ExitCode -ne 0) {
        throw "validator rejected valid fixture: $($validResult.Output)"
    }

    Set-Content -LiteralPath (Join-Path $fixtureRoot 'invalid.md') -Value ($frontmatter + "`n[Missing](./does-not-exist.md#missing)") -Encoding UTF8
    $invalidResult = Invoke-Validator $fixtureRoot
    if ($invalidResult.ExitCode -eq 0) {
        throw 'validator accepted invalid fixture'
    }
    if ($invalidResult.Output -notmatch 'Missing relative link target') {
        throw "validator failed for an unexpected reason: $($invalidResult.Output)"
    }

    Write-Output 'Validator self-test passed: valid fixture accepted, missing-link fixture rejected.'
} finally {
    if (Test-Path -LiteralPath $fixtureRoot) {
        Remove-Item -LiteralPath $fixtureRoot -Recurse -Force
    }
}
