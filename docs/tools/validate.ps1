param(
    [string]$DocsRoot
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$defaultDocsRoot = Join-Path $repoRoot 'docs'
$docsRoot = if ([string]::IsNullOrWhiteSpace($DocsRoot)) { $defaultDocsRoot } else { (Resolve-Path $DocsRoot).Path }
$failures = New-Object System.Collections.Generic.List[string]

$frontmatterExceptions = @(
    'CHANGELOG.md', 'README.md',
    'audits/README.md', 'audits/templates/audit-record.md', 'audits/templates/follow-up-audit-record.md',
    'audits/templates/implementation-acceptance-audit-record.md', 'audits/templates/implementation-audit-record.md',
    'audits/templates/plan-acceptance-audit-record.md', 'audits/templates/plan-audit-record.md',
    'decisions/README.md', 'evidence/README.md', 'implementations/README.md',
    'implementations/templates/implementation-record.md', 'plans/README.md', 'plans/archived/README.md',
    'plans/templates/checklist.md', 'plans/templates/plan.md', 'remediations/README.md',
    'remediations/templates/remediation-record.md', 'scenarios/README.md', 'tools/README.md'
)

function Get-DocsRelativePath([string]$Path) {
    return $Path.Substring($docsRoot.Length + 1).Replace('\', '/')
}

function Get-Frontmatter([string]$Content) {
    $match = [regex]::Match($Content, '\A---\s*\r?\n(?<body>.*?)\r?\n---\s*(?:\r?\n|\z)', 'Singleline')
    if (-not $match.Success) { return $null }
    return $match.Groups['body'].Value
}

function Get-Field([string]$Frontmatter, [string]$Name) {
    if ([string]::IsNullOrWhiteSpace($Frontmatter)) { return $null }
    $match = [regex]::Match($Frontmatter, "(?m)^$([regex]::Escape($Name)):\s*(?<value>.*?)\s*$")
    if (-not $match.Success) { return $null }
    return $match.Groups['value'].Value.Trim()
}

function Get-Values([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -eq 'none') { return @() }
    return @($Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { $_ })
}

function Require-Fields([string]$Frontmatter, [string[]]$Fields, [string]$Label) {
    foreach ($field in $Fields) {
        if ([string]::IsNullOrWhiteSpace((Get-Field $Frontmatter $field))) {
            $failures.Add("$Label is missing frontmatter field: $field")
        }
    }
}

function Test-IndexEntry([string]$IndexContent, [string]$RelativeTarget, [string]$Label) {
    $matches = [regex]::Matches($IndexContent, '(?m)^.*\]\(' + [regex]::Escape($RelativeTarget) + '\).*$')
    if ($matches.Count -ne 1) {
        $failures.Add("$Label must be indexed exactly once: $RelativeTarget")
    }
}

function Test-Findings([string]$Content, [string]$AuditId, [string]$Label) {
    $matches = [regex]::Matches($Content, "(?m)^###\s+$([regex]::Escape($AuditId))-F\d{3}\s+-.*$")
    foreach ($match in $matches) {
        $start = $match.Index
        $tail = $Content.Substring($start + $match.Length)
        $next = [regex]::Match($tail, '(?m)^###\s+')
        $end = if ($next.Success) { $start + $match.Length + $next.Index } else { $Content.Length }
        $section = $Content.Substring($start, $end - $start)
        foreach ($field in @('Severity', 'Evidence', 'Impact', 'Recommendation', 'Owner', 'Disposition')) {
            if ($section -notmatch "(?m)^-\s*$field`:\s*\S") {
                $failures.Add("$Label finding is missing ${field}: $($match.Value.Trim())")
            }
        }
    }
}

function Test-TerminalRecordsImmutable([string]$RelativeRoot, [string]$TerminalPattern, [string]$Label) {
    if ($docsRoot -ne $defaultDocsRoot) { return }
    $tracked = @(& git -C $repoRoot ls-tree -r --name-only HEAD -- "docs/$RelativeRoot" 2>$null)
    foreach ($path in $tracked) {
        if ([string]::IsNullOrWhiteSpace($path)) { continue }
        $fullPath = Join-Path $repoRoot $path
        if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
            $failures.Add("$Label record must not be deleted or moved: $path")
            continue
        }
        $headContent = (& git -C $repoRoot show "HEAD:$path" 2>$null | Out-String)
        if ($headContent -match $TerminalPattern) {
            & git -C $repoRoot diff --quiet HEAD -- $path
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("$Label record is immutable; create a new related record: $path")
            }
        }
    }
}

