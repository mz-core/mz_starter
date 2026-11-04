$ErrorActionPreference = 'Stop'

$root = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$oxRoot = Join-Path $root 'resources\[ox]'
$tmpRoot = Join-Path $root 'tmp'
$session = Join-Path $tmpRoot ('ox-install-' + [guid]::NewGuid().ToString('N'))

# All targets are fixed paths below this server-data root. Never use the old
# tmp\oxmysql.zip: a failed download may have left a truncated archive there.
$releases = @(
    @{ Name = 'oxmysql'; Url = 'https://github.com/overextended/oxmysql/releases/latest/download/oxmysql.zip'; Build = 'dist/build.js' },
    @{ Name = 'ox_lib'; Url = 'https://github.com/overextended/ox_lib/releases/latest/download/ox_lib.zip'; Build = 'web/build/index.html' }
)

function Test-ReleaseArchive {
    param([string] $Archive, [string] $Name, [string] $Build)

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($Archive)
    try {
        $entries = @($zip.Entries | ForEach-Object { $_.FullName.Replace('\', '/') })
        foreach ($required in @("$Name/fxmanifest.lua", "$Name/$Build")) {
            if ($entries -notcontains $required) {
                throw "$Name release incompleta: falta $required"
            }
        }
    } finally {
        $zip.Dispose()
    }
}

$movedOld = @{}
$installedNew = @{}

try {
    New-Item -ItemType Directory -Path $oxRoot, $session -Force | Out-Null

    # Download and validate BOTH releases before changing either live resource.
    foreach ($release in $releases) {
        $name = $release.Name
        $archive = Join-Path $session "$name.zip"
        $extract = Join-Path $session "extract-$name"
        Write-Host "Baixando e validando $name..."
        & curl.exe --fail --location --retry 3 --retry-all-errors --silent --show-error --output $archive $release.Url
        if ($LASTEXITCODE -ne 0) { throw "Download de $name falhou (curl: $LASTEXITCODE)." }
        Test-ReleaseArchive -Archive $archive -Name $name -Build $release.Build
        Expand-Archive -LiteralPath $archive -DestinationPath $extract
        foreach ($required in @('fxmanifest.lua', $release.Build)) {
            if (-not (Test-Path -LiteralPath (Join-Path (Join-Path $extract $name) $required) -PathType Leaf)) {
                throw "$name extraido sem $required"
            }
        }
    }

    foreach ($release in $releases) {
        $name = $release.Name
        $target = Join-Path $oxRoot $name
        $backup = Join-Path $session "backup-$name"
        $source = Join-Path (Join-Path $session "extract-$name") $name

        if (Test-Path -LiteralPath $target) {
            Move-Item -LiteralPath $target -Destination $backup -ErrorAction Stop
            $movedOld[$name] = $backup
        }
        Move-Item -LiteralPath $source -Destination $target -ErrorAction Stop
        $installedNew[$name] = $target
    }

    Write-Host 'oxmysql e ox_lib instalados e validados.'
    if ($movedOld.Count -gt 0) {
        Write-Host "Backups recuperaveis dos recursos anteriores: $session"
    }
    exit 0
} catch {
    [Console]::Error.WriteLine("Instalacao ox falhou: $($_.Exception.Message)")
    for ($index = $releases.Count - 1; $index -ge 0; $index--) {
        $release = $releases[$index]
        $name = $release.Name
        $target = Join-Path $oxRoot $name
        if ($installedNew.ContainsKey($name) -and (Test-Path -LiteralPath $target)) {
            Move-Item -LiteralPath $target -Destination (Join-Path $session "failed-$name") -ErrorAction Continue
        }
        if ($movedOld.ContainsKey($name) -and (Test-Path -LiteralPath $movedOld[$name])) {
            Move-Item -LiteralPath $movedOld[$name] -Destination $target -ErrorAction Continue
        }
    }
    [Console]::Error.WriteLine("Arquivos de diagnostico preservados em: $session")
    exit 1
}
