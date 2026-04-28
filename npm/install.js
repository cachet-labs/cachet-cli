'use strict';

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const { execSync } = require('child_process');

const REPO = 'cachet-labs/cachet-cli';
const BIN_DIR = path.join(__dirname, 'bin');

function getPlatformInfo() {
  const platformMap = { win32: 'windows', darwin: 'darwin', linux: 'linux' };
  const archMap = { x64: 'amd64', arm64: 'arm64' };

  const goos = platformMap[process.platform];
  const goarch = archMap[process.arch];

  if (!goos) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }
  if (!goarch) {
    throw new Error(
      `Unsupported architecture: ${process.arch}. Only x64 and arm64 are supported.`
    );
  }

  return {
    goos,
    goarch,
    ext: goos === 'windows' ? '.zip' : '.tar.gz',
    binary: goos === 'windows' ? 'cachet.exe' : 'cachet',
  };
}

function getVersion() {
  return process.env.npm_package_version || require('./package.json').version;
}

function fetchTo(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      https
        .get(url, { headers: { 'User-Agent': 'cachet-cli-installer' } }, (res) => {
          if (res.statusCode === 301 || res.statusCode === 302) {
            follow(res.headers.location);
            return;
          }
          if (res.statusCode !== 200) {
            reject(new Error(`HTTP ${res.statusCode} for ${url}`));
            return;
          }
          const file = fs.createWriteStream(dest);
          res.pipe(file);
          file.on('finish', () => file.close(resolve));
          file.on('error', (err) => {
            fs.unlink(dest, () => {});
            reject(err);
          });
        })
        .on('error', reject);
    };
    follow(url);
  });
}

function fetchText(url) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      https
        .get(url, { headers: { 'User-Agent': 'cachet-cli-installer' } }, (res) => {
          if (res.statusCode === 301 || res.statusCode === 302) {
            follow(res.headers.location);
            return;
          }
          if (res.statusCode !== 200) {
            reject(new Error(`HTTP ${res.statusCode} for ${url}`));
            return;
          }
          let body = '';
          res.on('data', (chunk) => (body += chunk));
          res.on('end', () => resolve(body));
        })
        .on('error', reject);
    };
    follow(url);
  });
}

async function verifyChecksum(archivePath, archiveName, version) {
  const checksumUrl = `https://github.com/${REPO}/releases/download/v${version}/checksums.txt`;
  const text = await fetchText(checksumUrl);

  const line = text.split('\n').find((l) => l.trimEnd().endsWith(archiveName));
  if (!line) {
    throw new Error(`No checksum entry found for ${archiveName} in checksums.txt`);
  }

  const expected = line.split(/\s+/)[0];
  const actual = crypto.createHash('sha256').update(fs.readFileSync(archivePath)).digest('hex');

  if (actual !== expected) {
    throw new Error(
      `Checksum mismatch for ${archiveName}\n  expected: ${expected}\n  got:      ${actual}`
    );
  }
}

function extract(archivePath, binaryName, destDir, ext) {
  if (ext === '.tar.gz') {
    execSync(`tar -xzf "${archivePath}" -C "${destDir}" ${binaryName}`, { stdio: 'pipe' });
    return;
  }
  // Windows zip: try built-in tar (Win10 1803+), fall back to PowerShell
  try {
    execSync(`tar -xf "${archivePath}" -C "${destDir}" ${binaryName}`, { stdio: 'pipe' });
  } catch {
    execSync(
      `powershell -NoProfile -NonInteractive -Command ` +
        `"Expand-Archive -Force -LiteralPath '${archivePath}' -DestinationPath '${destDir}'"`,
      { stdio: 'pipe' }
    );
  }
}

async function main() {
  const { goos, goarch, ext, binary } = getPlatformInfo();
  const version = getVersion();
  const archiveName = `cachet_${version}_${goos}_${goarch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;
  const tmp = path.join(os.tmpdir(), archiveName);
  const dest = path.join(BIN_DIR, binary);

  if (fs.existsSync(dest)) {
    console.log(`cachet already installed.`);
    return;
  }

  console.log(`Downloading cachet v${version} (${goos}/${goarch})...`);
  await fetchTo(url, tmp);

  console.log(`Verifying checksum...`);
  await verifyChecksum(tmp, archiveName, version);

  fs.mkdirSync(BIN_DIR, { recursive: true });
  extract(tmp, binary, BIN_DIR, ext);
  fs.unlinkSync(tmp);

  if (goos !== 'windows') {
    fs.chmodSync(dest, 0o755);
  }

  console.log(`cachet v${version} installed successfully.`);
}

main().catch((err) => {
  console.error(`\nFailed to install cachet: ${err.message}`);
  console.error(`Download manually: https://github.com/${REPO}/releases\n`);
  process.exit(1);
});
