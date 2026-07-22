# Goal Governance Skills installer (Claude Code + Grok Build + GitHub Copilot)
# Run from the target project root. No network access required.
#
# Typical flow:
#   1. Copy this whole skills package into the project root
#      (may rename, e.g. my-governance-skills)
#   2. cd to project root
#   3. .\skills\install.ps1 -Claude -SkillsDir .\skills
#      or: .\my-governance-skills\install.ps1 -All -SkillsDir .\my-governance-skills

param(
    [switch]$Claude,
    [switch]$Grok,
    [switch]$Copilot,
    [switch]$All,
    [switch]$Help,
    [switch]$WithPrimitives,
    [string]$SkillsDir = './skills',
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs
)

$ErrorActionPreference = 'Stop'

function Show-Usage {
    @"
Goal Governance Skills installer

Prerequisites:
  Copy the entire skills package into the target project root first
  (you may rename it, e.g. my-governance-skills). Then run this script
  from the project root.

Usage (run from target project root):
  .\install.ps1 -Claude [-SkillsDir DIR]
  .\install.ps1 -Grok [-SkillsDir DIR]
  .\install.ps1 -Copilot [-SkillsDir DIR] [-WithPrimitives]
  .\install.ps1 -All [-SkillsDir DIR] [-WithPrimitives]
  .\install.ps1 -Help

Options:
  -Claude / --claude       Install Claude Code: .\AGENTS.md + project skills
                           .\.claude\skills\govern\  ->  /govern
                           .\.claude\skills\audit\   ->  /audit
  -Grok / --grok           Install Grok Build project skills
                           .\.grok\skills\govern\  ->  /govern
                           .\.grok\skills\audit\   ->  /audit
  -Copilot / --copilot     Install GitHub Copilot rules -> .\.github\copilot-instructions.md
                           and default slashes -> govern + audit
  -WithPrimitives / --with-primitives
                           Also install advanced Copilot form-fill slash wrappers. Opt-in only.
  -All / --all             Install Claude + Grok + Copilot + prompts/templates/contracts under -SkillsDir
  -SkillsDir / --skills-dir DIR
                           Skills package / destination directory (default: .\skills)
  -Help / --help           Show this help

Behavior:
  - Claude skills -> govern + audit
  - Grok skills -> govern + audit
  - Default Copilot slash surface: /govern + /audit
  - Core orchestrator: prompts\00-govern-orchestrator.md
  - Cross-audit core: prompts\05-independent-audit.md
  - Compatibility contract mirror: contracts\skills-consumer-contract.json
  - Offline only; prompts before overwriting

Examples:
  cd C:\path\to\your-project
  .\skills\install.ps1 -Claude -SkillsDir .\skills
  .\skills\install.ps1 -Grok -SkillsDir .\skills
  .\skills\install.ps1 -All -SkillsDir .\skills
"@
}

function Write-Err([string]$Message) {
    Write-Host "Error: $Message" -ForegroundColor Red
    exit 1
}

function Confirm-Overwrite([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return $true
    }
    $answer = Read-Host "File already exists: $Path`nOverwrite? [y/N]"
    if ($answer -match '^(y|yes)$') {
        return $true
    }
    Write-Host "Skipped: $Path"
    return $false
}

function Get-ResolvedPath([string]$Path, [string]$BaseDir) {
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path $BaseDir $Path))
}

