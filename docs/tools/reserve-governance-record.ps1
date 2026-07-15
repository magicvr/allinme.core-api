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
$gitCommand = Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1
$gitExecutable = [IO.Path]::GetFullPath($gitCommand.Source)
$repoPathPrefix = $repoRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
if ($gitExecutable.StartsWith($repoPathPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to execute a repository-local Git binary: $gitExecutable"
}

function git {
    & $gitExecutable @args
}

$recordsDirectory = switch ($Kind) {
    'AUD' { Join-Path $repoRoot 'docs\audits\records' }
    'REM' { Join-Path $repoRoot 'docs\remediations\records' }
    'IMP' { Join-Path $repoRoot 'docs\implementations\records' }
}

if (-not (Test-Path -LiteralPath $recordsDirectory -PathType Container)) {
    throw "Records directory does not exist: $recordsDirectory"
}

$gitTopLevel = (& git -C $repoRoot rev-parse --show-toplevel 2>&1 | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($gitTopLevel)) {
    throw 'Unable to resolve the Git repository root'
}
$resolvedGitTopLevel = [IO.Path]::GetFullPath($gitTopLevel)
if (-not [string]::Equals($resolvedGitTopLevel.TrimEnd('\', '/'), $repoRoot.TrimEnd('\', '/'), [StringComparison]::OrdinalIgnoreCase)) {
    throw "RepositoryRoot must be the Git top-level directory: supplied=$repoRoot actual=$resolvedGitTopLevel"
}

$gitCommonDirectoryValue = (& git -C $repoRoot rev-parse --git-common-dir 2>&1 | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($gitCommonDirectoryValue)) {
    throw 'Unable to resolve the Git common directory'
}
$gitCommonDirectory = if ([IO.Path]::IsPathRooted($gitCommonDirectoryValue)) {
    [IO.Path]::GetFullPath($gitCommonDirectoryValue)
} else {
    [IO.Path]::GetFullPath((Join-Path $repoRoot $gitCommonDirectoryValue))
}
$reservationsDirectory = Join-Path $gitCommonDirectory 'allinme-governance-reservations'
[IO.Directory]::CreateDirectory($reservationsDirectory) | Out-Null

$sha256 = [System.Security.Cryptography.SHA256]::Create()
try {
    $rootHashBytes = $sha256.ComputeHash([Text.Encoding]::UTF8.GetBytes($gitCommonDirectory.ToLowerInvariant()))
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
    foreach ($reservation in Get-ChildItem -LiteralPath $reservationsDirectory -File -Filter "$Kind-*.lock") {
        if ($reservation.Name -match "^$Kind-(?<number>\d{4})\.lock$") {
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
        $reservationPath = Join-Path $reservationsDirectory "$recordId.lock"
        $reservationCreated = $false
        try {
            $reservationStream = [IO.File]::Open($reservationPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
            $reservationCreated = $true
            try {
                $reservationBytes = [Text.Encoding]::UTF8.GetBytes("$repoRoot`n$targetPath`n")
                $reservationStream.Write($reservationBytes, 0, $reservationBytes.Length)
            } finally {
                $reservationStream.Dispose()
            }
            $stream = [IO.File]::Open($targetPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
            $stream.Dispose()
            $relativePath = $targetPath.Substring($repoRoot.Length + 1).Replace('\', '/')
            Write-Output "$recordId`t$relativePath"
            return
        } catch [IO.IOException] {
            if ($reservationCreated -and (Test-Path -LiteralPath $reservationPath -PathType Leaf) -and -not (Test-Path -LiteralPath $targetPath -PathType Leaf)) {
                Remove-Item -LiteralPath $reservationPath -Force
            }
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
