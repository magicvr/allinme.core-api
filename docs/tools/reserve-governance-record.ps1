param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('AUD', 'REM', 'IMP')]
    [string]$Kind,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^\d{8}-[a-z0-9]+(?:-[a-z0-9]+)*$')]
    [string]$Suffix,

    [string]$RepositoryRoot
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
} else {
    $repoRoot = (Resolve-Path $RepositoryRoot).Path
}

$recordsDirectory = switch ($Kind) {
    'AUD' { Join-Path $repoRoot 'docs\audits\records' }
    'REM' { Join-Path $repoRoot 'docs\remediations\records' }
    'IMP' { Join-Path $repoRoot 'docs\implementations\records' }
}

if (-not (Test-Path -LiteralPath $recordsDirectory -PathType Container)) {
    throw "Records directory does not exist: $recordsDirectory"
}

$sha256 = [System.Security.Cryptography.SHA256]::Create()
try {
    $rootHashBytes = $sha256.ComputeHash([Text.Encoding]::UTF8.GetBytes($repoRoot.ToLowerInvariant()))
} finally {
    $sha256.Dispose()
}
$rootHash = ([BitConverter]::ToString($rootHashBytes)).Replace('-', '').Substring(0, 16)
$mutex = New-Object System.Threading.Mutex($false, "allinme-core-api-governance-$rootHash")
$lockTaken = $false

try {
    $lockTaken = $mutex.WaitOne([TimeSpan]::FromSeconds(30))
    if (-not $lockTaken) {
        throw 'Timed out waiting for the governance record allocator'
    }

    $maxNumber = 0
    foreach ($file in Get-ChildItem -LiteralPath $recordsDirectory -File) {
        if ($file.Name -match "^$Kind-(?<number>\d{4})-") {
            $number = [int]$Matches['number']
            if ($number -gt $maxNumber) {
                $maxNumber = $number
            }
        }
    }

    $number = $maxNumber + 1
    while ($number -le 9999) {
        $recordId = '{0}-{1:D4}' -f $Kind, $number
        $fileName = "$recordId-$Suffix.md"
        $targetPath = Join-Path $recordsDirectory $fileName
        try {
            $stream = [IO.File]::Open($targetPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
            $stream.Dispose()
            $relativePath = $targetPath.Substring($repoRoot.Length + 1).Replace('\', '/')
            Write-Output "$recordId`t$relativePath"
            exit 0
        } catch [IO.IOException] {
            $number++
        }
    }

    throw "No available $Kind identifiers remain"
} finally {
    if ($lockTaken) {
        $mutex.ReleaseMutex()
    }
    $mutex.Dispose()
}
