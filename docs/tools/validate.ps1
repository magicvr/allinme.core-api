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
    'audits/templates/implementation-acceptance-audit-record.md',
    'audits/templates/implementation-audit-record.md',
    'audits/templates/plan-acceptance-audit-record.md',
    'audits/templates/plan-audit-record.md',
    'decisions/README.md',
    'evidence/README.md',
    'implementations/README.md',
    'implementations/templates/implementation-record.md',
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

function Get-PhaseFiveStatementClauses([string]$Text) {
    $separator = '(?<=[.;\u3002\uFF1B!?\uFF01\uFF1F])\s*|[,\uFF0C]?\s*(?=(?i:(?:but|however|yet)\b|\u4F46\u662F|\u7136\u800C|\u4E0D\u8FC7|\u4F46))'
    return @([regex]::Split($Text, $separator) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Get-PhaseFiveDependencyClauses([string]$Text) {
    $clauses = New-Object System.Collections.Generic.List[string]
    foreach ($sentence in [regex]::Split($Text, '(?<=[.;\u3002\uFF1B!?\uFF01\uFF1F])\s*')) {
        if ([string]::IsNullOrWhiteSpace($sentence)) {
            continue
        }
        if ($sentence -match '(?i)(reject|must\s+reject|\u62D2\u7EDD)') {
            foreach ($clause in Get-PhaseFiveStatementClauses $sentence) {
                $clauses.Add($clause)
            }
        } else {
            $clauses.Add($sentence)
        }
    }
    return @($clauses)
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

function Test-PhaseFiveFactsOutput([string]$Content, [System.Collections.Generic.List[string]]$Failures) {
    $row = [regex]::Match($Content, '(?m)^\|\s*WP-Facts\s*\|(?<items>[^|]*)\|(?<owner>[^|]*)\|(?<inputs>[^|]*)\|(?<timebox>[^|]*)\|(?<evidence>[^|]*)\|\s*$')
    if (-not $row.Success) {
        $Failures.Add('PLN-0005 tracked work-package table is missing the WP-Facts output contract')
        return
    }

    $requiredOutputs = @(
        'docs/01-architecture.md',
        'docs/05-domain-model.md',
        'docs/03-http-api-target.md',
        'docs/06-implementation-roadmap.md',
        'docs/04-validation.md',
        'plan/checklist'
    )
    $evidence = $row.Groups['evidence'].Value
    $outputTokens = @(
        [regex]::Split($evidence, '[,\uFF0C\u3001;\uFF1B\s]+') |
            ForEach-Object { $_.Trim('`', '.', ':', ',', ';') } |
            Where-Object { $_ -match '^(?:docs|plan)[/\\]' }
    )
    foreach ($outputToken in $outputTokens) {
        if ($outputToken -notin $requiredOutputs) {
            $Failures.Add("PLN-0005 WP-Facts output contains unknown or non-canonical token: $outputToken")
        }
    }
    foreach ($duplicate in @($outputTokens | Group-Object | Where-Object { $_.Count -gt 1 })) {
        $Failures.Add("PLN-0005 WP-Facts output contains duplicate token: $($duplicate.Name)")
    }
    foreach ($requiredOutput in $requiredOutputs) {
        if ($outputTokens -notcontains $requiredOutput) {
            $Failures.Add("PLN-0005 WP-Facts output is missing required fact source: $requiredOutput")
        }
    }
}

function Test-PhaseFiveDependencyStatements([string]$Content, [hashtable]$Dag, [System.Collections.Generic.List[string]]$Failures) {
    $contentWithoutDagRows = [regex]::Replace($Content, '(?m)^\|\s*WP-[A-Za-z0-9-]+\s*\|.*$', '')
    foreach ($line in [regex]::Split($contentWithoutDagRows, '\r?\n')) {
        foreach ($clause in Get-PhaseFiveDependencyClauses $line) {
            if ($clause -notmatch 'WP-[A-Za-z0-9-]+') {
                continue
            }
            $isRejectionClause = $clause -match '(?i)(reject|must\s+reject|\u62D2\u7EDD)'
            if ($isRejectionClause) {
                continue
            }

            $consumedPackages = New-Object System.Collections.Generic.HashSet[string]
            $relationshipFound = $false
            foreach ($match in [regex]::Matches($clause, '(?<prerequisite>WP-[A-Za-z0-9-]+)\s*(?:->|\u2192)\s*(?<dependent>WP-[A-Za-z0-9-]+)')) {
                $relationshipFound = $true
                [void]$consumedPackages.Add($match.Groups['prerequisite'].Value)
                [void]$consumedPackages.Add($match.Groups['dependent'].Value)
                if (-not (Test-PhaseFiveEdge $Dag $match.Groups['prerequisite'].Value $match.Groups['dependent'].Value)) {
                    $Failures.Add("PLN-0005 dependency statement is not present in the tracked DAG: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)(?<middle>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?)(?<prerequisite>WP-[A-Za-z0-9-]+)(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?)(?:does\s+not|doesn''t|need\s+not)\s+depend\s+(?:on|upon)\s+it\b')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                $prerequisite = $match.Groups['prerequisite'].Value
                [void]$consumedPackages.Add($dependent)
                [void]$consumedPackages.Add($prerequisite)
                if (Test-PhaseFiveEdge $Dag $prerequisite $dependent) {
                    $Failures.Add("PLN-0005 dependency statement denies a tracked edge: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?<negation>does\s+not|doesn''t|need\s+not|without|\u4E0D\u4F9D\u8D56|\u65E0\u9700\u7B49\u5F85)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:depend(?:ing)?\s+(?:on|upon)\s+)?(?<prerequisite>WP-[A-Za-z0-9-]+)')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                $prerequisite = $match.Groups['prerequisite'].Value
                [void]$consumedPackages.Add($dependent)
                [void]$consumedPackages.Add($prerequisite)
                if (Test-PhaseFiveEdge $Dag $prerequisite $dependent) {
                    $Failures.Add("PLN-0005 dependency statement denies a tracked edge: $($match.Value)")
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:depends?\s+(?:on|upon)|(?<!\u5148\u4E8E)\u4F9D\u8D56)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                if ($match.Value -match '(?i)(?:does\s+not|doesn''t|need\s+not)\s+depend|\u4E0D\u4F9D\u8D56|\u65E0\u9700\u7B49\u5F85') {
                    continue
                }
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                [void]$consumedPackages.Add($dependent)
                $prerequisites = @([regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value })
                if ($prerequisites.Count -eq 0) {
                    $Failures.Add("PLN-0005 dependency statement could not be fully parsed: $($match.Value)")
                }
                foreach ($prerequisite in $prerequisites) {
                    [void]$consumedPackages.Add($prerequisite)
                    if (-not (Test-PhaseFiveEdge $Dag $prerequisite $dependent)) {
                        $Failures.Add("PLN-0005 dependency statement contradicts the tracked DAG: $dependent depends on $prerequisite")
                    }
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<first>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:before|precedes?|\u5148\u4E8E|\u65E9\u4E8E)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                $relationshipFound = $true
                $first = $match.Groups['first'].Value
                [void]$consumedPackages.Add($first)
                foreach ($secondMatch in [regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+')) {
                    $second = $secondMatch.Value
                    [void]$consumedPackages.Add($second)
                    if (-not (Test-PhaseFiveEdge $Dag $first $second)) {
                        $Failures.Add("PLN-0005 dependency ordering contradicts or is absent from the tracked DAG: $first before $second")
                    }
                }
            }

            foreach ($match in [regex]::Matches($clause, '(?i)(?<dependent>WP-[A-Za-z0-9-]+)[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*?(?:runs?\s+after|after|follows?|\u665A\u4E8E|\u4E4B\u540E)\s*(?<tail>[^.;\u3002\uFF1B!?\uFF01\uFF1F\r\n]*)')) {
                $relationshipFound = $true
                $dependent = $match.Groups['dependent'].Value
                [void]$consumedPackages.Add($dependent)
                foreach ($prerequisiteMatch in [regex]::Matches($match.Groups['tail'].Value, 'WP-[A-Za-z0-9-]+')) {
                    $prerequisite = $prerequisiteMatch.Value
                    [void]$consumedPackages.Add($prerequisite)
                    if (-not (Test-PhaseFiveEdge $Dag $prerequisite $dependent)) {
                        $Failures.Add("PLN-0005 dependency ordering contradicts or is absent from the tracked DAG: $dependent after $prerequisite")
                    }
                }
            }

            $allPackages = @([regex]::Matches($clause, 'WP-[A-Za-z0-9-]+') | ForEach-Object { $_.Value } | Select-Object -Unique)
            if ($allPackages.Count -gt 1) {
                if (-not $relationshipFound) {
                    $Failures.Add("PLN-0005 dependency statement could not be fully parsed: $($clause.Trim())")
                } else {
                    foreach ($package in $allPackages) {
                        if (-not $consumedPackages.Contains($package)) {
                            $Failures.Add("PLN-0005 dependency statement contains an unconsumed work package: $package in $($clause.Trim())")
                        }
                    }
                }
            }
        }
    }
}

function Get-PhaseFiveLiveEvidencePattern([string[]]$Categories, [System.Collections.Generic.List[string]]$Failures) {
    $patterns = @{
        'release-binary' = '(?:release[-\s]+binary|live\s+binary|real\s+binary|\u9636\u6BB5\u4E94\s*binary|\u771F\u5B9E(?:\u9636\u6BB5\u4E94\s*)?binary)'
        'supervisor-run' = '(?:live\s+supervisor|real\s+supervisor|supervisor.{0,24}(?:run|evidence)|\u771F\u5B9E(?:\u9636\u6BB5\u4E94\s*)?\u76D1\u7763\u5668|\u76D1\u7763\u5668.{0,20}(?:run|Evidence|\u8BC1\u636E))'
        'cleanup-schedule-run' = '(?:cleanup\s+schedule|cleanup\s*\u8C03\u5EA6|\u6E05\u7406\u8C03\u5EA6)'
        'watchdog-recovery-run' = '(?:watchdog(?:/|\s+and\s+|\s*)recovery|watchdog.{0,24}(?:run|evidence))'
        'enospc-run' = '(?:ENOSPC.{0,40}(?:run|evidence|\u8BC1\u636E|\u6F14\u7EC3))'
        'live-profile-run' = '(?:live\s+(?:deployment\s+)?profile|real\s+(?:deployment\s+)?profile|\u90E8\u7F72\s*profile\s*run|\u5B9E\u6D4B.{0,20}(?:\u90E8\u7F72|profile))'
    }
    $selected = New-Object System.Collections.Generic.List[string]
    foreach ($category in $Categories) {
        if (-not $patterns.ContainsKey($category)) {
            $Failures.Add("PLN-0005 deployment evidence contract contains an unknown forbidden category: $category")
            continue
        }
        $selected.Add($patterns[$category])
    }
    if ($selected.Count -eq 0) {
        return $null
    }
    return '(?i)(' + ($selected -join '|') + ')'
}

function Test-PhaseFiveDeploymentClauses([string]$Content, [string[]]$ForbiddenCategories, [bool]$Checklist, [System.Collections.Generic.List[string]]$Failures) {
    $liveEvidence = Get-PhaseFiveLiveEvidencePattern $ForbiddenCategories $Failures
    if ([string]::IsNullOrWhiteSpace($liveEvidence)) {
        return
    }
    $obligation = '(?i)(must|required?|shall|before\s+P0|P0.{0,20}(?:complete|gate)|\u5FC5\u987B|\u8981\u6C42|\u5B8C\u6210\u524D|\u963B\u585E|\u4E0D\u5F97\u8FDB\u5165)'
    $deferral = '(?i)(not\s+require|does\s+not\s+require|must\s+not\s+require|not.{0,30}P0.{0,20}(?:prerequisite|gate)|defer|belongs?\s+to|only.{0,40}(?:5A-D|5B)|\u4E0D\u5F97\u8981\u6C42|\u4E0D\u8981\u6C42|\u7559\u7ED9|\u5C5E\u4E8E|\u53EA\u80FD\u7531|\u540E\u79FB|\u5E76\u975E|\u4E0D\u662F.{0,30}P0.{0,20}(?:\u524D\u7F6E|\u95E8\u7981)|\u4E0D\u5F97\u628A|\u7531.{0,40}(?:M1A|5A-D|5B).{0,20}(?:\u63D0\u4F9B|\u4EA7\u751F|\u5B8C\u6210))'
    $scopes = if ($Checklist) {
        @([regex]::Matches($Content, '(?ms)^- \[[ xX]\] P0-(?<number>\d+)\.(?<body>.*?)(?=^- \[[ xX]\] (?:P0-\d+|[A-Za-z0-9]+-\d+)\.|^## |\z)') | ForEach-Object {
            @{ Label = "P0-$($_.Groups['number'].Value)"; Text = $_.Groups['body'].Value }
        })
    } else {
        @(@{ Label = 'plan'; Text = $Content })
    }
    foreach ($scope in $scopes) {
        $clauses = Get-PhaseFiveStatementClauses $scope.Text
        foreach ($clause in $clauses) {
            if (-not $Checklist -and $clause -notmatch '(?i)\bP0(?:-\d+)?\b') {
                continue
            }
            $liveMatches = @([regex]::Matches($clause, $liveEvidence))
            if ($liveMatches.Count -eq 0) {
                continue
            }
            $deferralMatches = @([regex]::Matches($clause, $deferral))
            $obligationMatches = New-Object System.Collections.Generic.List[object]
            foreach ($obligationMatch in [regex]::Matches($clause, $obligation)) {
                $overlapsDeferral = $false
                foreach ($deferralMatch in $deferralMatches) {
                    if ($obligationMatch.Index -lt ($deferralMatch.Index + $deferralMatch.Length) -and
                        $deferralMatch.Index -lt ($obligationMatch.Index + $obligationMatch.Length)) {
                        $overlapsDeferral = $true
                        break
                    }
                }
                if (-not $overlapsDeferral) {
                    $obligationMatches.Add($obligationMatch)
                }
            }

            $violation = $false
            foreach ($liveMatch in $liveMatches) {
                $nearestType = $null
                $nearestDistance = [int]::MaxValue
                foreach ($signal in @($deferralMatches | ForEach-Object { @{ Type = 'deferral'; Match = $_ } }) + @($obligationMatches | ForEach-Object { @{ Type = 'obligation'; Match = $_ } })) {
                    $signalMatch = $signal.Match
                    $liveEnd = $liveMatch.Index + $liveMatch.Length
                    $signalEnd = $signalMatch.Index + $signalMatch.Length
                    $distance = if ($signalEnd -le $liveMatch.Index) {
                        $liveMatch.Index - $signalEnd
                    } elseif ($liveEnd -le $signalMatch.Index) {
                        $signalMatch.Index - $liveEnd
                    } else {
                        0
                    }
                    if ($distance -lt $nearestDistance -or
                        ($distance -eq $nearestDistance -and $signal.Type -eq 'obligation')) {
                        $nearestDistance = $distance
                        $nearestType = $signal.Type
                    }
                }
                if ($nearestType -eq 'obligation') {
                    $violation = $true
                    break
                }
            }
            if ($violation) {
                $Failures.Add("PLN-0005 P0 deployment evidence clause must stop at contract fixtures and defer live runs ($($scope.Label)): $($clause.Trim())")
            }
        }
    }
}

function Test-AuditMatrix([string]$Content, [string]$Marker, [string[]]$Controls, [string]$AuditId, [System.Collections.Generic.List[string]]$Failures, [string]$Label) {
    $markers = [regex]::Matches($Content, [regex]::Escape($Marker))
    if ($markers.Count -ne 1) {
        $Failures.Add("$Label must contain exactly one matrix marker: $Marker")
        return
    }
    $separatorIndex = $Marker.LastIndexOf(':')
    $markerPrefix = if ($separatorIndex -ge 0) { $Marker.Substring(0, $separatorIndex + 1) } else { $Marker }
    $matrixContent = Get-MatrixContent $Content $Marker $markerPrefix
    foreach ($control in $Controls) {
        $rows = [regex]::Matches($matrixContent, "(?m)^\|\s*$([regex]::Escape($control))\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail)\s*\|(?<finding>[^|]*)\|\s*$")
        if ($rows.Count -ne 1) {
            $Failures.Add("$Label must contain one valid $control row: $Marker")
            continue
        }
        $evidence = $rows[0].Groups['evidence'].Value.Trim()
        $verdict = $rows[0].Groups['verdict'].Value
        $finding = $rows[0].Groups['finding'].Value.Trim()
        if ([string]::IsNullOrWhiteSpace($evidence) -or $evidence -match '^(...|TODO|TBD)$') {
            $Failures.Add("$Label evidence is empty for ${control}: $Marker")
        }
        if ($verdict -eq 'fail') {
            if ($finding -notmatch "^$([regex]::Escape($AuditId))-F\d{3}$" -or
                $Content -notmatch "(?m)^###\s+$([regex]::Escape($finding))\s+-") {
                $Failures.Add("Failed $Label control must reference an existing finding: $control in $Marker")
            }
        } elseif ($finding -ne 'none') {
            $Failures.Add("Passing $Label control must use finding=none: $control in $Marker")
        }
    }
}

function Get-MatrixContent([string]$Content, [string]$Marker, [string]$MarkerPrefix) {
    $markerIndex = $Content.IndexOf($Marker, [StringComparison]::Ordinal)
    if ($markerIndex -lt 0) {
        return $null
    }
    $nextMarker = $Content.IndexOf($MarkerPrefix, $markerIndex + $Marker.Length, [StringComparison]::Ordinal)
    $matrixEnd = if ($nextMarker -ge 0) { $nextMarker } else { $Content.Length }
    return $Content.Substring($markerIndex, $matrixEnd - $markerIndex)
}

function Test-AcceptanceVerdict([string]$Content, [string]$Marker, [string]$MarkerPrefix, [string[]]$Controls, [string]$Verdict, [string]$Status, [string]$AuditId, [string]$Label, [System.Collections.Generic.List[string]]$Failures) {
    if ($Status -eq 'open' -and $Verdict -ne 'pending') {
        $Failures.Add("Closed acceptance result must use status=closed: $AuditId")
    }
    if ($Status -eq 'closed' -and $Verdict -eq 'pending') {
        $Failures.Add("Closed $Label cannot keep acceptance_verdict=pending: $AuditId")
    }
    if ($Verdict -eq 'pending') {
        return
    }
    $matrixContent = if ([string]::IsNullOrWhiteSpace($Marker)) { $Content } else { Get-MatrixContent $Content $Marker $MarkerPrefix }
    if ($null -eq $matrixContent) { return }
    $failedCount = 0
    foreach ($control in $Controls) {
        $rows = [regex]::Matches($matrixContent, "(?m)^\|\s*$([regex]::Escape($control))\s*\|(?<evidence>[^|]*)\|\s*(?<verdict>pass|fail)\s*\|(?<finding>[^|]*)\|\s*$")
        foreach ($row in $rows) {
            if ($row.Groups['verdict'].Value -eq 'fail') {
                $failedCount++
            }
        }
    }
    if ($Verdict -in @('ready', 'complete') -and $failedCount -gt 0) {
        $Failures.Add("Acceptance verdict $Verdict requires every Control to pass: $AuditId")
    }
    if ($Verdict -in @('not-ready', 'blocked', 'incomplete') -and $failedCount -eq 0) {
        $Failures.Add("Acceptance verdict $Verdict requires at least one failed Control with a finding: $AuditId")
    }
}

$failures = New-Object System.Collections.Generic.List[string]
$markdownFiles = Get-ChildItem $docsRoot -Recurse -File -Filter '*.md'
$planIds = @{}
$planStems = @{}
$planMetadata = @{}
$auditIds = @{}
$auditRecords = @{}
$auditMetadata = @{}
$auditRemediationStates = @{}
$remediationIds = @{}
$remediationRecords = @{}
$remediationMetadata = @{}
$implementationIds = @{}
$implementationRecords = @{}
$implementationMetadata = @{}

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
            if ($docsRelativePath.StartsWith('implementations/records/')) {
                $requiredFields = @('status', 'implementation_id', 'implementer', 'scope', 'related_plans', 'plan_acceptance_audits', 'baseline', 'result_revision', 'started_at', 'last_updated')
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
        $planMetadata[$planId] = @{ Path = $relativePath; Status = $declaredStatus; Stem = $stem }
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
        if ($docsRelativePath -notmatch '^audits/records/(?<auditId>AUD-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-(?<scopeKind>repository|plan|implementation|feature|control|follow-up)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
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

            if ($scopeKind -eq 'implementation') {
                $auditSchema = Get-FrontmatterValue $frontmatter 'audit_schema'
                if ($auditSchema -ne 'implementation-audit/v1') {
                    $failures.Add("Implementation audit must use audit_schema implementation-audit/v1: $relativePath")
                }
                $relatedImplementations = Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations')
                if ($relatedImplementations.Count -eq 0) {
                    $failures.Add("Implementation audit must list related_implementations: $relativePath")
                }
                foreach ($relatedImplementation in $relatedImplementations) {
                    if ($relatedImplementation -notmatch '^IMP-\d{4}$') {
                        $failures.Add("Invalid related implementation ID in implementation audit: $relativePath ($relatedImplementation)")
                        continue
                    }
                    Test-AuditMatrix $content "<!-- implementation-audit: $relatedImplementation -->" @('IMP_TRACEABILITY', 'CHECKLIST_EVIDENCE', 'CODE_CONTRACT', 'TEST_FAILURE', 'SECURITY_DATA', 'MIGRATION_RECOVERY', 'DOCS_CI_RELEASE') $auditId $failures 'Implementation audit matrix'
                }
            } elseif ($scopeKind -eq 'plan') {
                $auditSchema = Get-FrontmatterValue $frontmatter 'audit_schema'
                $legacyPlanAudit = $auditId -in @('AUD-0002', 'AUD-0003')
                if ($legacyPlanAudit) {
                    if ($auditSchema -eq 'plan-audit/v2') {
                        $failures.Add("Legacy plan audit must not claim v2 checklist evidence: $relativePath")
                    }
                } else {
                    $relatedPlans = Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans')
                    if ($auditSchema -in @('plan-acceptance/v1', 'implementation-acceptance/v1')) {
                        if ($relatedPlans.Count -eq 0) {
                            $failures.Add("Acceptance audit must list related_plans: $relativePath")
                        }
                        $acceptanceType = Get-FrontmatterValue $frontmatter 'acceptance_type'
                        $expectedAcceptanceType = if ($auditSchema -eq 'plan-acceptance/v1') { 'plan-readiness' } else { 'implementation-completion' }
                        if ($acceptanceType -ne $expectedAcceptanceType) {
                            $failures.Add("Acceptance audit has incorrect acceptance_type: $relativePath")
                        }
                        $acceptanceVerdict = Get-FrontmatterValue $frontmatter 'acceptance_verdict'
                        if ($acceptanceVerdict -notin @('pending', 'ready', 'not-ready', 'blocked', 'complete', 'incomplete')) {
                            $failures.Add("Acceptance audit has invalid acceptance_verdict: $relativePath ($acceptanceVerdict)")
                        }
                        $independenceBasis = Get-FrontmatterValue $frontmatter 'independence_basis'
                        if ($independenceBasis -notin @('separate-auditor', 'fresh-context-independent-rerun')) {
                            $failures.Add("Acceptance audit must record a valid independence_basis: $relativePath")
                        }
                        $acceptanceBaseline = Get-FrontmatterValue $frontmatter 'baseline'
                        $evidenceRevision = Get-FrontmatterValue $frontmatter 'evidence_revision'
                        if ($acceptanceBaseline -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                            $failures.Add("Acceptance audit baseline must be a full git SHA on a clean worktree: $relativePath")
                        }
                        if ($evidenceRevision -notmatch '^git:[0-9a-fA-F]{40};\s*worktree:clean$') {
                            $failures.Add("Acceptance audit evidence_revision must be a full git SHA on a clean worktree: $relativePath")
                        }
                        $markerPrefix = if ($auditSchema -eq 'plan-acceptance/v1') { 'plan-acceptance-audit' } else { 'implementation-acceptance-audit' }
                        $controls = if ($auditSchema -eq 'plan-acceptance/v1') {
                            @('READY_IDENTITY', 'READY_SCOPE', 'READY_FACTS', 'READY_DEPENDENCIES', 'READY_DESIGN', 'READY_EVIDENCE', 'READY_GATES', 'PLAN_AUDIT_CHAIN_CLEAN')
                        } else {
                            @('IMP_PRESENT', 'SCOPE_COMPLETE', 'CHECKLIST_COMPLETE', 'VALIDATION_GATES', 'AUDIT_CHAIN_CLEAN', 'RESIDUAL_RISK', 'ARCHIVE_READY')
                        }
                        foreach ($relatedPlan in $relatedPlans) {
                            if ($relatedPlan -notmatch '^PLN-\d{4}$') {
                                $failures.Add("Invalid related plan ID in acceptance audit: $relativePath ($relatedPlan)")
                                continue
                            }
                            Test-AuditMatrix $content "<!-- $markerPrefix`: $relatedPlan -->" $controls $auditId $failures 'Acceptance audit matrix'
                        }
                        Test-AcceptanceVerdict $content $null $null $controls $acceptanceVerdict $auditStatus $auditId 'acceptance audit' $failures
                        if ($auditSchema -eq 'implementation-acceptance/v1') {
                            $relatedImplementations = Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations')
                            if ($relatedImplementations.Count -eq 0) {
                                $failures.Add("Implementation acceptance audit must list related_implementations: $relativePath")
                            }
                        }
                    } elseif ($auditSchema -ne 'plan-audit/v2') {
                        $failures.Add("Plan audit must use audit_schema plan-audit/v2: $relativePath")
                    } else {
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
            }
            $auditMetadata[$auditId] = @{
                Path = $relativePath
                Status = $auditStatus
                Schema = Get-FrontmatterValue $frontmatter 'audit_schema'
                Verdict = Get-FrontmatterValue $frontmatter 'acceptance_verdict'
                RelatedAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_audits'))
                RelatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
                RelatedImplementations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_implementations'))
                RelatedRemediations = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_remediations'))
                Scope = $declaredScope
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
            $remediationMetadata[$remediationId] = @{
                Path = $relativePath
                Status = $remediationStatus
                SourceAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'source_audits'))
                RelatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
            }
            $remediationRecords[$file.Name] = $remediationStatus
        }
    }

    if ($docsRelativePath.StartsWith('implementations/records/')) {
        if ($docsRelativePath -notmatch '^implementations/records/(?<implementationId>IMP-\d{4})-(?<date>\d{8})-[a-z0-9]+(?:-[a-z0-9]+)*-plan-(?<planId>pln-\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $failures.Add("Invalid implementation filename: $relativePath")
        } elseif ($null -ne $frontmatter) {
            $implementationId = $Matches['implementationId']
            $implementationDate = $Matches['date']
            $filenamePlanId = "PLN-$($Matches['planId'].Substring(4))"
            $declaredImplementationId = Get-FrontmatterValue $frontmatter 'implementation_id'
            if ($declaredImplementationId -ne $implementationId) {
                $failures.Add("Implementation ID does not match filename: $relativePath ($declaredImplementationId != $implementationId)")
            }
            if ($implementationIds.ContainsKey($implementationId)) {
                $failures.Add("Duplicate implementation ID: $implementationId")
            } else {
                $implementationIds[$implementationId] = $relativePath
            }
            $implementationStatus = Get-FrontmatterValue $frontmatter 'status'
            if ($implementationStatus -notin @('in-progress', 'completed', 'partial', 'blocked')) {
                $failures.Add("Invalid implementation status: $relativePath ($implementationStatus)")
            }
            if ($implementationStatus -ne 'in-progress') {
                $completedAt = Get-FrontmatterValue $frontmatter 'completed_at'
                if ([string]::IsNullOrWhiteSpace($completedAt) -or $completedAt -eq 'pending') {
                    $failures.Add("Closed implementation must record completed_at: $relativePath")
                }
            }
            $implementationStartedAt = Get-FrontmatterValue $frontmatter 'started_at'
            $expectedImplementationDate = "$($implementationDate.Substring(0, 4))-$($implementationDate.Substring(4, 2))-$($implementationDate.Substring(6, 2))"
            if ($implementationStartedAt -notlike "$expectedImplementationDate*") {
                $failures.Add("Implementation filename date does not match started_at: $relativePath")
            }
            $declaredImplementationScope = Get-FrontmatterValue $frontmatter 'scope'
            if ($declaredImplementationScope -notlike 'plan:*') {
                $failures.Add("Implementation scope must start with plan:: $relativePath")
            }
            $relatedPlans = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'related_plans'))
            if ($relatedPlans.Count -ne 1 -or $relatedPlans[0] -notmatch '^PLN-\d{4}$') {
                $failures.Add("Implementation must identify exactly one matching related plan: $relativePath")
            } elseif ($relatedPlans[0] -ne $filenamePlanId) {
                $failures.Add("Implementation related plan does not match filename: $relativePath ($($relatedPlans[0]) != $filenamePlanId)")
            }
            $planAcceptanceAudits = @(Get-ListValues (Get-FrontmatterValue $frontmatter 'plan_acceptance_audits'))
            if ($planAcceptanceAudits.Count -eq 0) {
                $failures.Add("Implementation must reference at least one plan acceptance audit: $relativePath")
            }
            foreach ($planAcceptanceAudit in $planAcceptanceAudits) {
                if ($planAcceptanceAudit -notmatch '^AUD-\d{4}$') {
                    $failures.Add("Invalid plan acceptance audit ID in implementation: $relativePath ($planAcceptanceAudit)")
                }
            }
            $resultRevision = Get-FrontmatterValue $frontmatter 'result_revision'
            if ($implementationStatus -eq 'completed' -and $resultRevision -notmatch '^git:[0-9a-fA-F]{40}$') {
                $failures.Add("Completed implementation must record a full git result_revision: $relativePath")
            }
            $implementationMetadata[$implementationId] = @{
                Path = $relativePath
                Status = $implementationStatus
                PlanId = if ($relatedPlans.Count -eq 1) { $relatedPlans[0] } else { $filenamePlanId }
                PlanAcceptanceAudits = $planAcceptanceAudits
                ResultRevision = $resultRevision
            }
            $implementationRecords[$file.Name] = $implementationStatus
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

foreach ($auditInfo in $auditMetadata.Values) {
    foreach ($relatedPlan in @($auditInfo.RelatedPlans)) {
        if (-not $planMetadata.ContainsKey($relatedPlan)) {
            $failures.Add("Audit references a missing plan: $($auditInfo.Path) ($relatedPlan)")
        }
    }
    foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
        if (-not $implementationMetadata.ContainsKey($relatedImplementation)) {
            $failures.Add("Audit references a missing implementation: $($auditInfo.Path) ($relatedImplementation)")
        }
    }
    foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($relatedAudit)) {
            $failures.Add("Audit references a missing related audit: $($auditInfo.Path) ($relatedAudit)")
        }
    }
    if ($auditInfo.Schema -eq 'plan-acceptance/v1') {
        foreach ($relatedPlan in @($auditInfo.RelatedPlans)) {
            $matchingPlanAudits = New-Object System.Collections.Generic.List[string]
            foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
                if (-not $auditMetadata.ContainsKey($relatedAudit)) { continue }
                $relatedAuditInfo = $auditMetadata[$relatedAudit]
                if ($relatedAuditInfo.Schema -eq 'plan-audit/v2' -and
                    $relatedAuditInfo.Status -eq 'closed' -and
                    $relatedAuditInfo.RelatedPlans -contains $relatedPlan) {
                    $matchingPlanAudits.Add($relatedAudit)
                }
            }
            if ($matchingPlanAudits.Count -eq 0) {
                $failures.Add("Plan acceptance must reference a matching closed plan-audit/v2 record: $($auditInfo.Path) ($relatedPlan)")
            }
        }
    }
    if ($auditInfo.Schema -eq 'implementation-acceptance/v1') {
        foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
            $matchingImplementationAudits = New-Object System.Collections.Generic.List[string]
            foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
                if (-not $auditMetadata.ContainsKey($relatedAudit)) { continue }
                $relatedAuditInfo = $auditMetadata[$relatedAudit]
                if ($relatedAuditInfo.Schema -eq 'implementation-audit/v1' -and
                    $relatedAuditInfo.Status -eq 'closed' -and
                    $relatedAuditInfo.RelatedImplementations -contains $relatedImplementation) {
                    $matchingImplementationAudits.Add($relatedAudit)
                }
            }
            if ($matchingImplementationAudits.Count -eq 0) {
                $failures.Add("Implementation acceptance must reference a matching closed implementation-audit/v1 record: $($auditInfo.Path) ($relatedImplementation)")
            }
        }
        if ($auditInfo.RelatedPlans.Count -eq 0 -or $auditInfo.RelatedImplementations.Count -ne $auditInfo.RelatedPlans.Count) {
            $failures.Add("Implementation acceptance must map exactly one IMP to each related plan: $($auditInfo.Path)")
        } else {
            foreach ($relatedImplementation in @($auditInfo.RelatedImplementations)) {
                if ($implementationMetadata.ContainsKey($relatedImplementation) -and
                    $relatedImplementation -and
                    $auditInfo.RelatedPlans -notcontains $implementationMetadata[$relatedImplementation].PlanId) {
                    $failures.Add("Implementation acceptance IMP does not belong to a related plan: $($auditInfo.Path) ($relatedImplementation)")
                }
            }
        }
    }
}

foreach ($implementationInfo in $implementationMetadata.Values) {
    if (-not $planMetadata.ContainsKey($implementationInfo.PlanId)) {
        $failures.Add("Implementation references a missing plan: $($implementationInfo.Path) ($($implementationInfo.PlanId))")
    }
    foreach ($planAcceptanceAudit in @($implementationInfo.PlanAcceptanceAudits)) {
        if (-not $auditMetadata.ContainsKey($planAcceptanceAudit)) {
            $failures.Add("Implementation references a missing plan acceptance audit: $($implementationInfo.Path) ($planAcceptanceAudit)")
            continue
        }
        $acceptanceInfo = $auditMetadata[$planAcceptanceAudit]
        if ($acceptanceInfo.Schema -ne 'plan-acceptance/v1' -or
            $acceptanceInfo.Verdict -ne 'ready' -or
            $acceptanceInfo.RelatedPlans -notcontains $implementationInfo.PlanId) {
            $failures.Add("Implementation plan acceptance audit is not the matching ready audit: $($implementationInfo.Path) ($planAcceptanceAudit)")
        }
    }
}

foreach ($remediationInfo in $remediationMetadata.Values) {
    foreach ($sourceAudit in @($remediationInfo.SourceAudits)) {
        if (-not $auditMetadata.ContainsKey($sourceAudit)) {
            $failures.Add("Remediation references a missing source audit: $($remediationInfo.Path) ($sourceAudit)")
        }
    }
}

$phaseFivePlanPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle.md'
$phaseFiveChecklistPath = Join-Path $docsRoot 'plans\PLN-0005-phase-05-attachment-lifecycle-checklist.md'
if ((Test-Path -LiteralPath $phaseFivePlanPath) -and (Test-Path -LiteralPath $phaseFiveChecklistPath)) {
    $phaseFivePlan = Get-Content -Raw -Encoding UTF8 $phaseFivePlanPath
    $phaseFiveChecklist = Get-Content -Raw -Encoding UTF8 $phaseFiveChecklistPath
    $phaseFiveDag = Get-PhaseFiveDag $phaseFivePlan $failures
    Test-PhaseFiveFactsOutput $phaseFivePlan $failures
    Test-PhaseFiveDependencyStatements $phaseFivePlan $phaseFiveDag $failures
    Test-PhaseFiveDependencyStatements $phaseFiveChecklist $phaseFiveDag $failures

    $deploymentContract = [regex]::Match($phaseFivePlan, '(?s)<!--\s*phase5-p0-deployment-evidence-contract\s*\r?\n(?<body>.*?)\r?\n-->')
    $forbiddenP0LiveEvidence = @()
    if ($deploymentContract.Success) {
        $forbiddenMatch = [regex]::Match($deploymentContract.Groups['body'].Value, '(?m)^forbidden_p0_live_evidence:\s*(?<value>[^\r\n]+?)\s*$')
        if ($forbiddenMatch.Success) {
            $forbiddenP0LiveEvidence = @($forbiddenMatch.Groups['value'].Value.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { $_ })
        }
    }
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

    Test-PhaseFiveDeploymentClauses $phaseFiveChecklist $forbiddenP0LiveEvidence $true $failures
    Test-PhaseFiveDeploymentClauses $phaseFivePlan $forbiddenP0LiveEvidence $false $failures

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
            $auditIdForEntry = [regex]::Match($entry.Key, '^(AUD-\d{4})-').Groups[1].Value
            $remediationStateMatch = [regex]::Match($indexLine, 'remediation=(?<state>pending|required|none|accepted-risk|awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4})')
            if ($remediationStateMatch.Success) {
                $auditRemediationStates[$auditIdForEntry] = $remediationStateMatch.Groups['state'].Value
            }
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'remediation=(pending|required|none|accepted-risk|awaiting-verification:REM-\d{4}|verified-by:AUD-\d{4}|continued-by:AUD-\d{4})') {
                $failures.Add("Audit index status is missing or inconsistent: $($entry.Key)")
            } else {
                $auditInfo = $auditMetadata[$auditIdForEntry]
                if ($null -ne $auditInfo -and $auditInfo.Schema -in @('plan-acceptance/v1', 'implementation-acceptance/v1')) {
                    $expectedRemediation = if ($auditInfo.Verdict -eq 'pending') { 'pending' } elseif ($auditInfo.Verdict -in @('ready', 'complete')) { 'none' } else { 'required' }
                    if ($indexLine -notmatch ('remediation=' + [regex]::Escape($expectedRemediation) + '\b')) {
                        $failures.Add("Acceptance audit index remediation does not match verdict: $($entry.Key) (expected $expectedRemediation)")
                    }
                }
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

$implementationIndexPath = Join-Path $docsRoot 'implementations\README.md'
if ($implementationRecords.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $implementationIndexPath)) {
        $failures.Add('Implementation records exist but implementations/README.md is missing')
    } else {
        $implementationIndexContent = Get-Content -Raw -Encoding UTF8 $implementationIndexPath
        foreach ($entry in $implementationRecords.GetEnumerator()) {
            $target = './records/' + $entry.Key
            $indexLines = Get-IndexLines $implementationIndexContent $target
            if ($indexLines.Count -ne 1) {
                $failures.Add("Implementation record must be indexed exactly once: $($entry.Key)")
                continue
            }
            $indexLine = $indexLines[0].Value
            if ($indexLine -notmatch ('status=' + [regex]::Escape($entry.Value)) -or
                $indexLine -notmatch 'audit=(not-ready|pending|audited-by:AUD-\d{4})' -or
                $indexLine -notmatch 'acceptance=(not-ready|pending|accepted-by:AUD-\d{4}|rejected-by:AUD-\d{4})') {
                $failures.Add("Implementation index status is missing or inconsistent: $($entry.Key)")
            } else {
                $implementationIdForEntry = [regex]::Match($entry.Key, '^(IMP-\d{4})-').Groups[1].Value
                $auditMatch = [regex]::Match($indexLine, 'audit=audited-by:(?<audit>AUD-\d{4})')
                if ($auditMatch.Success) {
                    $implementationAuditId = $auditMatch.Groups['audit'].Value
                    if (-not $auditMetadata.ContainsKey($implementationAuditId) -or
                        $auditMetadata[$implementationAuditId].Schema -ne 'implementation-audit/v1' -or
                        $auditMetadata[$implementationAuditId].RelatedImplementations -notcontains $implementationIdForEntry) {
                        $failures.Add("Implementation audit index references a non-matching audit: $($entry.Key) ($implementationAuditId)")
                    }
                }
                $acceptanceMatch = [regex]::Match($indexLine, 'acceptance=(?<state>accepted-by|rejected-by):(?<audit>AUD-\d{4})')
                if ($acceptanceMatch.Success) {
                    if ($acceptanceMatch.Groups['state'].Value -eq 'accepted-by' -and -not $auditMatch.Success) {
                        $failures.Add("Accepted implementation must reference a completed implementation audit: $($entry.Key)")
                    }
                    $acceptanceAuditId = $acceptanceMatch.Groups['audit'].Value
                    if (-not $auditMetadata.ContainsKey($acceptanceAuditId) -or
                        $auditMetadata[$acceptanceAuditId].Schema -ne 'implementation-acceptance/v1' -or
                        $auditMetadata[$acceptanceAuditId].RelatedImplementations -notcontains $implementationIdForEntry) {
                        $failures.Add("Implementation acceptance index references a non-matching audit: $($entry.Key) ($acceptanceAuditId)")
                    } else {
                        $expectedState = if ($auditMetadata[$acceptanceAuditId].Verdict -eq 'complete') { 'accepted-by' } else { 'rejected-by' }
                        if ($acceptanceMatch.Groups['state'].Value -ne $expectedState) {
                            $failures.Add("Implementation acceptance index state does not match audit verdict: $($entry.Key)")
                        }
                    }
                }
            }
        }
    }
}

foreach ($auditInfo in $auditMetadata.Values) {
    if ($auditInfo.Schema -notin @('plan-acceptance/v1', 'implementation-acceptance/v1') -or
        $auditInfo.Verdict -notin @('ready', 'complete')) {
        continue
    }
    foreach ($relatedAudit in @($auditInfo.RelatedAudits)) {
        if (-not $auditMetadata.ContainsKey($relatedAudit)) { continue }
        if ($auditMetadata[$relatedAudit].Status -ne 'closed') {
            $failures.Add("Successful acceptance cannot reference an open audit: $($auditInfo.Path) ($relatedAudit)")
        }
        if (-not $auditRemediationStates.ContainsKey($relatedAudit)) {
            $failures.Add("Successful acceptance cannot verify an unindexed audit chain: $($auditInfo.Path) ($relatedAudit)")
            continue
        }
        $relatedRemediationState = $auditRemediationStates[$relatedAudit]
        if ($relatedRemediationState -in @('pending', 'required') -or $relatedRemediationState -like 'awaiting-verification:*') {
            $failures.Add("Successful acceptance requires a clean related audit chain: $($auditInfo.Path) ($relatedAudit=$relatedRemediationState)")
        }
    }
}

$repositoryDocsRoot = (Join-Path $repoRoot 'docs')
if ($docsRoot -eq $repositoryDocsRoot) {
    $requiredAuditEntrypoints = @(
        '.github/prompts/backend-plan-audit.prompt.md',
        '.github/prompts/backend-plan-acceptance-audit.prompt.md',
        '.github/prompts/backend-implement-plan.prompt.md',
        '.github/prompts/backend-implementation-audit.prompt.md',
        '.github/prompts/backend-implementation-acceptance-audit.prompt.md',
        '.github/prompts/backend-fix-audit-findings.prompt.md',
        '.github/prompts/backend-follow-up-audit.prompt.md',
        '.agents/skills/backend-plan-audit/SKILL.md',
        '.agents/skills/backend-plan-audit/agents/openai.yaml',
        '.agents/skills/backend-plan-acceptance-audit/SKILL.md',
        '.agents/skills/backend-plan-acceptance-audit/agents/openai.yaml',
        '.agents/skills/backend-implement-plan/SKILL.md',
        '.agents/skills/backend-implement-plan/agents/openai.yaml',
        '.agents/skills/backend-implementation-audit/SKILL.md',
        '.agents/skills/backend-implementation-audit/agents/openai.yaml',
        '.agents/skills/backend-implementation-acceptance-audit/SKILL.md',
        '.agents/skills/backend-implementation-acceptance-audit/agents/openai.yaml',
        '.agents/skills/backend-fix-audit-findings/SKILL.md',
        '.agents/skills/backend-fix-audit-findings/agents/openai.yaml',
        '.agents/skills/backend-follow-up-audit/SKILL.md',
        '.agents/skills/backend-follow-up-audit/agents/openai.yaml',
        '.agents/skills/backend-plan-audit-until-ready/SKILL.md',
        '.agents/skills/backend-plan-audit-until-ready/agents/openai.yaml',
        '.agents/skills/backend-implement-audit-until-complete/SKILL.md',
        '.agents/skills/backend-implement-audit-until-complete/agents/openai.yaml',
        'docs/implementations/README.md',
        'docs/implementations/templates/implementation-record.md'
    )
    foreach ($entrypoint in $requiredAuditEntrypoints) {
        if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $entrypoint))) {
            $failures.Add("Missing audit command entrypoint: $entrypoint")
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

    $promptContracts = @{
        'backend-plan-acceptance-audit' = 'acceptance-contract:\s*plan-readiness;\s*default-target=active;\s*independent=true;\s*creates-audit'
        'backend-implement-plan' = 'implementation-contract:\s*creates-imp-record;\s*default-target=active;\s*explicit-targets=true'
        'backend-implementation-audit' = 'implementation-audit-contract:\s*default-target=pending-implementations;\s*creates-audit'
        'backend-implementation-acceptance-audit' = 'acceptance-contract:\s*implementation-completion;\s*default-target=active;\s*independent=true;\s*creates-audit'
    }
    foreach ($promptName in $promptContracts.Keys) {
        $promptPath = Join-Path $repoRoot ".github/prompts/$promptName.prompt.md"
        if (Test-Path -LiteralPath $promptPath) {
            $promptContent = Get-Content -Raw -Encoding UTF8 $promptPath
            if ($promptContent -notmatch "(?m)^name:\s*$promptName\s*$" -or
                $promptContent -notmatch $promptContracts[$promptName]) {
                $failures.Add("Prompt contract is missing or invalid: $promptName")
            }
            if ($promptName -in @('backend-plan-acceptance-audit', 'backend-implementation-acceptance-audit') -and
                ($promptContent -notmatch 'independence_basis' -or $promptContent -notmatch 'evidence_revision')) {
                $failures.Add("Acceptance prompt must define independence and evidence revision requirements: $promptName")
            }
            if ($promptName -eq 'backend-plan-acceptance-audit' -and $promptContent -notmatch 'PLAN_AUDIT_CHAIN_CLEAN') {
                $failures.Add('Plan acceptance prompt must require a clean plan audit chain')
            }
        }
    }

    foreach ($skillName in @('backend-plan-audit', 'backend-plan-acceptance-audit', 'backend-implement-plan', 'backend-implementation-audit', 'backend-implementation-acceptance-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit')) {
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

    $orchestrators = @{
        'backend-plan-audit-until-ready' = @('backend-plan-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit', 'backend-plan-acceptance-audit')
        'backend-implement-audit-until-complete' = @('backend-plan-acceptance-audit', 'backend-implement-plan', 'backend-implementation-audit', 'backend-fix-audit-findings', 'backend-follow-up-audit', 'backend-implementation-acceptance-audit')
    }
    foreach ($orchestratorName in $orchestrators.Keys) {
        $orchestratorSkillPath = Join-Path $repoRoot ".agents/skills/$orchestratorName/SKILL.md"
        $orchestratorMetadataPath = Join-Path $repoRoot ".agents/skills/$orchestratorName/agents/openai.yaml"
        if (Test-Path -LiteralPath $orchestratorSkillPath) {
            $orchestratorContent = Get-Content -Raw -Encoding UTF8 $orchestratorSkillPath
            foreach ($dependencySkill in $orchestrators[$orchestratorName]) {
                if ($orchestratorContent -notmatch ('\$' + [regex]::Escape($dependencySkill))) {
                    $failures.Add("Audit orchestrator $orchestratorName must invoke existing skill: $dependencySkill")
                }
                if ($orchestratorContent -notmatch ('\$' + [regex]::Escape($dependencySkill) + '\s+TARGET=')) {
                    $failures.Add("Audit orchestrator $orchestratorName must propagate an explicit TARGET to: $dependencySkill")
                }
            }
            if ($orchestratorContent -notmatch 'MAX_CYCLES' -or
                $orchestratorContent -notmatch 'MAX_STAGNANT_CYCLES' -or
                $orchestratorContent -notmatch 'persistent goal') {
                $failures.Add("Audit orchestrator must establish a goal and enforce bounded loop circuit breakers: $orchestratorName")
            }
        }
        if (Test-Path -LiteralPath $orchestratorMetadataPath) {
            $orchestratorMetadata = Get-Content -Raw -Encoding UTF8 $orchestratorMetadataPath
            if ($orchestratorMetadata -notmatch '(?m)^\s*allow_implicit_invocation:\s*false\s*$') {
                $failures.Add("Audit orchestrator must require explicit invocation: $orchestratorName")
            }
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

    $trackedImplementationPaths = & git -C $repoRoot ls-tree -r --name-only HEAD -- docs/implementations/records 2>$null
    foreach ($trackedImplementationPath in $trackedImplementationPaths) {
        $currentImplementationPath = Join-Path $repoRoot $trackedImplementationPath
        if (-not (Test-Path -LiteralPath $currentImplementationPath)) {
            $failures.Add("Tracked implementation record must not be deleted or moved: $trackedImplementationPath")
            continue
        }
        $headImplementationContent = (& git -C $repoRoot show "HEAD:$trackedImplementationPath" 2>$null | Out-String)
        if ($headImplementationContent -match '(?m)^status:\s*(completed|partial|blocked)\s*$') {
            & git -C $repoRoot diff --quiet HEAD -- $trackedImplementationPath
            if ($LASTEXITCODE -ne 0) {
                $failures.Add("Closed implementation record is immutable; create a new implementation instead: $trackedImplementationPath")
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
