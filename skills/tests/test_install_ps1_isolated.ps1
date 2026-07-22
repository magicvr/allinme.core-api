# F-018 · PowerShell isolated install smoke test
# Run from anywhere:
#   powershell -NoProfile -ExecutionPolicy Bypass -File skills/tests/test_install_ps1_isolated.ps1
# Exit 0 = pass; non-zero = fail. Prints JSON-ish summary lines for logs.

$ErrorActionPreference = 'Stop'

$TestsDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$PackageRoot = Split-Path -Parent $TestsDir
$InstallPs1 = Join-Path $PackageRoot 'install.ps1'

if (-not (Test-Path -LiteralPath $InstallPs1 -PathType Leaf)) {
    Write-Error "install.ps1 not found: $InstallPs1"
    exit 2
}

$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("gg-skills-install-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $TempRoot -Force | Out-Null
$SkillsDest = Join-Path $TempRoot 'skills'

Write-Host "F-018 isolated install"
Write-Host "  package: $PackageRoot"
Write-Host "  target:  $TempRoot"

try {
    Push-Location $TempRoot
    & $InstallPs1 -All -SkillsDir $SkillsDest
    if ($LASTEXITCODE -ne 0 -and $null -ne $LASTEXITCODE) {
        throw "install.ps1 exited with code $LASTEXITCODE"
    }
    Pop-Location

    $required = @(
        (Join-Path $TempRoot 'AGENTS.md'),
        (Join-Path $TempRoot '.claude\skills\govern\SKILL.md'),
        (Join-Path $TempRoot '.claude\skills\audit\SKILL.md'),
        (Join-Path $TempRoot '.grok\skills\govern\SKILL.md'),
        (Join-Path $TempRoot '.grok\skills\audit\SKILL.md'),
        (Join-Path $TempRoot '.github\copilot-instructions.md'),
        (Join-Path $TempRoot '.github\prompts\govern.prompt.md'),
        (Join-Path $TempRoot '.github\prompts\audit.prompt.md'),
        (Join-Path $SkillsDest 'prompts\00-govern-orchestrator.md'),
        (Join-Path $SkillsDest 'prompts\05-independent-audit.md'),
        (Join-Path $SkillsDest 'templates\workspace-context.md'),
        (Join-Path $SkillsDest 'contracts\skills-consumer-contract.schema.json'),
        (Join-Path $SkillsDest 'contracts\skills-consumer-contract.json')
    )

    $missing = @()
    foreach ($path in $required) {
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            $missing += $path
        }
    }

    $forbidden = @(
        (Join-Path $TempRoot '.github\prompts\new-goal.prompt.md')
    )
    $leaked = @()
    foreach ($path in $forbidden) {
        if (Test-Path -LiteralPath $path -PathType Leaf) {
            $leaked += $path
        }
    }

    $governText = Get-Content -LiteralPath (Join-Path $TempRoot '.claude\skills\govern\SKILL.md') -Raw -Encoding UTF8
    $auditText = Get-Content -LiteralPath (Join-Path $TempRoot '.claude\skills\audit\SKILL.md') -Raw -Encoding UTF8

    $contentOk = $true
    if ($governText -notmatch '00-govern-orchestrator') {
        Write-Host 'FAIL: Claude govern skill missing 00-govern-orchestrator ref'
        $contentOk = $false
    }
    if ($auditText -notmatch '05-independent-audit') {
        Write-Host 'FAIL: Claude audit skill missing 05-independent-audit ref'
        $contentOk = $false
    }
    if ($governText -notmatch 'workspace-<NNN>-<slug>/workspace\.md') {
        Write-Host 'FAIL: Claude govern skill missing workspace context ref'
        $contentOk = $false
    }
    if ($auditText -notmatch 'workspace-<NNN>-<slug>/workspace\.md') {
        Write-Host 'FAIL: Claude audit skill missing workspace context ref'
        $contentOk = $false
    }

    if ($missing.Count -gt 0) {
        Write-Host 'FAIL: missing required install outputs:'
        $missing | ForEach-Object { Write-Host "  - $_" }
        exit 1
    }
    if ($leaked.Count -gt 0) {
        Write-Host 'FAIL: advanced primitive prompts installed without -WithPrimitives:'
        $leaked | ForEach-Object { Write-Host "  - $_" }
        exit 1
    }
    if (-not $contentOk) {
        exit 1
    }

    Write-Host 'PASS: isolated -All install produced /govern + /audit surface; no form-fill primitives.'
    Write-Host "  evidence_dir=$TempRoot"
    exit 0
}
catch {
    Write-Host "FAIL: $($_.Exception.Message)"
    if ((Get-Location).Path -eq $TempRoot) { Pop-Location }
    exit 1
}
finally {
    if (Test-Path -LiteralPath $TempRoot) {
        try {
            Remove-Item -LiteralPath $TempRoot -Recurse -Force -ErrorAction SilentlyContinue
        } catch {
            Write-Host "WARN: could not remove temp dir: $TempRoot"
        }
    }
}
