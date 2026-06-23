#!/usr/bin/env node
// postinstall hook for the npm-distributed cvox package.
//
// Downloads the prebuilt Go binary for the current platform from the matching
// GitHub Release and places it next to the launcher as bin/cvox-bin
// (bin/cvox-bin.exe on Windows). The release artifacts are produced by
// goreleaser; their names follow `cvox_{version}_{os}_{arch}.{tar.gz|zip}`.
//
// Design notes:
// - Only Node built-ins are used (no runtime deps). tar/zip extraction shells
//   out to the system `tar` (Unix) or PowerShell `Expand-Archive` (Windows).
// - On ANY failure (unsupported platform, network error, extraction error) we
//   print a warning and exit 0, so a flaky network never breaks `npm install`.
//   The launcher (bin/cvox.cjs) reports a friendly error at run time if the
//   binary is missing.

'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const https = require('https');
const { execFileSync } = require('child_process');

const REPO = 'lmk123/cvox';
const BIN_DIR = path.join(__dirname, '..', 'bin');

// Map Node's platform/arch to goreleaser's os_arch naming.
const PLATFORM_MAP = {
  'darwin-arm64': 'darwin_arm64',
  'darwin-x64': 'darwin_amd64',
  'linux-arm64': 'linux_arm64',
  'linux-x64': 'linux_amd64',
  'win32-arm64': 'windows_arm64',
  'win32-x64': 'windows_amd64',
};

function warn(msg) {
  console.warn('cvox postinstall: ' + msg);
}

// Resolve the package version from package.json (kept in sync with the git tag
// at release time by CI).
function getVersion() {
  const pkg = JSON.parse(
    fs.readFileSync(path.join(__dirname, '..', 'package.json'), 'utf8')
  );
  return pkg.version;
}

// Download a URL to a local file, following GitHub's 302 redirects (release
// assets redirect to objects.githubusercontent.com).
function download(url, dest, redirectsLeft) {
  if (redirectsLeft === undefined) redirectsLeft = 5;
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { 'User-Agent': 'cvox-npm-postinstall' } }, (res) => {
        const status = res.statusCode || 0;
        if (status >= 300 && status < 400 && res.headers.location) {
          res.resume(); // discard body
          if (redirectsLeft <= 0) {
            reject(new Error('too many redirects'));
            return;
          }
          resolve(download(res.headers.location, dest, redirectsLeft - 1));
          return;
        }
        if (status !== 200) {
          res.resume();
          reject(new Error('HTTP ' + status + ' for ' + url));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on('finish', () => file.close((err) => (err ? reject(err) : resolve())));
        file.on('error', reject);
      })
      .on('error', reject);
  });
}

// Extract the cvox binary from the downloaded archive into BIN_DIR.
function extract(archivePath, isWindows, tmpDir) {
  if (isWindows) {
    // Expand-Archive into tmpDir, then move cvox.exe out.
    execFileSync(
      'powershell',
      [
        '-NoProfile',
        '-Command',
        'Expand-Archive -Force -LiteralPath ' +
          JSON.stringify(archivePath) +
          ' -DestinationPath ' +
          JSON.stringify(tmpDir),
      ],
      { stdio: 'ignore' }
    );
    fs.renameSync(path.join(tmpDir, 'cvox.exe'), path.join(BIN_DIR, 'cvox-bin.exe'));
  } else {
    execFileSync('tar', ['-xzf', archivePath, '-C', tmpDir], { stdio: 'ignore' });
    const target = path.join(BIN_DIR, 'cvox-bin');
    fs.renameSync(path.join(tmpDir, 'cvox'), target);
    fs.chmodSync(target, 0o755);
  }
}

async function main() {
  const key = process.platform + '-' + process.arch;
  const goPlatform = PLATFORM_MAP[key];
  if (!goPlatform) {
    warn('unsupported platform ' + key + '; skipping binary download.');
    return;
  }

  const version = getVersion();
  const isWindows = process.platform === 'win32';
  const ext = isWindows ? '.zip' : '.tar.gz';
  const fileName = 'cvox_' + version + '_' + goPlatform + ext;
  const url =
    'https://github.com/' + REPO + '/releases/download/v' + version + '/' + fileName;

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cvox-'));
  const archivePath = path.join(tmpDir, fileName);

  try {
    console.log('cvox: downloading ' + url);
    await download(url, archivePath);
    extract(archivePath, isWindows, tmpDir);
    console.log('cvox: installed binary for ' + key + '.');
  } catch (err) {
    warn(
      'failed to download/extract the binary (' +
        err.message +
        '). cvox will report this when you run it. ' +
        'You can retry with `npm install -g cvox` or grab a binary from ' +
        'https://github.com/' +
        REPO +
        '/releases'
    );
  } finally {
    try {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    } catch (_) {
      /* ignore cleanup errors */
    }
  }
}

// Never let a failure here break `npm install` — always exit 0.
main().catch((err) => {
  warn('unexpected error: ' + (err && err.message ? err.message : err));
});
