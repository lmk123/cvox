#!/usr/bin/env node
// CLI launcher for the npm-distributed cvox package.
//
// The actual cvox CLI is a Go binary downloaded by scripts/npm-postinstall.cjs
// into this same directory (bin/cvox-bin, or bin/cvox-bin.exe on Windows).
// This launcher locates that binary and execs it, passing through argv, stdin,
// stdout/stderr, and the exit code unchanged.
//
// Why a JS launcher instead of pointing "bin" straight at the binary: at publish
// / `npm pack` time the platform binary does not exist yet (it is fetched in
// postinstall), so "bin" must point at a file that always exists. The launcher
// also turns a missing binary into a friendly message instead of a raw ENOENT.

'use strict';

const path = require('path');
const fs = require('fs');
const { spawnSync } = require('child_process');

const binName = process.platform === 'win32' ? 'cvox-bin.exe' : 'cvox-bin';
const binPath = path.join(__dirname, binName);

if (!fs.existsSync(binPath)) {
  console.error(
    'cvox: the platform binary was not found at ' + binPath + '.\n' +
      'This usually means the postinstall download failed (e.g. no network ' +
      'access to github.com when `npm install` ran).\n' +
      'Try reinstalling: `npm install -g cvox`, or download a binary manually ' +
      'from https://github.com/lmk123/cvox/releases'
  );
  process.exit(1);
}

// stdio: 'inherit' is essential — `cvox notify` reads the hook event JSON from
// stdin, so the child must inherit the parent's stdin/stdout/stderr directly.
const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error('cvox: failed to launch binary: ' + result.error.message);
  process.exit(1);
}

// Propagate the child's exit code; if it was killed by a signal, mimic the
// conventional 128 + signal-number exit status.
if (result.signal) {
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
