param(
    [string]$DocsRoot
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
if ([string]::IsNullOrWhiteSpace($DocsRoot)) {
    $docsRoot = Join-Path $repoRoot 'docs'
} else {
    $docsRoot = (Resolve-Path $DocsRoot).Path
}
$frontmatterExceptions = @(
    'CHANGELOG.md',
    'README.md',
    'audits/README.md',
    'audits/templates/audit-record.md',
    'audits/templates/plan-audit-record.md',
    'decisions/README.md',
    'evidence/README.md',
    'plans/README.md',
    'plans/archived/README.md',
    'plans/templates/checklist.md',
    'plans/templates/plan.md',
    'remediations/README.md',
    'remediations/templates/remediation-record.md',
    'scenarios/README.md',
    'tools/README.md'
)

function Get-RepoRelativePath([string]$Path) {
    return $Path.Substring($repoRoot.Length + 1).Replace('\', '/')
}

function Get-DocsRelativePath([string]$Path) {
    return $Path.Substring($docsRoot.Length + 1).Replace('\', '/')
}

function Get-FrontmatterValue([string]$Frontmatter, [string]$Field) {
    $match = [regex]::Match($Frontmatter, "(?m)^${Field}:\s*(?<value>.+?)\s*$")
    if (-not $match.Success) {
        return $null
    }
    return $match.Groups['value'].Value
}

function Get-IndexLines([string]$Content, [string]$Target) {
    $pattern = '(?m)^.*\]\(' + [regex]::Escape($Target) + '\).*$'
    return [regex]::Matches($Content, $pattern)
}

function Get-ListValues([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -eq 'none') {
        return @()
    }
    return @($Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Test-PhaseFiveEdge([hashtable]$Dag, [string]$Prerequisite, [string]$Dependent) {
    return $Dag.ContainsKey($Dependent) -and @($Dag[$Dependent]) -contains $Prerequisite
}

function Get-PhaseFiveDag([string]$Content, [System.Collections.Generic.List[string]]$Failures) {
    $dag = @{}
    $rows = [regex]::Matches($Content, '(?m)^\|\s*(?<package>WP-[A-Za-z0-9-]+)\s*\|(?<items>[^|]*)\|(?<owner>[^|]*)\|(?<inputs>[^|]*)\|(?<timebox>[^|]*)\|(?<evidence>[^|]*)\|\s*$')
    foreach ($row in $rows) {
        $package = $row.Groups['package'].Value
        if ($dag.ContainsKey($package)) {
            $Failures.Add("PLN-0005 dependency DAG contains duplicate work package: $package")
            continue
        }
        $dependencies = @([regex]::Matches($row.Groups['inputs'].Value, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value } | Select-Object -Unique)
        $dag[$package] = $dependencies
    }

    $expectedPackages = @('WP-Facts', 'WP-Schema-Recovery', 'WP-HTTP-Order', 'WP-Lock', 'WP-Baseline-Evidence', 'WP-Files', 'WP-Runtime', 'WP-Release')
    foreach ($package in $expectedPackages) {
        if (-not $dag.ContainsKey($package)) {
            $Failures.Add("PLN-0005 dependency DAG is missing work package: $package")
        }
    }
    foreach ($package in @($dag.Keys)) {
        if ($package -notin $expectedPackages) {
            $Failures.Add("PLN-0005 dependency DAG contains unknown work package: $package")
        }
        foreach ($dependency in @($dag[$package])) {
            if ($dependency -eq $package) {
                $Failures.Add("PLN-0005 dependency DAG contains a self dependency: $package")
            } elseif (-not $dag.ContainsKey($dependency)) {
                $Failures.Add("PLN-0005 dependency DAG references unknown dependency: $package -> $dependency")
            }
        }
    }

    $requiredDependencies = @{
        'WP-Facts' = @()
        'WP-Schema-Recovery' = @('WP-Facts')
        'WP-HTTP-Order' = @('WP-Facts')
        'WP-Lock' = @('WP-Facts')
        'WP-Baseline-Evidence' = @('WP-Facts')
        'WP-Files' = @('WP-Lock')
        'WP-Runtime' = @('WP-Lock')
        'WP-Release' = @('WP-Facts', 'WP-Schema-Recovery', 'WP-HTTP-Order', 'WP-Lock', 'WP-Baseline-Evidence', 'WP-Files', 'WP-Runtime')
    }
    foreach ($package in $requiredDependencies.Keys) {
        if (-not $dag.ContainsKey($package)) {
            continue
        }
        $actual = @($dag[$package] | Sort-Object)
        $expected = @($requiredDependencies[$package] | Sort-Object)
        if (($actual -join ',') -ne ($expected -join ',')) {
            $Failures.Add("PLN-0005 dependency DAG inputs do not match the tracked contract for ${package}: expected [$($expected -join ', ')], found [$($actual -join ', ')]")
        }
    }

    $remaining = @{}
    foreach ($package in $dag.Keys) {
        $remaining[$package] = @($dag[$package]).Count
    }
    $ready = New-Object System.Collections.Generic.Queue[string]
    foreach ($package in $remaining.Keys) {
        if ($remaining[$package] -eq 0) {
            $ready.Enqueue($package)
        }
    }
    $processed = 0
    while ($ready.Count -gt 0) {
        $completed = $ready.Dequeue()
        $processed++
        foreach ($dependent in @($dag.Keys)) {
            if (@($dag[$dependent]) -contains $completed) {
                $remaining[$dependent]--
                if ($remaining[$dependent] -eq 0) {
                    $ready.Enqueue($dependent)
                }
            }
        }
    }
    if ($dag.Count -gt 0 -and $processed -ne $dag.Count) {
        $Failures.Add('PLN-0005 dependency DAG contains a cycle')
    }
    return $dag
}

function Test-PhaseFiveDependencyStatements([string]$Content, [hashtable]$Dag, [System.Collections.Generic.List[string]]$Failures) {
    $contentWithoutDagRows = [regex]::Replace($Content, '(?m)^\|\s*WP-[A-Za-z0-9-]+\s*\|.*$', '')
    foreach ($line in [regex]::Split($contentWithoutDagRows, '\r?\n')) {
        if ($line -notmatch 'WP-[A-Za-z0-9-]+') {
            continue
        }
        foreach ($match in [regex]::Matches($line, '(?<prerequisite>WP-[A-Za-z0-9-]+)\s*(?:->|\u2192)\s*(?<dependent>WP-[A-Za-z0-9-]+)')) {
            if (-not (Test-PhaseFiveEdge $Dag $match.Groups['prerequisite'].Value $match.Groups['dependent'].Value)) {
                $Failures.Add("PLN-0005 dependency statement is not present in the tracked DAG: $($match.Value)")
            }
        }
        foreach ($match in [regex]::Matches($line, '(?i)(?<dependent>WP-[A-Za-z0-9-]+).*?(?:depends?\s+on|\u4F9D\u8D56)\s*(?<prerequisite>WP-[A-Za-z0-9-]+)')) {
            if (-not (Test-PhaseFiveEdge $Dag $match.Groups['prerequisite'].Value $match.Groups['dependent'].Value)) {
                $Failures.Add("PLN-0005 dependency statement contradicts the tracked DAG: $($match.Value)")
            }
        }
        foreach ($match in [regex]::Matches($line, '(?i)(?<first>WP-[A-Za-z0-9-]+).*?(?:before|precedes?|\u5148\u4E8E|\u65E9\u4E8E).*?(?<second>WP-[A-Za-z0-9-]+)')) {
            $first = $match.Groups['first'].Value
            $second = $match.Groups['second'].Value
            if (-not (Test-PhaseFiveEdge $Dag $first $second)) {
                $Failures.Add("PLN-0005 dependency ordering contradicts or is absent from the tracked DAG: $($match.Value)")
            }
        }
        if ($line -notmatch '(?i)(reject|must\s+reject|\u62D2\u7EDD)') {
            foreach ($match in [regex]::Matches($line, '(?i)(?<dependent>WP-[A-Za-z0-9-]+).*?(?:does\s+not|doesn''t|need\s+not|without|\u4E0D\u4F9D\u8D56|\u65E0\u9700\u7B49\u5F85).*?(?:depend(?:ing)?\s+on\s+)?(?<prerequisite>WP-[A-Za-z0-9-]+)')) {
                if (Test-PhaseFiveEdge $Dag $match.Groups['prerequisite'].Value $match.Groups['dependent'].Value) {
                    $Failures.Add("PLN-0005 dependency statement denies a tracked edge: $($match.Value)")
                }
            }
        }
    }
}

function Test-PhaseFiveDeploymentClauses([string]$Content, [System.Collections.Generic.List[string]]$Failures) {
    $liveEvidence = '(?i)(live\s+(?:deployment|profile|supervisor)|live-test|real\s+(?:deployment|profile|supervisor)|cleanup\s+schedule|watchdog(?:/|\s+and\s+)recovery|ENOSPC.{0,40}(?:run|evidence)|\u771F\u5B9E(?:\u9636\u6BB5\u4E94\s*)?(?:binary|\u76D1\u7763\u5668|\u90E8\u7F72)|cleanup\s*\u8C03\u5EA6|watchdog/recovery|ENOSPC.{0,40}(?:run|Evidence|\u8BC1\u636E)|\u90E8\u7F72\s*profile\s*run|\u5B9E\u6D4B.{0,20}(?:\u90E8\u7F72|profile))'
    $obligation = '(?i)(must|required?|shall|before\s+P0|P0.{0,20}(?:complete|gate)|\u5FC5\u987B|\u8981\u6C42|\u5B8C\u6210\u524D|\u963B\u585E|\u4E0D\u5F97\u8FDB\u5165)'
    $deferral = '(?i)(not\s+require|does\s+not\s+require|must\s+not\s+require|defer|belongs?\s+to|only.{0,40}(?:5A-D|5B)|\u4E0D\u5F97\u8981\u6C42|\u4E0D\u8981\u6C42|\u7559\u7ED9|\u5C5E\u4E8E|\u53EA\u80FD\u7531|\u540E\u79FB|\u5E76\u975E|\u4E0D\u5F97\u628A|\u7531.{0,40}(?:M1A|5A-D|5B).{0,20}(?:\u63D0\u4F9B|\u4EA7\u751F|\u5B8C\u6210))'
    foreach ($line in [regex]::Split($Content, '\r?\n')) {
        if ($line -notmatch '(?i)(^- \[[ xX]\] P0-\d+\.|P0.{0,40}(?:must|required?|complete|gate|\u5FC5\u987B|\u8981\u6C42|\u5B8C\u6210\u524D|\u963B\u585E))') {
            continue
        }
        if ($line -match $liveEvidence -and $line -match $obligation -and $line -notmatch $deferral) {
            $Failures.Add("PLN-0005 P0 deployment evidence clause must stop at contract fixtures and defer live runs: $($line.Trim())")
        }
    }
}

$failures = New-Object System.Collections.Generic.List[string]
$markdownFiles = Get-ChildItem $docsRoot -Recurse -File -Filter '*.md'
$planIds = @{}
$planStems = @{}
$auditIds = @{}
$auditRecords = @{}
$remediationIds = @{}
$remediationRecords = @{}

$auditsRoot = Join-Path $docsRoot 'audits'
if (Test-Path -LiteralPath $auditsRoot) {
    foreach ($auditFile in Get-ChildItem $auditsRoot -Recurse -File) {
        if ($auditFile.Extension -ne '.md') {
            $failures.Add("Non-Markdown file is not allowed under audits/: $(Get-RepoRelativePath $auditFile.FullName)")
        }
    }
}

$remediationsRoot = Join-Path $docsRoot 'remediations'
if (Test-Path -LiteralPath $remediationsRoot) {
    foreach ($remediationFile in Get-ChildItem $remediationsRoot -Recurse -File) {
        if ($remediationFile.Extension -ne '.md') {
            $failures.Add("Non-Markdown file is not allowed under remediations/: $(Get-RepoRelativePath $remediationFile.FullName)")
        }
    }
}

foreach ($file in $markdownFiles) {
    $relativePath = Get-RepoRelativePath $file.FullName
    $docsRelativePath = Get-DocsRelativePath $file.FullName
    $content = Get-Content $file.FullName -Raw -Encoding UTF8
    $frontmatter = $null

    if ($frontmatterExceptions -notcontains $docsRelativePath) {
        if ($content -notmatch '(?s)\A(?:\uFEFF)?---\r?\n(?<frontmatter>.*?)\r?\n---\r?\n') {
            $failures.Add("Missing frontmatter: $relativePath")
        } else {
            $frontmatter = $Matches['frontmatter']
            $requiredFields = @('status', 'owner', 'last_updated', 'applies_to')
            if ($docsRelativePath.StartsWith('decisions/')) {
                $requiredFields = @('status', 'date')
            }
            if ($docsRelativePath -match '^plans/(?!templates/)(?:archived/)?PLN-') {
                $requiredFields = @('status', 'plan_id', 'owner', 'created', 'last_updated', 'applies_to')
            }
            if ($docsRelativePath.StartsWith('audits/records/')) {
                $requiredFields = @('status', 'audit_id', 'auditor', 'audit_type', 'scope', 'subject', 'baseline', 'started_at', 'last_updated')
            }
            if ($docsRelativePath.StartsWith('remediations/records/')) {
                $requiredFields = @('status', 'remediation_id', 'implementer', 'scope', 'source_audits', 'source_findings', 'baseline', 'started_at', 'last_updated')
            }
            foreach ($field in $requiredFields) {
                if ($frontmatter -notmatch "(?m)^${field}:\s*\S.*$") {
                    $failures.Add("Missing frontmatter field '$field': $relativePath")
                }
            }
        }
    }

    if ($docsRelativePath -match '^plans/(?<archived>archived/)?(?<stem>PLN-(?<number>\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*?)(?<checklist>-checklist)?\.md$') {
        if ($null -eq $frontmatter) {
            continue
        }
        $planId = "PLN-$($Matches['number'])"
        $declaredPlanId = Get-FrontmatterValue $frontmatter 'plan_id'
        if ($declaredPlanId -ne $planId) {
            $failures.Add("Plan ID does not match filename: $relativePath ($declaredPlanId != $planId)")
        }
        $expectedStatus = if ($Matches['archived']) { 'archived' } else { 'active' }
        $declaredStatus = Get-FrontmatterValue $frontmatter 'status'
        if ($declaredStatus -ne $expectedStatus) {
            $failures.Add("Plan status does not match directory: $relativePath ($declaredStatus != $expectedStatus)")
        }
        $stem = $Matches['stem']
        if ($planStems.ContainsKey($planId) -and $planStems[$planId] -ne $stem) {
            $failures.Add("Plan ID is reused by multiple subjects: $planId")
        } else {
            $planStems[$planId] = $stem
        }
        $key = "$planId|$stem"
        if (-not $planIds.ContainsKey($key)) {
            $planIds[$key] = @{ Plan = 0; Checklist = 0 }
        }
        if ($Matches['checklist']) {
            $planIds[$key].Checklist++
        } else {
            $planIds[$key].Plan++
        }
    } elseif ($docsRelativePath.StartsWith('plans/') -and
        $docsRelativePath -notin @('plans/README.md', 'plans/archived/README.md') -and
        -not $docsRelativePath.StartsWith('plans/templates/')) {
        $failures.Add("Invalid plan filename or location: $relativePath")
    }

    if ($docsRelativePath.StartsWith('audits/records/')) {
        if ($docsRelativePath -notmatch '^audits/records/(?<auditId>AUD-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-(?<scopeKind>repository|plan|feature|control|follow-up)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid audit filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $auditId = $Matches['auditId']
            $auditDate = $Matches['date']
            $scopeKind = $Matches['scopeKind']
            $declaredAuditId = Get-FrontmatterValue $frontmatter 'audit_id'
            if ($declaredAuditId -ne $auditId) {
                $failures.Add("Audit ID does not match filename: $relativePath ($declaredAuditId != $auditId)")
            }
            if ($auditIds.ContainsKey($auditId)) {
                $failures.Add("Duplicate audit ID: $auditId")
            } else {
                $auditIds[$auditId] = $relativePath
            }
            $auditStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($auditStatus -notin @('open', 'closed')) {
                $failures.Add("Invalid audit status: $relativePath ($auditStatus)")
            }
            if ($auditStatus -eq 'closed') {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Closed audit must record completed_at: $relativePath")
                }
            }
            $startedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $expectedDate = "$($auditDate.Substring(0, 4))-$($auditDate.Substring(4, 2))-$($auditDate.Substring(6, 2))"
            if ($startedAt -notlike "$expectedDate*") {
                $failures.Add("Audit filename date does not match started_at: $relativePath")
            }
            $declaredScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($scopeKind -ne 'follow-up' -and $declaredScope -notlike "${scopeKind}:*") {
                $failures.Add("Audit scope does not match filename scope kind: $relativePath")
            }

            if ($scopeKind -eq 'plan') {
                $auditSchema = Get-FrontmatterValue $frontmatter 'audit_schema'
                $legacyPlanAudit = $auditId -in @('AUD-0002', 'AUD-0003')
                if ($legacyPlanAudit) {
                    if ($auditSchema -eq 'plan-audit/v2') {
                        $failures.Add("Legacy plan audit must not claim v2 checklist evidence: $relativePath")
                    }
                } else {
                    if ($auditSchema -ne 'plan-audit/v2') {
                        $failures.Add("Plan audit must use audit_schema plan-audit/v2: $relativePath")
                    }
                    $relatedPlans = Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans')
                    if ($relatedPlans.Count -eq 0) {
                        $failures.Add("Plan audit must list related_plans: $relativePath")
                    }
                    foreach ($relatedPlan in $relatedPlans) {
                        if ($relatedPlan -notmatch '^PLN-\d{4}$') {
                            $failures.Add("Invalid related plan ID in plan audit: $relativePath ($relatedPlan)")
                            continue
                        }
                        $matrixMarker = "<!-- plan-checklist-audit: $relatedPlan -->"
                        $matrixMatches = [regex]::Matches($content, [regex]::Escape($matrixMarker))
                        if ($matrixMatches.Count -ne 1) {
                            $failures.Add("Plan audit must contain exactly one checklist matrix for ${relatedPlan}: $relativePath")
                            continue
                        }
                        $markerIndex = $matrixMatches[0].Index
                        $nextMarker = $content.IndexOf('<!-- plan-checklist-audit:', $markerIndex + $matrixMarker.Length, [StringComparison]::Ordinal)
                        $matrixEnd = if ($nextMarker -ge 0) { $nextMarker } else { $content.Length }
                        $matrixContent = $content.Substring($markerIndex, $matrixEnd - $markerIndex)

                        $planLinkPattern = '\]\((?:\.\./)+plans/(?:archived/)?' + [regex]::Escape($relatedPlan) + '-[a-z0-9]+(?:-[a-z0-9]+)*\.md\)'
                        $checklistLinkPattern = '\]\((?:\.\./)+plans/(?:archived/)?' + [regex]::Escape($relatedPlan) + '-[a-z0-9]+(?:-[a-z0-9]+)*-checklist\.md\)'
                        if ($matrixContent -notmatch $planLinkPattern) {
                            $failures.Add("Checklist matrix must link the plan file for ${relatedPlan}: $relativePath")
                        }
                        if ($matrixContent -notmatch $checklistLinkPattern) {
                            $failures.Add("Checklist matrix must link the checklist file for ${relatedPlan}: $relativePath")
                        }

                        foreach ($control in @('PAIRING', 'PLAN_TO_CHECKLIST', 'CHECKLIST_TO_PLAN', 'CHECKED_EVIDENCE', 'GATE_COMPLETENESS', 'ARCHIVE_CLOSURE')) {
                            $controlRows = [regex]::Matches($matrixContent, "(?m)^\|\s*$control\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail|not-applicable)\s*\|(?<finding>[^|]*)\|\s*$")
                            if ($controlRows.Count -ne 1) {
                                $failures.Add("Checklist matrix must contain one valid $control row for ${relatedPlan}: $relativePath")
                                continue
                            }
                            $evidence = $controlRows[0].Groups['evidence'].Value.Trim()
                            $verdict = $controlRows[0].Groups['verdict'].Value
                            $finding = $controlRows[0].Groups['finding'].Value.Trim()
                            if ([string]::IsNullOrWhiteSpace($evidence) -or $evidence -match '^(\.\.\.|TODO|TBD)$') {
                                $failures.Add("Checklist matrix evidence is empty for $control/${relatedPlan}: $relativePath")
                            }
                            if ($control -ne 'CHECKED_EVIDENCE' -and $verdict -eq 'not-applicable') {
                                $failures.Add("Only CHECKED_EVIDENCE may be not-applicable: $control/${relatedPlan} in $relativePath")
                            }
                            if ($verdict -eq 'fail') {
                                if ($finding -notmatch "^$([regex]::Escape($auditId))-F\d{3}$" -or
                                    $content -notmatch "(?m)^###\s+$([regex]::Escape($finding))\s+-") {
                                    $failures.Add("Failed checklist control must reference an existing finding: $control/${relatedPlan} in $relativePath")
                                }
                            } elseif ($finding -ne 'none') {
                                $failures.Add("Passing checklist control must use finding=none: $control/${relatedPlan} in $relativePath")
                            }
                        }
                    }
                }
            }
            $auditRecords[$file.Name] = $auditStatus
        }
    }

    if ($docsRelativePath.StartsWith('remediations/records/')) {
        if ($docsRelativePath -notmatch '^remediations/records/(?<remediationId>REM-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-(?<scopeKind>audit|repository|plan|feature)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid remediation filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $remediationId = $Matches['remediationId']
            $remediationDate = $Matches['date']
            $remediationScopeKind = $Matches['scopeKind']
            $declaredRemediationId = Get-FrontmatterValue $frontmatter 'remediation_id'
            if ($declaredRemediationId -ne $remediationId) {
                $failures.Add("Remediation ID does not match filename: $relativePath ($declaredRemediationId != $remediationId)")
            }
            if ($remediationIds.ContainsKey($remediationId)) {
                $failures.Add("Duplicate remediation ID: $remediationId")
            } else {
                $remediationIds[$remediationId] = $relativePath
            }
            $remediationStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($remediationStatus -notin @('in-progress', 'completed', 'partial', 'blocked')) {
                $failures.Add("Invalid remediation status: $relativePath ($remediationStatus)")
            }
            if ($remediationStatus -ne 'in-progress') {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Closed remediation must record completed_at: $relativePath")
                }
            }
            $remediationStartedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $expectedRemediationDate = "$($remediationDate.Substring(0, 4))-$($remediationDate.Substring(4, 2))-$($remediationDate.Substring(6, 2))"
            if ($remediationStartedAt -notlike "$expectedRemediationDate*") {
                $failures.Add("Remediation filename date does not match started_at: $relativePath")
            }
            $declaredRemediationScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($declaredRemediationScope -notlike "${remediationScopeKind}:*") {
                $failures.Add("Remediation scope does not match filename scope kind: $relativePath")
            }
            $remediationRecords[$file.Name] = $remediationStatus
        }
    }

    foreach ($match in [regex]::Matches($content, '\]\((?<target>[^)]+)\)')) {
        $target = $match.Groups['target'].Value.Trim()
        if ($target.StartsWith('<') -and $target.EndsWith('>')) {
            $target = $target.Substring(1, $target.Length - 2)
        }
        if ($target -match '^[A-Za-z][A-Za-z0-9+.-]*:' -or $target.StartsWith('#')) {
            continue
        }
        $targetPath = $target.Split('#')[0].Split('?')[0]
        if ([string]::IsNullOrWhiteSpace($targetPath)) {
            continue
        }
        $targetPath = [Uri]::UnescapeDataString($targetPath)
        $resolvedTarget = [System.IO.Path]::GetFullPath((Join-Path $file.DirectoryName $targetPath))
        if (-not (Test-Path -LiteralPath $resolvedTarget)) {
            $failures.Add("Missing relative link target: $relativePath -> $target")
        }
    }
}

foreach ($entry in $planIds.GetEnumerator()) {
    if ($entry.Value.Plan -ne 1 -or $entry.Value.Checklist -ne 1) {
        $failures.Add("Plan/checklist pair is incomplete or duplicated: $($entry.Key)")
    }
}

$phaseFivePlanPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle.md'
$phaseFiveChecklistPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle-checklist.md'
if ((Test-Path -LiteralPath $phaseFivePlanPath) -and (Test-Path -LiteralPath $phaseFiveChecklistPath)) {
    $phaseFivePlan = Get-Content -Raw -Encoding UTF8 $phaseFivePlanPath
    $phaseFiveChecklist = Get-Content -Raw -Encoding UTF8 $phaseFiveChecklistPath
    $phaseFiveDag = Get-PhaseFiveDag $phaseFivePlan $failures
    Test-PhaseFiveDependencyStatements $phaseFivePlan $phaseFiveDag $failures
    Test-PhaseFiveDependencyStatements $phaseFiveChecklist $phaseFiveDag $failures

    $deploymentContract = [regex]::Match($phaseFivePlan, '(?s)<!--\s*phase5-p0-deployment-evidence-contract\s*\r?\n(?<body>.*?)\r?\n-->')
    if (-not $deploymentContract.Success -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^p0_artifact_kinds:\s*contract-fixture,disposable-spike\s*$' -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^live_evidence_gates:\s*5A-D-2,5B-4\s*$' -or
        $deploymentContract.Groups['body'].Value -notmatch '(?m)^forbidden_p0_live_evidence:\s*release-binary,supervisor-run,cleanup-schedule-run,watchdog-recovery-run,enospc-run,live-profile-run\s*$') {
        $failures.Add('PLN-0005 must define the structured P0 deployment evidence contract')
    }

    $p0Items = [regex]::Matches($phaseFiveChecklist, '(?m)^- \[[ xX]\] P0-(?<number>\d+)\..*$')
    $p0Counts = @{}
    foreach ($item in $p0Items) {
        $number = [int]$item.Groups['number'].Value
        if (-not $p0Counts.ContainsKey($number)) {
            $p0Counts[$number] = 0
        }
        $p0Counts[$number]++
    }
    foreach ($number in 1..25) {
        if (-not $p0Counts.ContainsKey($number)) {
            $failures.Add("PLN-0005 checklist is missing P0-${number}")
        } elseif ($p0Counts[$number] -ne 1) {
            $failures.Add("PLN-0005 checklist contains duplicate P0-${number}")
        }
    }
    foreach ($number in $p0Counts.Keys) {
        if ($number -lt 1 -or $number -gt 25) {
            $failures.Add("PLN-0005 checklist contains an unexpected P0 item: P0-${number}")
        }
    }

    Test-PhaseFiveDeploymentClauses $phaseFiveChecklist $failures
    Test-PhaseFiveDeploymentClauses $phaseFivePlan $failures

    $p021Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-21\..*$')
    if (-not $p021Line.Success -or $p021Line.Value -notmatch 'WP-Baseline-Evidence.*WP-Facts') {
        $failures.Add('PLN-0005 P0-21 must reject a missing WP-Baseline-Evidence to WP-Facts edge')
    }
    $p022Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-22\..*$')
    if (-not $p022Line.Success -or
        $p022Line.Value -notmatch 'artifactKind=contract-fixture' -or
        $p022Line.Value -notmatch '5A-D-2' -or
        $p022Line.Value -notmatch '5B-4') {
        $failures.Add('PLN-0005 P0-22 must stop at a deployment contract fixture and defer live profile evidence')
    }
    $p023Line = [regex]::Match($phaseFiveChecklist, '(?m)^- \[ \] P0-23\..*$')
    if (-not $p023Line.Success -or
        $p023Line.Value -notmatch 'P0-22.*artifactKind=contract-fixture' -or
        $p023Line.Value -notmatch '5A-D' -or
        $p023Line.Value -notmatch '5B') {
        $failures.Add('PLN-0005 P0-23 must keep P0-22 evidence as a contract fixture')
    }
}

$auditIndexPath = Join-Path $docsRoot 'audits\README.md'
if ($auditRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $auditIndexPath)) {
        $failures.Add('Audit records exist but audits/README.md is missing')
    } else {
        $auditIndexContent = Get-Content -Raw -Encoding UTF8 $auditIndexPath
        foreach ($entry in $auditRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $auditIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Audit record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'remediation=(pending|required|none|accepted-risk|awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4})') {
                $failures.Add("Audit index status is missing or inconsistent: $($entry.Key)")
            }
        }
    }
}

