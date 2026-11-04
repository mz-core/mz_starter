const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const root = path.resolve(__dirname, "..");
const repositoryRoot = path.resolve(root, "..");
const read = (name) => fs.readFileSync(path.join(root, name), "utf8");
const readRepositoryFile = (name) => fs.readFileSync(path.join(repositoryRoot, name), "utf8");

const bootResources = [...read("cfg/resources.cfg").matchAll(/^ensure\s+(mz_[\w-]+)\s*$/gm)]
  .map((match) => match[1]);
const bootLines = [...read("cfg/resources.cfg").matchAll(/^ensure\s+([\w-]+)\s*$/gm)]
  .map((match) => match[1]);

const installerResources = (script) => [...script.matchAll(/sync_(?:mz|maps)_resource\s+["']?(mz_[\w-]+)/g)]
  .map((match) => match[1]);

test("Windows and Linux installers cover the complete MZ boot manifest in boot order", () => {
  assert.ok(bootResources.length > 0);

  for (const name of ["tools/install_deps.bat", "tools/install_deps.sh"]) {
    assert.deepEqual(installerResources(read(name)), bootResources, name);
  }
});

test("structure preflight requires every MZ resource manifest", () => {
  const windows = read("tools/check_structure.bat");
  const linux = read("tools/check_structure.sh");

  for (const resource of bootResources) {
    const groupPath = resource === "mz_cayo" ? "[maps]" : "[mz]";
    assert.ok(windows.includes(`resources\\${groupPath}\\${resource}`), resource);
  }
  assert.match(windows, /^setlocal EnableExtensions EnableDelayedExpansion$/m);
  assert.match(windows, /if exist "%%~D\\" \(\s*set \/a BOB74_COUNT\+=1/s);
  assert.match(windows, /resources\\\[maps\]\\mz_cayo/);
  assert.match(windows, /fxmanifest\.lua/);
  assert.match(linux, /cfg\/resources\.cfg/);
  assert.match(linux, /resources\/\[maps\]\/mz_cayo/);
  assert.match(linux, /resources\/\[mz\]\/\$resource\/fxmanifest\.lua/);
});

test("distribution-only resources are explicit and fail closed when unavailable", () => {
  const expected = [
    "mz_status",
    "mz_medical",
    "mz_world_editor",
    "mz_police",
    "mz_prison",
    "mz_dispatch",
    "mz_furniture",
    "mz_computer",
    "mz_transport",
    "mz_multichar",
    "mz_warns",
    "mz_feedback",
  ];

  for (const name of ["tools/install_deps.bat", "tools/install_deps.sh"]) {
    const script = read(name);
    assert.match(script, /MZ_RESOURCE_SOURCE_ROOT/);
    assert.match(script, /distribuicao MZ esta incompleta/);
    for (const resource of expected) {
      assert.match(script, new RegExp(`sync_mz_resource\\s+["']?${resource}["']?\\s+["']{2}`));
    }
  }
});

test("boot order respects the current identity runtime dependencies", () => {
  const position = (resource) => bootResources.indexOf(resource);

  assert.ok(position("mz_core") < position("mz_warns"));
  assert.ok(position("mz_warns") < position("mz_admin"));
  assert.ok(position("mz_core") < position("mz_computer"));
  assert.ok(position("mz_inventory") < position("mz_computer"));
  assert.ok(position("mz_world_editor") < position("mz_computer"));
  assert.ok(position("mz_world_editor") < position("mz_phone"));
  assert.ok(position("mz_inventory") < position("mz_fuel"));
  assert.ok(position("mz_fuel") < position("mz_garagem"));
  assert.ok(position("mz_garagem") < position("mz_prison"));
  assert.ok(position("mz_prison") < position("mz_dispatch"));
  assert.ok(position("mz_dispatch") < position("mz_police"));
  assert.ok(position("mz_police") < position("mz_medical"));
  assert.ok(position("mz_target") < position("mz_police"));
  assert.ok(position("mz_phone") < position("mz_transport"));
  assert.ok(position("mz_creator") < position("mz_multichar"));
});

test("Bob74 and mz_cayo start once in deterministic dependency order", () => {
  assert.equal(bootLines.filter((name) => name === "bob74_ipl").length, 1);
  assert.equal(bootLines.filter((name) => name === "mz_cayo").length, 1);
  assert.ok(bootLines.indexOf("bob74_ipl") < bootLines.indexOf("mz_cayo"));
  assert.ok(bootLines.indexOf("mz_settings") < bootLines.indexOf("mz_cayo"));
  assert.ok(bootLines.indexOf("bob74_ipl") < bootLines.indexOf("mz_settings"));
  assert.ok(bootLines.indexOf("mz_settings") < bootLines.indexOf("mz_interiors"));
  assert.ok(bootLines.indexOf("mz_interiors") < bootLines.indexOf("mz_houses"));

  for (const name of ["tools/install_deps.bat", "tools/install_deps.sh"]) {
    const script = read(name);
    assert.match(script, /bob74_ipl/);
    assert.match(script, /2\.7\.1/);
    assert.match(script, /037dd6c71f6894be771e9343008223d7a3305d79/);
    assert.match(script, /github\.com\/Bob74\/bob74_ipl/);
  }

  for (const name of ["tools/check_structure.bat", "tools/check_structure.sh"]) {
    assert.match(read(name), /bob74_ipl/);
    assert.match(read(name), /2\.7\.1/);
    assert.match(read(name), /resources[\\/]\[maps\][\\/]mz_cayo/);
  }

  for (const name of ["tools/install_deps.bat", "tools/install_deps.sh"]) {
    const script = read(name);
    assert.match(script, /resources[\\/]\[maps\][\\/]bob74_ipl/);
    assert.match(script, /resources[\\/]\[standalone\][\\/]bob74_ipl/);
    assert.match(script, /sync_maps_resource[" ]+["']?mz_cayo/);
  }
});

test("maps provisioning rejects duplicate or modified legacy installs", () => {
  const windowsInstaller = read("tools/install_deps.bat");
  const linuxInstaller = read("tools/install_deps.sh");

  assert.match(windowsInstaller, /resources\\\[standalone\]\\bob74_ipl/);
  assert.match(windowsInstaller, /resources\\\[maps\]\\bob74_ipl/);
  assert.match(windowsInstaller, /status --porcelain --untracked-files/);
  assert.match(windowsInstaller, /move "!BOB74_PATH!" "!BOB74_TARGET!"/);
  assert.match(linuxInstaller, /resources\/\[standalone\]\/bob74_ipl/);
  assert.match(linuxInstaller, /resources\/\[maps\]\/bob74_ipl/);
  assert.match(linuxInstaller, /status --porcelain --untracked-files=all/);
  assert.match(linuxInstaller, /mv -- "\$existing" "\$target"/);
  assert.match(windowsInstaller, /Foram encontradas varias instalacoes de bob74_ipl/);
  assert.match(linuxInstaller, /Foram encontradas várias instalações de bob74_ipl/);
  assert.match(read("tools/check_structure.bat"), /MISSING\/CONFLICT.*mz_cayo/);
  assert.match(read("tools/check_structure.sh"), /MISSING\/CONFLICT.*mz_cayo/);
});

test("Cayo ownership is fully extracted from mz_settings", () => {
  const manifest = readRepositoryFile("mz_settings/fxmanifest.lua");
  const config = readRepositoryFile("mz_settings/config/ipl_exceptions.lua");
  const loader = readRepositoryFile("mz_settings/client/ipl_exceptions.lua");
  const cayoManifest = readRepositoryFile("mz_cayo/fxmanifest.lua");
  const cayoConfig = readRepositoryFile("mz_cayo/config.lua");
  const cayoData = readRepositoryFile("mz_cayo/client/ipl_data.lua");

  assert.doesNotMatch(manifest, /ipl_catalog|ipl_builds\.lua|client\/ipl\.lua|ipl_cayo\.lua/);
  assert.match(manifest, /client\/ipl_exceptions\.lua/);
  assert.doesNotMatch(config, /cayo_perico|cayoRadar|h4_|island_/);
  assert.doesNotMatch(loader, /Cayo|cayo|RadarAsInterior|RadarAsExterior/);
  assert.match(loader, /LoadMpDlcMaps\(\)/);
  assert.match(loader, /EnableMpDlcMaps\(true\)/);
  for (const ipl of [
    "h4_ch2_mansion_final",
    "h4_clubposter_keinemusik",
    "h4_clubposter_moodymann",
    "h4_clubposter_palmstraxx",
  ]) {
    assert.doesNotMatch(cayoData, new RegExp(`['"]${ipl}['"]`), `${ipl} must remain Bob-owned`);
  }
  for (const ipl of [
    "bnkheist_apt_norm",
    "FIBlobbyfake",
    "v_rockclub",
    "TrevorsMP",
    "cs3_05_water_grp2",
  ]) {
    assert.ok(config.includes(ipl), `${ipl} uncovered exception missing`);
  }
  assert.match(cayoManifest, /client\/ipl_data\.lua/);
  assert.match(cayoManifest, /client\/main\.lua/);
  assert.match(cayoManifest, /dependency 'mz_settings'/);
  assert.match(cayoConfig, /activeVariant = 'secured'/);
  assert.match(cayoData, /gameBuild = 2189/);
  assert.doesNotMatch(loader, /RegisterCommand\('mzsettings_ipllist'/);
});

test("Cayo IPL sequence matches the audited pre-extraction data", () => {
  const cayoData = readRepositoryFile("mz_cayo/client/ipl_data.lua");
  const ids = [...cayoData.matchAll(/'((?:h4_|island_)[^']*)'/g)]
    .map((match) => match[1]);
  const fingerprint = crypto.createHash("sha256").update(ids.join("\n")).digest("hex");

  assert.equal(
    fingerprint,
    "ea787516bff65260d36dd3690180d4c61bbbcd02c2fcba682f95734c8c5886ec",
  );
  assert.ok(ids.length > 0);
});