$markdownFiles = @(Get-ChildItem -LiteralPath $docsRoot -Recurse -File -Filter '*.md')
$contents = @{}
foreach ($file in $markdownFiles) {
    $relative = Get-DocsRelativePath $file.FullName
    $content = Get-Content -Raw -Encoding UTF8 $file.FullName
    $contents[$relative] = $content
    $frontmatter = Get-Frontmatter $content
    if ($relative -notin $frontmatterExceptions -and $null -eq $frontmatter) {
        $failures.Add("Markdown file requires frontmatter: $relative")
    }

    foreach ($link in [regex]::Matches($content, '(?<!\!)\[[^\]]*\]\((?<target>[^)]+)\)')) {
        $target = $link.Groups['target'].Value.Trim().Trim('<', '>')
        if ($target -match '^(?:https?://|mailto:|#)' -or [string]::IsNullOrWhiteSpace($target)) { continue }
        $pathPart = $target.Split('#')[0]
        if ([string]::IsNullOrWhiteSpace($pathPart)) { continue }
        $resolved = [IO.Path]::GetFullPath((Join-Path $file.DirectoryName $pathPart))
        if ($docsRoot -ne $defaultDocsRoot -and -not $resolved.StartsWith($docsRoot, [StringComparison]::OrdinalIgnoreCase)) { continue }
        if (-not (Test-Path -LiteralPath $resolved)) {
            $failures.Add("Broken relative link: $relative -> $target")
        }
    }
}

$planFiles = @(Get-ChildItem -LiteralPath (Join-Path $docsRoot 'plans') -File -Filter 'PLN-*.md' | Where-Object { $_.Name -notlike '*-checklist.md' })
$checklistFiles = @(Get-ChildItem -LiteralPath (Join-Path $docsRoot 'plans') -File -Filter 'PLN-*-checklist.md')
$plans = @{}
foreach ($file in $planFiles) {
    if ($file.Name -notmatch '^(?<id>PLN-\d{4})-(?<subject>[a-z0-9]+(?:-[a-z0-9]+)*)\.md$') {
        $failures.Add("Invalid plan filename: $($file.Name)")
        continue
    }
    $frontmatter = Get-Frontmatter (Get-Content -Raw -Encoding UTF8 $file.FullName)
    Require-Fields $frontmatter @('status', 'plan_id', 'owner', 'created', 'last_updated', 'applies_to') "Plan $($file.Name)"
    if ((Get-Field $frontmatter 'plan_id') -ne $Matches['id']) { $failures.Add("Plan ID does not match filename: $($file.Name)") }
    $plans[$Matches['id']] = @{ Subject = $Matches['subject']; File = $file }
    $expectedChecklist = Join-Path $file.DirectoryName ($file.BaseName + '-checklist.md')
    if (-not (Test-Path -LiteralPath $expectedChecklist -PathType Leaf)) { $failures.Add("Plan is missing checklist: $($file.Name)") }
}
foreach ($file in $checklistFiles) {
    if ($file.Name -notmatch '^(?<id>PLN-\d{4})-(?<subject>[a-z0-9]+(?:-[a-z0-9]+)*)-checklist\.md$') {
        $failures.Add("Invalid checklist filename: $($file.Name)")
        continue
    }
    $frontmatter = Get-Frontmatter (Get-Content -Raw -Encoding UTF8 $file.FullName)
    Require-Fields $frontmatter @('status', 'plan_id', 'owner', 'created', 'last_updated', 'applies_to') "Checklist $($file.Name)"
    if ((Get-Field $frontmatter 'plan_id') -ne $Matches['id']) { $failures.Add("Checklist ID does not match filename: $($file.Name)") }
    $expectedPlan = Join-Path $file.DirectoryName ($file.Name -replace '-checklist\.md$', '.md')
    if (-not (Test-Path -LiteralPath $expectedPlan -PathType Leaf)) { $failures.Add("Checklist is missing plan: $($file.Name)") }
}

