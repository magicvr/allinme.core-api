param([string]$RepositoryRoot)

$ErrorActionPreference = 'Stop'
$repoRoot = if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) { (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path } else { (Resolve-Path $RepositoryRoot).Path }
$failures = New-Object System.Collections.Generic.List[string]

$names = @(
    'backend-plan-audit', 'backend-plan-acceptance-audit', 'backend-plan-audit-until-ready',
    'backend-fix-audit-findings', 'backend-follow-up-audit', 'backend-implement-plan',
    'backend-implementation-audit', 'backend-implementation-acceptance-audit',
    'backend-implement-audit-until-complete'
)

foreach ($name in $names) {
    $promptPath = Join-Path $repoRoot ".github\prompts\$name.prompt.md"
    $skillPath = Join-Path $repoRoot ".agents\skills\$name\SKILL.md"
    $metadataPath = Join-Path $repoRoot ".agents\skills\$name\agents\openai.yaml"
    foreach ($path in @($promptPath, $skillPath, $metadataPath)) {
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { $failures.Add("Missing workflow asset: $path") }
    }
    if (-not (Test-Path -LiteralPath $promptPath) -or -not (Test-Path -LiteralPath $skillPath)) { continue }
    $prompt = Get-Content -Raw -Encoding UTF8 $promptPath
    $skill = Get-Content -Raw -Encoding UTF8 $skillPath
    if (-not $skill.Contains(".github/prompts/$name.prompt.md")) {
        $failures.Add("Skill must delegate to its prompt as the single specification: $name")
    }
    if (-not $prompt.Contains("name: $name")) { $failures.Add("Prompt name mismatch: $name") }
    if ((Get-Content -Raw -Encoding UTF8 $metadataPath) -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') { $failures.Add("Skill must require explicit invocation: $name") }
}

foreach ($name in @('backend-plan-acceptance-audit', 'backend-follow-up-audit', 'backend-implementation-audit', 'backend-implementation-acceptance-audit')) {
    $content = Get-Content -Raw -Encoding UTF8 (Join-Path $repoRoot ".github\prompts\$name.prompt.md")
    if (-not $content.Contains('runtime_context_ref')) { $failures.Add("Independent audit must require a separate runtime context: $name") }
}

$planLoop = Get-Content -Raw -Encoding UTF8 (Join-Path $repoRoot '.github\prompts\backend-plan-audit-until-ready.prompt.md')
if (-not $planLoop.Contains('backend-follow-up-audit') -or -not $planLoop.Contains('backend-fix-audit-findings') -or -not $planLoop.Contains('MAX_CYCLES=8')) {
    $failures.Add('Plan loop must preserve verification-first ordering, bounded cycles, and stagnation stop')
}
$implementationLoop = Get-Content -Raw -Encoding UTF8 (Join-Path $repoRoot '.github\prompts\backend-implement-audit-until-complete.prompt.md')
if ((-not $implementationLoop.Contains('verification=pending') -and -not $implementationLoop.Contains('backend-follow-up-audit')) -or -not $implementationLoop.Contains('MAX_CYCLES=12') -or -not $implementationLoop.Contains('acceptance_next_action')) {
    $failures.Add('Implementation loop must preserve verification-first ordering, bounded cycles, and explicit routing')
}

$activeFiles = @(
    Get-ChildItem (Join-Path $repoRoot '.github\prompts') -Filter 'backend-*.prompt.md' -File
    Get-ChildItem (Join-Path $repoRoot '.agents\skills') -Recurse -Filter 'SKILL.md' -File
    Get-ChildItem (Join-Path $repoRoot 'docs\audits\templates') -Filter '*.md' -File
    Get-Item (Join-Path $repoRoot 'docs\audits\README.md'), (Join-Path $repoRoot 'docs\remediations\README.md'), (Join-Path $repoRoot 'docs\implementations\README.md')
)
$forbidden = 'runtime_context_attestation|evidence_attestation|invoke-governance-transaction|update-loop-run-state|governance-loop-run|refs/allinme/governance-head|external signer|trust anchor'
foreach ($file in $activeFiles) {
    if ((Get-Content -Raw -Encoding UTF8 $file.FullName) -match $forbidden) { $failures.Add("Out-of-scope trusted-runtime mechanism remains in active workflow specification: $($file.FullName)") }
}

if ($failures.Count -gt 0) { $failures | ForEach-Object { [Console]::Error.WriteLine($_) }; exit 1 }
Write-Output 'Audit workflow contracts passed: single-source prompts, independent review, verification-first routing, bounded loops, and no trusted-runtime expansion.'
