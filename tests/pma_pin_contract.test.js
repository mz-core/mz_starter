const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const root = path.resolve(__dirname, "..");
const read = (name) => fs.readFileSync(path.join(root, name), "utf8");
const commit = "3379e45dcf7aaa8b7651317190c108095a121553";

test("Linux and Windows installers pin the same official pma revision", () => {
  for (const script of [read("tools/install_deps.sh"), read("tools/install_deps.bat")]) {
    assert.match(script, /https:\/\/github\.com\/AvarianKnight\/pma-voice\.git/);
    assert.match(script, /v7\.0\.0/);
    assert.ok(script.includes(commit));
    assert.doesNotMatch(script, /sync_git_repo ["']?pma-voice/);
  }
});

test("installers verify and preserve the server-only export patch", () => {
  const patch = read("patches/pma-voice-server-authoritative-calls.patch");
  assert.match(patch, /^-RegisterNetEvent\('pma-voice:setPlayerCall'/m);
  assert.doesNotMatch(patch, /^\+RegisterNetEvent\('pma-voice:setPlayerCall'/m);
  assert.match(patch, /exports\('setPlayerCall', setPlayerCall\)/);

  for (const script of [read("tools/install_deps.sh"), read("tools/install_deps.bat")]) {
    assert.match(script, /apply --reverse --check/);
    assert.match(script, /handler client-authoritative/);
    assert.match(script, /alteracoes locais inesperadas/);
  }
});

test("boot order and voice convars match the pinned provider contract", () => {
  const resources = read("cfg/resources.cfg");
  assert.ok(resources.indexOf("ensure pma-voice") < resources.indexOf("ensure mz_phone"));
  assert.ok(resources.indexOf("ensure pma-voice") < resources.indexOf("ensure mz_radio"));

  const onesync = read("cfg/onesync.cfg");
  assert.match(onesync, /set onesync on/);
  assert.match(onesync, /setr voice_enableCalls 1/);
  assert.match(onesync, /setr voice_enableRadios 1/);
  assert.match(onesync, /setr voice_useSendingRangeOnly true/);
});