function Test-SamePath([string]$PathA, [string]$PathB) {
    try {
        $a = [System.IO.Path]::GetFullPath($PathA).TrimEnd('\', '/')
        $b = [System.IO.Path]::GetFullPath($PathB).TrimEnd('\', '/')
        return $a.Equals($b, [System.StringComparison]::OrdinalIgnoreCase)
    } catch {
        return $false
    }
}

function Copy-RuleFile {
    param(
        [string]$Source,
        [string]$Destination
    )
    if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) {
        Write-Err "Source file not found: $Source"
    }
    $destDir = Split-Path -Parent $Destination
    if ($destDir -and -not (Test-Path -LiteralPath $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }
    if (Confirm-Overwrite -Path $Destination) {
        Copy-Item -LiteralPath $Source -Destination $Destination -Force
        Write-Host "Installed: $Destination"
    }
}

function Copy-DirMerge {
    param(
        [string]$Source,
        [string]$Destination,
        [string]$Label
    )
    if (-not (Test-Path -LiteralPath $Source -PathType Container)) {
        Write-Err "Source directory not found: $Source"
    }

    if ((Test-Path -LiteralPath $Destination -PathType Container) -and (Test-SamePath $Source $Destination)) {
        Write-Host "Already present: $Destination\  (from $Label)"
        return
    }

    if (Test-Path -LiteralPath $Destination) {
        $answer = Read-Host "Directory already exists: $Destination`nOverwrite contents from $Label? [y/N]"
        if ($answer -notmatch '^(y|yes)$') {
            Write-Host "Skipped: $Destination"
            return
        }
    }
    if (-not (Test-Path -LiteralPath $Destination)) {
        New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    }
    Copy-Item -Path (Join-Path $Source '*') -Destination $Destination -Recurse -Force
    Write-Host "Installed: $Destination\  (from $Label)"
}

function Show-NextSteps {
    param(
        [string]$TargetDir,
        [string]$SkillsDir,
        [string]$PackageRoot
    )
    @"

Done.

Next steps:
  1. Review installed rule file(s) and adjust paths for your project.
  2. Ensure docs\goals\goal-tree.md exists (create if needed).
  3. DEFAULT user path: /govern (orchestrator) + /audit (cross-audit)
     - Core: $SkillsDir\prompts\00-govern-orchestrator.md
     - Cross: $SkillsDir\prompts\05-independent-audit.md
     - Contract: $SkillsDir\contracts\skills-consumer-contract.json
     - Claude: /govern + /audit under .\.claude\skills\
     - Grok:   /govern + /audit under .\.grok\skills\
     - Copilot: govern.prompt.md + audit.prompt.md
  4. Advanced Copilot form-filling slashes only if you used -WithPrimitives.

Project root:  $TargetDir
Skills dir:    $SkillsDir
Package root:  $PackageRoot
"@
}

# Accept GNU-style flags (avoid @($null) which is a 1-element array)
$extraArgs = @()
if ($null -ne $RemainingArgs) {
    $extraArgs = @($RemainingArgs)
}
$i = 0
while ($i -lt $extraArgs.Count) {
    $arg = $extraArgs[$i]
    switch -Regex ($arg) {
        '^--claude$' { $Claude = $true; $i++ }
        '^--grok$' { $Grok = $true; $i++ }
        '^--copilot$' { $Copilot = $true; $i++ }
        '^--all$' { $All = $true; $i++ }
        '^--with-primitives$' { $WithPrimitives = $true; $i++ }
        '^(--help|-h)$' { $Help = $true; $i++ }
        '^--skills-dir$' {
            if ($i + 1 -ge $extraArgs.Count) {
                Write-Err "--skills-dir requires a path argument"
            }
            $SkillsDir = $extraArgs[$i + 1]
            $i += 2
        }
        '^--skills-dir=(.+)$' {
            $SkillsDir = $Matches[1]
            if ([string]::IsNullOrWhiteSpace($SkillsDir)) {
                Write-Err "--skills-dir requires a path argument"
            }
            $i++
        }
        default { Write-Err "Unknown option: $arg (use -Help)" }
    }
}

if ($Help -or (-not $Claude -and -not $Grok -and -not $Copilot -and -not $All)) {
    Show-Usage
    if ($Help) { exit 0 } else { exit 1 }
}

if ($All) {
    $Claude = $true
    $Grok = $true
    $Copilot = $true
    $installExtras = $true
} else {
    $installExtras = $false
}

$PackageRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$TargetDir = (Get-Location).Path
$SkillsDirResolved = Get-ResolvedPath -Path $SkillsDir -BaseDir $TargetDir

$ClaudeAgentsSrc = Join-Path $PackageRoot 'install\claude\AGENTS.md'
$ClaudeGovernSrc = Join-Path $PackageRoot 'install\claude\skills\govern\SKILL.md'
$ClaudeAuditSrc = Join-Path $PackageRoot 'install\claude\skills\audit\SKILL.md'
$GrokGovernSrc = Join-Path $PackageRoot 'install\grok\skills\govern\SKILL.md'
$GrokAuditSrc = Join-Path $PackageRoot 'install\grok\skills\audit\SKILL.md'
$CopilotSrc = Join-Path $PackageRoot 'install\copilot\copilot-instructions.md'
$CopilotWrappersSrc = Join-Path $PackageRoot 'install\copilot\prompts'
$PromptsSrc = Join-Path $PackageRoot 'prompts'
$TemplatesSrc = Join-Path $PackageRoot 'templates'
$ContractsSrc = Join-Path $PackageRoot 'contracts'

if (-not (Test-Path -LiteralPath $PromptsSrc -PathType Container)) {
    Write-Err "Missing package directory: $PromptsSrc"
}
if (-not (Test-Path -LiteralPath $TemplatesSrc -PathType Container)) {
    Write-Err "Missing package directory: $TemplatesSrc"
}
if (-not (Test-Path -LiteralPath $ContractsSrc -PathType Container)) {
    Write-Err "Missing package directory: $ContractsSrc"
}
if (-not (Test-Path -LiteralPath $TargetDir -PathType Container)) {
    Write-Err "Current working directory is not a directory: $TargetDir"
}

if ($Claude) {
    if (-not (Test-Path -LiteralPath $ClaudeAgentsSrc -PathType Leaf)) {
        Write-Err "Missing package file: $ClaudeAgentsSrc"
    }
    if (-not (Test-Path -LiteralPath $ClaudeGovernSrc -PathType Leaf)) {
        Write-Err "Missing package file: $ClaudeGovernSrc"
    }
    if (-not (Test-Path -LiteralPath $ClaudeAuditSrc -PathType Leaf)) {
        Write-Err "Missing package file: $ClaudeAuditSrc"
    }
}
if ($Grok) {
    if (-not (Test-Path -LiteralPath $GrokGovernSrc -PathType Leaf)) {
        Write-Err "Missing package file: $GrokGovernSrc"
    }
    if (-not (Test-Path -LiteralPath $GrokAuditSrc -PathType Leaf)) {
        Write-Err "Missing package file: $GrokAuditSrc"
    }
}
if ($Copilot) {
    if (-not (Test-Path -LiteralPath $CopilotSrc -PathType Leaf)) {
        Write-Err "Missing package file: $CopilotSrc"
    }
    if (-not (Test-Path -LiteralPath $CopilotWrappersSrc -PathType Container)) {
        Write-Err "Missing package directory: $CopilotWrappersSrc"
    }
}

Write-Host "Project root:  $TargetDir"
Write-Host "Skills dir:    $SkillsDirResolved"
Write-Host "Package root:  $PackageRoot"
Write-Host ''

if ($Claude) {
    Copy-RuleFile -Source $ClaudeAgentsSrc -Destination (Join-Path $TargetDir 'AGENTS.md')
    Copy-RuleFile -Source $ClaudeGovernSrc -Destination (Join-Path $TargetDir '.claude\skills\govern\SKILL.md')
    Copy-RuleFile -Source $ClaudeAuditSrc -Destination (Join-Path $TargetDir '.claude\skills\audit\SKILL.md')
    Write-Host 'Claude skills: /govern + /audit'
}

if ($Grok) {
    Copy-RuleFile -Source $GrokGovernSrc -Destination (Join-Path $TargetDir '.grok\skills\govern\SKILL.md')
    Copy-RuleFile -Source $GrokAuditSrc -Destination (Join-Path $TargetDir '.grok\skills\audit\SKILL.md')
    Write-Host 'Grok skills: /govern + /audit'
    $agentsPath = Join-Path $TargetDir 'AGENTS.md'
    if (-not (Test-Path -LiteralPath $agentsPath -PathType Leaf) -and (Test-Path -LiteralPath $ClaudeAgentsSrc -PathType Leaf)) {
        Write-Host 'Note: no AGENTS.md yet; consider -Claude or copy install\claude\AGENTS.md for project rules.'
    }
}

if ($Copilot) {
    $githubDir = Join-Path $TargetDir '.github'
    if (-not (Test-Path -LiteralPath $githubDir)) {
        New-Item -ItemType Directory -Path $githubDir -Force | Out-Null
    }
    Copy-RuleFile -Source $CopilotSrc -Destination (Join-Path $githubDir 'copilot-instructions.md')

    $promptsDir = Join-Path $githubDir 'prompts'
    if (-not (Test-Path -LiteralPath $promptsDir)) {
        New-Item -ItemType Directory -Path $promptsDir -Force | Out-Null
    }
    $wrapperNames = @('govern', 'audit')
    if ($WithPrimitives) {
        $wrapperNames += @('new-goal', 'log-decision', 'update-execution', 'write-audit')
        Write-Host 'Including advanced primitive slash wrappers (-WithPrimitives)'
    } else {
        Write-Host 'Copilot slash surface: /govern + /audit (pass -WithPrimitives for form-fill ops)'
    }
    foreach ($name in $wrapperNames) {
        Copy-RuleFile `
            -Source (Join-Path $CopilotWrappersSrc "$name.md") `
            -Destination (Join-Path $promptsDir "$name.prompt.md")
    }
}

if ($installExtras) {
    if (-not (Test-Path -LiteralPath $SkillsDirResolved)) {
        New-Item -ItemType Directory -Path $SkillsDirResolved -Force | Out-Null
    }
    Copy-DirMerge -Source $PromptsSrc -Destination (Join-Path $SkillsDirResolved 'prompts') -Label 'prompts'
    Copy-DirMerge -Source $TemplatesSrc -Destination (Join-Path $SkillsDirResolved 'templates') -Label 'templates'
    Copy-DirMerge -Source $ContractsSrc -Destination (Join-Path $SkillsDirResolved 'contracts') -Label 'contracts'
}

Show-NextSteps -TargetDir $TargetDir -SkillsDir $SkillsDirResolved -PackageRoot $PackageRoot
