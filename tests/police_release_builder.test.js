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
      'mz_core', 'mz_economy', 'mz_menu', 'mz_interact', 'mz_vehicles',
      'mz_org', 'mz_inventory', 'mz_fuel', 'mz_garagem', 'mz_police', 'mz_prison',
    ]);
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_police', 'server', 'records.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'resources', 'mz_prison', 'server', 'prepare.lua')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'starter', 'cfg', 'resources.cfg')));
    assert.ok(fs.existsSync(path.join(releaseRoot, 'SHA256SUMS')));

    const files = fs.readdirSync(releaseRoot, { recursive: true }).map(String);
    assert.equal(files.some((file) => /(^|[\\/])(server|database|permissions)\.cfg$/i.test(file)), false);
    assert.equal(files.some((file) => /(^|[\\/])\.env($|\.)/i.test(file)), false);
  } finally {
    fs.rmSync(outputRoot, { recursive: true, force: true });
  }
});
