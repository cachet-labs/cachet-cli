#!/usr/bin/env node
'use strict';

const path = require('path');
const { spawnSync } = require('child_process');

const binary = process.platform === 'win32' ? 'cachet.exe' : 'cachet';
const binaryPath = path.join(__dirname, binary);

const { status, error } = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: false,
});

if (error) {
  if (error.code === 'ENOENT') {
    process.stderr.write(
      'cachet binary not found.\nTry reinstalling: npm install -g cachet-cli\n'
    );
  } else {
    process.stderr.write(`cachet: ${error.message}\n`);
  }
  process.exit(1);
}

process.exit(status ?? 0);
