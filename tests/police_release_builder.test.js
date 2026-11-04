const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const test = require('node:test');

test('police release builder captures runtime working trees without private config', () => {
  const starterRoot = path.resolve(__dirname, '..');
  const sourceRoot = path.resolve(starterRoot, '..');
  const outputRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'mz-police-release-'));
  const releaseName = 'contract-test';

  try {
    execFileSync('powershell.exe', [
      '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File',
      path.join(starterRoot, 'tools', 'build_police_release.ps1'),
      '-SourceRoot', sourceRoot,
      '-OutputRoot', outputRoot,
      '-ReleaseName', releaseName,
      '-SkipArchive',
    ], { stdio: 'pipe' });

    const releaseRoot = path.join(outputRoot, releaseName);
    const manifest = JSON.parse(fs.readFileSync(path.join(releaseRoot, 'release-manifest.json'), 'utf8'));
    const names = manifest.resources.map((resource) => resource.name);

    assert.equal(manifest.includesWorkingTreeState, true);
    assert.match(manifest.starter.commit, /^[0-9a-f]{40}$/);
    assert.deepEqual(names, [
      'mz_notify', 'mz_core', 'mz_economy', 'mz_progress', 'mz_menu', 'mz_interact',
      'mz_world_editor', 'mz_vehicles', 'mz_org', 'mz_inventory', 'mz_target', 'mz_radialmenu', 'mz_fuel',
      'mz_garagem', 'mz_prison', 'mz_dispatch', 'mz_police', 'mz_medical', 'mz_phone',
    ]);
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_police', 'server', 'records', 'service.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_prison', 'server', 'prepare.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_target', 'client', 'main.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_radialmenu', 'client', 'main.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_dispatch', 'server', 'service.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_dispatch', 'sql', 'mz_dispatch.sql')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_police', 'sql', 'records_schema_additive.sql')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_phone', 'web', 'apps', 'emergency.js')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_medical', 'server', 'service.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'starter', 'cfg', 'resources.cfg')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'docs', 'police', 'POLICE_GAMEPLAY_GUIDE.md')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'docs', 'police', 'P8_DISPATCH_IMPLEMENTATION_REPORT.md')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'docs', 'police', 'P9_IMPLEMENTATION_REPORT.md')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'docs', 'police', 'P9_POLICE_OPERATIONAL_GUIDE.md')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'SHA256SUMS')));
    assert.equal(manifest.resources.find((resource) => resource.name === 'mz_dispatch').vcsAvailable, false);

    const files = fs.readdirSync(releaseRoot, { recursive: true }).map(String);
    assert.equal(files.some((file) => /(^|[\\/])(server|database|permissions)\.cfg$/i.test(file)), false);
    assert.equal(files.some((file) => /(^|[\\/])\.env($|\.)/i.test(file)), false);
  } finally {
    fs.rmSync(outputRoot, { recursive: true, force: true });
  }
});
