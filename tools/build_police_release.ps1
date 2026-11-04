param(
  [string]$SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path,
  [string]$OutputRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path 'release-artifacts'),
  [string]$ReleaseName = ('mz-police-p9-advanced-rc-{0}' -f (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ')),
  [switch]$SkipArchive
)

$ErrorActionPreference = 'Stop'

$resourceNames = @(
  'mz_notify', 'mz_core', 'mz_economy', 'mz_progress', 'mz_menu', 'mz_interact',
  'mz_world_editor', 'mz_vehicles', 'mz_org', 'mz_inventory', 'mz_target', 'mz_radialmenu', 'mz_fuel',
  'mz_garagem', 'mz_prison', 'mz_dispatch', 'mz_police', 'mz_medical', 'mz_phone'
)
$runtimeDirectories = @('client', 'server', 'shared', 'web', 'stream')
$runtimeRootFiles = @('fxmanifest.lua', 'config.lua')
$blockedNames = @(
  '.env', 'server.cfg', 'database.cfg', 'permissions.cfg', 'fivem_options.cfg',
  '*.pem', '*.key', '*.pfx', '*.p12', '*.kdbx', 'credentials*', 'secrets*'
)
$secretSignatures = @(
  'sv_licenseKey\s+', 'steam_webApiKey\s+', 'set\s+mysql_connection_string\s+',
  'discord_bot_token\s*=', '-----BEGIN [A-Z ]*PRIVATE KEY-----'
)

$sourcePath = [System.IO.Path]::GetFullPath($SourceRoot)
$outputPath = [System.IO.Path]::GetFullPath($OutputRoot)
$releasePath = Join-Path $outputPath $ReleaseName
$zipPath = "$releasePath.zip"

if (-not (Test-Path -LiteralPath $sourcePath -PathType Container)) {
  throw "Source root not found: $sourcePath"
}
if ((Test-Path -LiteralPath $releasePath) -or (Test-Path -LiteralPath $zipPath)) {
  throw "Release output already exists: $ReleaseName"
}

New-Item -ItemType Directory -Path $releasePath -Force | Out-Null

function Test-BlockedName([string]$name) {
  foreach ($pattern in $blockedNames) {
    if ($name -like $pattern) { return $true }
  }
  return $false
}

function Assert-NoSecretSignature([System.IO.FileInfo]$file) {
  if ($file.Extension -notin @('.cfg', '.env', '.ini', '.lua', '.json', '.toml', '.yaml', '.yml')) { return }
  $content = Get-Content -LiteralPath $file.FullName -Raw -ErrorAction Stop
  foreach ($signature in $secretSignatures) {
    if ($content -match $signature) {
      throw "Possible credential signature in release input: $($file.FullName)"
    }
  }
}

function Get-Sha256Hex([string]$path) {
  $stream = [System.IO.File]::OpenRead($path)
  $sha256 = [System.Security.Cryptography.SHA256]::Create()
  try {
    return ([System.BitConverter]::ToString($sha256.ComputeHash($stream))).Replace('-', '').ToLowerInvariant()
  } finally {
    $sha256.Dispose()
    $stream.Dispose()
  }
}

function Copy-ReleaseFile([System.IO.FileInfo]$file, [string]$relativePath) {
  if (Test-BlockedName $file.Name) {
    throw "Blocked private file in release input: $($file.FullName)"
  }
  Assert-NoSecretSignature $file
  $destination = Join-Path $releasePath $relativePath
  New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force | Out-Null
  Copy-Item -LiteralPath $file.FullName -Destination $destination
}

$resourceStates = @()
foreach ($resourceName in $resourceNames) {
  $resourcePath = Join-Path $sourcePath $resourceName
  $manifestPath = Join-Path $resourcePath 'fxmanifest.lua'
  if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "Missing resource manifest: $manifestPath"
  }

  foreach ($rootFile in $runtimeRootFiles) {
    $candidate = Join-Path $resourcePath $rootFile
    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
      Copy-ReleaseFile (Get-Item -LiteralPath $candidate) (Join-Path "resources\$resourceName" $rootFile)
    }
  }
  foreach ($directory in $runtimeDirectories) {
    $candidate = Join-Path $resourcePath $directory
    if (-not (Test-Path -LiteralPath $candidate -PathType Container)) { continue }
    Get-ChildItem -LiteralPath $candidate -File -Recurse | ForEach-Object {
      $childRelative = $_.FullName.Substring($resourcePath.Length).TrimStart('\', '/')
      Copy-ReleaseFile $_ (Join-Path "resources\$resourceName" $childRelative)
    }
  }
  if ($resourceName -in @('mz_dispatch', 'mz_police')) {
    foreach ($directory in @('sql', 'docs')) {
      $candidate = Join-Path $resourcePath $directory
      if (-not (Test-Path -LiteralPath $candidate -PathType Container)) { continue }
      Get-ChildItem -LiteralPath $candidate -File -Recurse | ForEach-Object {
        $childRelative = $_.FullName.Substring($resourcePath.Length).TrimStart('\', '/')
        Copy-ReleaseFile $_ (Join-Path "resources\$resourceName" $childRelative)
      }
    }
  }

  $manifest = Get-Content -LiteralPath $manifestPath -Raw
  $versionMatch = [regex]::Match($manifest, '(?m)^version\s+[''"]([^''"]+)[''"]')
  $vcsAvailable = Test-Path -LiteralPath (Join-Path $resourcePath '.git')
  $commit = if ($vcsAvailable) { (& git -C $resourcePath rev-parse HEAD 2>$null) } else { $null }
  $branch = if ($vcsAvailable) { (& git -C $resourcePath branch --show-current 2>$null) } else { $null }
  $statusLines = if ($vcsAvailable) { @(& git -C $resourcePath status --porcelain 2>$null) } else { @() }
  $resourceStates += [ordered]@{
    name = $resourceName
    version = if ($versionMatch.Success) { $versionMatch.Groups[1].Value } else { $null }
    commit = if ($vcsAvailable) { [string]$commit } else { $null }
    branch = if ($vcsAvailable) { [string]$branch } else { $null }
    vcsAvailable = $vcsAvailable
    dirty = if ($vcsAvailable) { $statusLines.Count -gt 0 } else { $null }
    statusEntryCount = if ($vcsAvailable) { $statusLines.Count } else { $null }
  }
}

$starterFiles = @(
  'cfg/resources.cfg', 'tools/install_deps.bat', 'tools/install_deps.sh',
  'tools/check_structure.bat', 'tools/check_structure.sh'
)
foreach ($relative in $starterFiles) {
  $candidate = Join-Path (Join-Path $sourcePath 'mz_starter') $relative
  if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
    throw "Missing starter release file: $candidate"
  }
  Copy-ReleaseFile (Get-Item -LiteralPath $candidate) (Join-Path 'starter' $relative)
}
$starterPath = Join-Path $sourcePath 'mz_starter'
$starterStatus = @(& git -C $starterPath status --porcelain 2>$null)
$starterState = [ordered]@{
  commit = [string](& git -C $starterPath rev-parse HEAD 2>$null)
  branch = [string](& git -C $starterPath branch --show-current 2>$null)
  dirty = $starterStatus.Count -gt 0
  statusEntryCount = $starterStatus.Count
}