$auditDirectory = Join-Path $docsRoot 'audits\records'
$auditIndex = Get-Content -Raw -Encoding UTF8 (Join-Path $docsRoot 'audits\README.md')
$auditIds = New-Object System.Collections.Generic.HashSet[string]
$auditRefs = @()
foreach ($file in @(Get-ChildItem -LiteralPath $auditDirectory -File -Filter 'AUD-*.md')) {
    if ($file.Name -notmatch '^(?<id>AUD-\d{4})-\d{8}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
        $failures.Add("Invalid audit filename: $($file.Name)")
        continue
    }
    $id = $Matches['id']; [void]$auditIds.Add($id)
    $content = Get-Content -Raw -Encoding UTF8 $file.FullName
    $frontmatter = Get-Frontmatter $content
    Require-Fields $frontmatter @('status', 'audit_id', 'auditor', 'audit_type', 'scope', 'subject', 'baseline', 'started_at', 'last_updated') "Audit $($file.Name)"
    if ((Get-Field $frontmatter 'audit_id') -ne $id) { $failures.Add("Audit ID does not match filename: $($file.Name)") }
    $status = Get-Field $frontmatter 'status'
    if ($status -notin @('open', 'closed', 'superseded')) { $failures.Add("Invalid audit status: $($file.Name) ($status)") }
    if ($status -ne 'open' -and (Get-Field $frontmatter 'completed_at') -in @($null, '', 'pending')) { $failures.Add("Terminal audit requires completed_at: $($file.Name)") }
    $schema = Get-Field $frontmatter 'audit_schema'
    if ($status -eq 'open' -and $schema -in @('plan-audit/v2', 'plan-acceptance/v2', 'implementation-audit/v2', 'implementation-acceptance/v2') -and [string]::IsNullOrWhiteSpace((Get-Field $frontmatter 'evidence_revision'))) {
        $failures.Add("Revision-bound audit requires evidence_revision: $($file.Name)")
    }
    if ($schema -eq 'plan-audit/v2' -and $content -notmatch '<!--\s*plan-checklist-audit:\s*PLN-\d{4}\s*-->') { $failures.Add("Plan audit requires checklist matrix marker: $($file.Name)") }
    if ($schema -eq 'plan-acceptance/v2' -and $content -notmatch '<!--\s*plan-acceptance-audit:\s*PLN-\d{4}\s*-->') { $failures.Add("Plan acceptance requires matrix marker: $($file.Name)") }
    if ($schema -eq 'implementation-audit/v2' -and $content -notmatch '<!--\s*implementation-audit:\s*IMP-\d{4}\s*-->') { $failures.Add("Implementation audit requires matrix marker: $($file.Name)") }
    if ($schema -eq 'implementation-acceptance/v2' -and $content -notmatch '<!--\s*implementation-acceptance-audit:\s*PLN-\d{4}\s*-->') { $failures.Add("Implementation acceptance requires matrix marker: $($file.Name)") }
    if ($status -eq 'open') { Test-Findings $content $id "Audit $($file.Name)" }
    Test-IndexEntry $auditIndex ("./records/" + $file.Name) 'Audit'
    $auditRefs += Get-Values (Get-Field $frontmatter 'related_audits')
}

