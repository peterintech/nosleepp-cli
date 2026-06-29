const fs = require("node:fs");
const path = require("node:path");

const supportedTargets = {
  "win32-x64": "nosleepp-win32-x64.exe",
  "darwin-arm64": "nosleepp-darwin-arm64",
  "darwin-x64": "nosleepp-darwin-x64"
};

function resolveBinary(platform, arch, rootDir) {
  const target = `${platform}-${arch}`;
  const fileName = supportedTargets[target];
  if (!fileName) {
    return {
      ok: false,
      error: `Unsupported platform: ${target}. Supported targets: ${Object.keys(supportedTargets).join(", ")}.`
    };
  }

  const binaryPath = path.join(rootDir, "bin", fileName);
  if (!fs.existsSync(binaryPath)) {
    return {
      ok: false,
      error: `Missing nosleepp binary for ${target}: ${binaryPath}. Rebuild the npm package before publishing.`
    };
  }

  return { ok: true, path: binaryPath };
}

module.exports = {
  resolveBinary,
  supportedTargets
};