$documentationFiles = @(
  'police/POLICE_GAMEPLAY_GUIDE.md',
  'police/POLICE_STAGING_OPERATIONAL_CONFIGURATION.md',
  'police/P8_DISPATCH_IMPLEMENTATION_REPORT.md',
  'police/P9_IMPLEMENTATION_REPORT.md',
  'police/P9_POLICE_OPERATIONAL_GUIDE.md'
)
foreach ($relative in $documentationFiles) {
  $candidate = Join-Path (Join-Path $sourcePath 'mz_docs') $relative
  if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
    throw "Missing release documentation: $candidate"
  }
  Copy-ReleaseFile (Get-Item -LiteralPath $candidate) (Join-Path 'docs' $relative)
}

$manifest = [ordered]@{
  release = $ReleaseName
  generatedAtUtc = (Get-Date).ToUniversalTime().ToString('o')
  sourceRoot = $sourcePath
  includesWorkingTreeState = $true
  externalDependencies = @(
    'FXServer', 'oxmysql', 'ox_lib', 'spawnmanager', 'pma-voice', 'screenshot-basic', 'MySQL/MariaDB'
  )
  bootOrder = @(
    'oxmysql', 'ox_lib', 'spawnmanager', 'pma-voice', 'screenshot-basic', 'mz_notify',
    'mz_core', 'mz_economy', 'mz_progress', 'mz_menu', 'mz_interact', 'mz_world_editor',
    'mz_vehicles', 'mz_org', 'mz_inventory', 'mz_target', 'mz_radialmenu', 'mz_fuel', 'mz_garagem',
    'mz_prison', 'mz_dispatch', 'mz_police', 'mz_medical', 'mz_phone'
  )
  activationRequired = @(
    'Verify that mz_dispatch creates and validates its schema at boot; apply resources/mz_dispatch/sql/mz_dispatch.sql manually only as a fallback',
    'Add ensure mz_dispatch to the approved server boot configuration',
    'Set MZDispatchConfig.enabled only after schema and dependencies are verified',
    'Verify that mz_police creates/validates mz_police_forensic_results at boot; resources/mz_police/sql/records_schema_additive.sql is a manual fallback only',
    'Configure P9 capabilities, approved vehicle models, FLIR offset and real CCTV coordinates',
    'Keep MZPoliceConfig.Advanced.enabled false until the staging runtime gate is ready'
  )
  resources = $resourceStates
  starter = $starterState
  starterFiles = $starterFiles
  documentationFiles = $documentationFiles
}
$manifestPath = Join-Path $releasePath 'release-manifest.json'
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($manifestPath, ($manifest | ConvertTo-Json -Depth 5), $utf8WithoutBom)

$hashLines = Get-ChildItem -LiteralPath $releasePath -File -Recurse |
  Sort-Object FullName |
  ForEach-Object {
    $relative = $_.FullName.Substring($releasePath.Length).TrimStart('\', '/').Replace('\', '/')
    '{0}  {1}' -f (Get-Sha256Hex $_.FullName), $relative
  }
$hashLines | Set-Content -LiteralPath (Join-Path $releasePath 'SHA256SUMS') -Encoding ascii

$zipHash = $null
if (-not $SkipArchive) {
  Compress-Archive -Path (Join-Path $releasePath '*') -DestinationPath $zipPath -CompressionLevel Optimal
  $zipHash = Get-Sha256Hex $zipPath
}

[ordered]@{
  releaseDirectory = $releasePath
  archive = if ($SkipArchive) { $null } else { $zipPath }
  archiveSha256 = $zipHash
  files = $hashLines.Count
} | ConvertTo-Json
