#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const { resolveBinary } = require("./lib/resolve-binary");

const resolved = resolveBinary(process.platform, process.arch, __dirname);
if (!resolved.ok) {
  console.error(resolved.error);
  process.exit(1);
}

const result = spawnSync(resolved.path, process.argv.slice(2), {
  stdio: "inherit"
});

if (result.error) {
  console.error(`Failed to run nosleepp: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 1);