$remediationDirectory = Join-Path $docsRoot 'remediations\records'
$remediationIndex = Get-Content -Raw -Encoding UTF8 (Join-Path $docsRoot 'remediations\README.md')
$remediationIds = New-Object System.Collections.Generic.HashSet[string]
$remediationAuditRefs = @()
foreach ($file in @(Get-ChildItem -LiteralPath $remediationDirectory -File -Filter 'REM-*.md')) {
    if ($file.Name -notmatch '^(?<id>REM-\d{4})-\d{8}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') { $failures.Add("Invalid remediation filename: $($file.Name)"); continue }
    $id = $Matches['id']; [void]$remediationIds.Add($id)
    $content = Get-Content -Raw -Encoding UTF8 $file.FullName; $frontmatter = Get-Frontmatter $content
    Require-Fields $frontmatter @('status', 'remediation_id', 'implementer', 'scope', 'source_audits', 'source_findings', 'baseline', 'started_at', 'last_updated') "Remediation $($file.Name)"
    if ((Get-Field $frontmatter 'remediation_id') -ne $id) { $failures.Add("Remediation ID does not match filename: $($file.Name)") }
    if ((Get-Field $frontmatter 'status') -notin @('in-progress', 'completed', 'partial', 'blocked', 'superseded')) { $failures.Add("Invalid remediation status: $($file.Name)") }
    Test-IndexEntry $remediationIndex ("./records/" + $file.Name) 'Remediation'
    $remediationAuditRefs += Get-Values (Get-Field $frontmatter 'source_audits')
}

$implementationDirectory = Join-Path $docsRoot 'implementations\records'
$implementationIndex = Get-Content -Raw -Encoding UTF8 (Join-Path $docsRoot 'implementations\README.md')
$implementationIds = New-Object System.Collections.Generic.HashSet[string]
if (Test-Path -LiteralPath $implementationDirectory) {
    foreach ($file in @(Get-ChildItem -LiteralPath $implementationDirectory -File -Filter 'IMP-*.md')) {
        if ($file.Name -notmatch '^(?<id>IMP-\d{4})-\d{8}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') { $failures.Add("Invalid implementation filename: $($file.Name)"); continue }
        $id = $Matches['id']; [void]$implementationIds.Add($id)
        $content = Get-Content -Raw -Encoding UTF8 $file.FullName; $frontmatter = Get-Frontmatter $content
        Require-Fields $frontmatter @('status', 'implementation_id', 'implementer', 'scope', 'related_plans', 'baseline', 'started_at', 'last_updated') "Implementation $($file.Name)"
        if ((Get-Field $frontmatter 'implementation_id') -ne $id) { $failures.Add("Implementation ID does not match filename: $($file.Name)") }
        if ((Get-Field $frontmatter 'status') -notin @('in-progress', 'completed', 'partial', 'blocked', 'superseded')) { $failures.Add("Invalid implementation status: $($file.Name)") }
        Test-IndexEntry $implementationIndex ("./records/" + $file.Name) 'Implementation'
    }
}

foreach ($reference in $auditRefs + $remediationAuditRefs) {
    if ($reference -match '^AUD-\d{4}$' -and -not $auditIds.Contains($reference)) { $failures.Add("Reference points to missing audit: $reference") }
}

Test-TerminalRecordsImmutable 'audits/records' '(?m)^status:\s*(?:closed|superseded)\s*$' 'Audit'
Test-TerminalRecordsImmutable 'remediations/records' '(?m)^status:\s*(?:completed|partial|blocked|superseded)\s*$' 'Remediation'
Test-TerminalRecordsImmutable 'implementations/records' '(?m)^status:\s*(?:completed|partial|blocked|superseded)\s*$' 'Implementation'

if ($docsRoot -eq $defaultDocsRoot) {
    $previousPreference = $ErrorActionPreference; $ErrorActionPreference = 'Continue'
    $diffOutput = & git -C $repoRoot diff HEAD --check 2>$null; $diffCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    if ($diffCode -ne 0) { foreach ($line in $diffOutput) { $failures.Add("git diff HEAD --check: $line") } }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output "Validated $($markdownFiles.Count) Markdown files: structure, links, records, indexes, and immutable history passed."
