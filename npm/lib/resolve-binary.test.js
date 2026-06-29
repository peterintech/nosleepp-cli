const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { resolveBinary, supportedTargets } = require("./resolve-binary");

test("resolves supported binaries", () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "nosleepp-"));
  fs.mkdirSync(path.join(tmp, "bin"));
  fs.writeFileSync(path.join(tmp, "bin", supportedTargets["darwin-arm64"]), "");

  const result = resolveBinary("darwin", "arm64", tmp);

  assert.equal(result.ok, true);
  assert.equal(result.path, path.join(tmp, "bin", "nosleepp-darwin-arm64"));
});

test("rejects unsupported platforms", () => {
  const result = resolveBinary("linux", "x64", __dirname);

  assert.equal(result.ok, false);
  assert.match(result.error, /Unsupported platform: linux-x64/);
  assert.match(result.error, /win32-x64/);
  assert.match(result.error, /darwin-arm64/);
});

test("reports missing binary for supported platform", () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "nosleepp-"));

  const result = resolveBinary("win32", "x64", tmp);

  assert.equal(result.ok, false);
  assert.match(result.error, /Missing nosleepp binary for win32-x64/);
});