$remediationIndexPath = Join-Path $docsRoot 'remediations\README.md'
if ($remediationRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $remediationIndexPath)) {
        $failures.Add('Remediation records exist but remediations/README.md is missing')
    } else {
        $remediationIndexContent = Get-Content -Raw -Encoding UTF8 $remediationIndexPath
        foreach ($entry in $remediationRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $remediationIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Remediation record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'verification=(not-ready|pending|verified-by:AUD-\d{4}|partial-by:AUD-\d{4}|failed-by:AUD-\d{4})') {
                $failures.Add("Remediation index status is missing or inconsistent: $($entry.Key)")
            }
        }
    }
}

$repositoryDocsRoot = (Join-Path $repoRoot 'docs')
if ($docsRoot -eq $repositoryDocsRoot) {
    $requiredAuditEntrypoints = @(
        '.github/prompts/backend-full-audit.prompt.md',
        '.github/prompts/backend-plan-audit.prompt.md',
        '.github/prompts/backend-fix-audit-findings.prompt.md',
        '.github/prompts/backend-follow-up-audit.prompt.md',
        '.agents/skills/backend-full-audit/SKILL.md',
        '.agents/skills/backend-full-audit/agents/openai.yaml',
        '.agents/skills/backend-plan-audit/SKILL.md',
        '.agents/skills/backend-plan-audit/agents/openai.yaml',
        '.agents/skills/backend-fix-audit-findings/SKILL.md',
        '.agents/skills/backend-fix-audit-findings/agents/openai.yaml',
        '.agents/skills/backend-follow-up-audit/SKILL.md',
        '.agents/skills/backend-follow-up-audit/agents/openai.yaml',
        '.agents/skills/backend-audit-until-clean/SKILL.md',
        '.agents/skills/backend-audit-until-clean/agents/openai.yaml'
    )
    foreach ($entrypoint in $requiredAuditEntrypoints) {
        if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $entrypoint))) {
            $failures.Add("Missing audit command entrypoint: $entrypoint")
        }
    }

    $legacyFullAuditPrompt = Join-Path $repoRoot '.github/prompts/backend-full-audit-cycle.prompt.md'
    if (Test-Path -LiteralPath $legacyFullAuditPrompt) {
        $failures.Add('Legacy full-audit prompt must not coexist with backend-full-audit.prompt.md')
    }

    $fullAuditPromptPath = Join-Path $repoRoot '.github/prompts/backend-full-audit.prompt.md'
    if (Test-Path -LiteralPath $fullAuditPromptPath) {
        $fullAuditPrompt = Get-Content -Raw -Encoding UTF8 $fullAuditPromptPath
        if ($fullAuditPrompt -notmatch '(?m)^name:\s*backend-full-audit\s*$' -or
            $fullAuditPrompt -notmatch 'scope:\s*repository:allinme\.core-api' -or
            $fullAuditPrompt -notmatch 'audit-contract:\s*full-repository;\s*scope-may-not-narrow') {
            $failures.Add('Full-audit prompt must enforce complete repository scope')
        }
    }

    $planAuditPromptPath = Join-Path $repoRoot '.github/prompts/backend-plan-audit.prompt.md'
    if (Test-Path -LiteralPath $planAuditPromptPath) {
        $planAuditPrompt = Get-Content -Raw -Encoding UTF8 $planAuditPromptPath
        if ($planAuditPrompt -notmatch '(?m)^name:\s*backend-plan-audit\s*$' -or
            $planAuditPrompt -notmatch 'TARGET' -or
            $planAuditPrompt -notmatch 'audit-contract:\s*plan;\s*default-target=active;\s*explicit-targets=true;\s*checklist-matrix-required' -or
            $planAuditPrompt -notmatch 'audit_schema:\s*plan-audit/v2') {
            $failures.Add('Plan-audit prompt must default to active plans and support explicit targets')
        }
    }

    $fixAuditPromptPath = Join-Path $repoRoot '.github/prompts/backend-fix-audit-findings.prompt.md'
    if (Test-Path -LiteralPath $fixAuditPromptPath) {
        $fixAuditPrompt = Get-Content -Raw -Encoding UTF8 $fixAuditPromptPath
        if ($fixAuditPrompt -notmatch '(?m)^name:\s*backend-fix-audit-findings\s*$' -or
            $fixAuditPrompt -notmatch 'remediation-contract:\s*default-target=required-audits;\s*creates-rem-record') {
            $failures.Add('Audit remediation prompt must target required audits and create REM records')
        }
    }

    $followUpPromptPath = Join-Path $repoRoot '.github/prompts/backend-follow-up-audit.prompt.md'
    if (Test-Path -LiteralPath $followUpPromptPath) {
        $followUpPrompt = Get-Content -Raw -Encoding UTF8 $followUpPromptPath
        if ($followUpPrompt -notmatch '(?m)^name:\s*backend-follow-up-audit\s*$' -or
            $followUpPrompt -notmatch 'follow-up-contract:\s*default-target=pending-remediations;\s*creates-new-audit') {
            $failures.Add('Follow-up prompt must target pending REM records and create a new AUD')
        }
    }

    foreach ($skillName in @('backend-full-audit', 'backend-plan-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit')) {
        $skillRoot = Join-Path $repoRoot ".agents/skills/$skillName"
        $skillPath = Join-Path $skillRoot 'SKILL.md'
        $metadataPath = Join-Path $skillRoot 'agents/openai.yaml'
        if (Test-Path -LiteralPath $skillPath) {
            $skillContent = Get-Content -Raw -Encoding UTF8 $skillPath
            if ($skillContent -notmatch "(?m)^name:\s*$skillName\s*$" -or
                $skillContent -notmatch "\.github/prompts/$skillName\.prompt\.md") {
                $failures.Add("Codex skill does not bind to its canonical prompt: $skillName")
            }
        }
        if (Test-Path -LiteralPath $metadataPath) {
            $metadataContent = Get-Content -Raw -Encoding UTF8 $metadataPath
            if ($metadataContent -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') {
                $failures.Add("Audit skill must require explicit invocation: $skillName")
            }
        }
    }

    $orchestratorSkillPath = Join-Path $repoRoot '.agents/skills/backend-audit-until-clean/SKILL.md'
    $orchestratorMetadataPath = Join-Path $repoRoot '.agents/skills/backend-audit-until-clean/agents/openai.yaml'
    if (Test-Path -LiteralPath $orchestratorSkillPath) {
        $orchestratorContent = Get-Content -Raw -Encoding UTF8 $orchestratorSkillPath
        foreach ($dependencySkill in @('backend-plan-audit', 'backend-full-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit')) {
            if ($orchestratorContent -notmatch ('\$' + [regex]::Escape($dependencySkill))) {
                $failures.Add("Audit orchestrator must invoke existing skill: $dependencySkill")
            }
        }
        if ($orchestratorContent -notmatch 'MAX_CYCLES' -or
            $orchestratorContent -notmatch 'MAX_STAGNANT_CYCLES' -or
            $orchestratorContent -notmatch 'persistent goal') {
            $failures.Add('Audit orchestrator must establish a goal and enforce bounded loop circuit breakers')
        }
    }
    if (Test-Path -LiteralPath $orchestratorMetadataPath) {
        $orchestratorMetadata = Get-Content -Raw -Encoding UTF8 $orchestratorMetadataPath
        if ($orchestratorMetadata -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') {
            $failures.Add('Audit orchestrator must require explicit invocation')
        }
    }

    $trackedAuditPaths = & git -C $repoRoot ls-tree -r --name-only HEAD -- docs/audits/records 2>$null
    foreach ($trackedAuditPath in $trackedAuditPaths) {
        $currentAuditPath = Join-Path $repoRoot $trackedAuditPath
        if (-not (Test-Path -LiteralPath $currentAuditPath)) {
            $failures.Add("Tracked audit record must not be deleted or moved: $trackedAuditPath")
            continue
        }
        $headAuditContent = (& git -C $repoRoot show "HEAD:$trackedAuditPath" 2>$null | Out-String)
        if ($headAuditContent -match '(?m)^status:\s*closed\s*$') {
            & git -C $repoRoot diff --quiet HEAD -- $trackedAuditPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Closed audit record is immutable; create a new related audit instead: $trackedAuditPath")
            }
        }
    }

    $trackedRemediationPaths = & git -C $repoRoot ls-tree -r --name-only HEAD -- docs/remediations/records 2>$null
    foreach ($trackedRemediationPath in $trackedRemediationPaths) {
        $currentRemediationPath = Join-Path $repoRoot $trackedRemediationPath
        if (-not (Test-Path -LiteralPath $currentRemediationPath)) {
            $failures.Add("Tracked remediation record must not be deleted or moved: $trackedRemediationPath")
            continue
        }
        $headRemediationContent = (& git -C $repoRoot show "HEAD:$trackedRemediationPath" 2>$null | Out-String)
        if ($headRemediationContent -match '(?m)^status:\s*(completed|partial|blocked)\s*$') {
            & git -C $repoRoot diff --quiet HEAD -- $trackedRemediationPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Closed remediation record is immutable; create a new remediation instead: $trackedRemediationPath")
            }
        }
    }
}

$previousErrorAction = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
$diffOutput = & git -C $repoRoot diff HEAD --check 2>$null
$diffExitCode = $LASTEXITCODE
$ErrorActionPreference = $previousErrorAction
if ($diffExitCode -ne 0) {
    foreach ($line in $diffOutput) {
        $failures.Add("git diff HEAD --check: $line")
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { [Console]::Error.WriteLine($_) }
    exit 1
}

Write-Output "Validated $($markdownFiles.Count) Markdown files: frontmatter, relative links, and git diff HEAD --check passed."
