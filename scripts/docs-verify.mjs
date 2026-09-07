#!/usr/bin/env node
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const required = [
  'README.md', 'README.en.md', 'README.es.md', 'docs/README.md',
  'docs/getting-started/installation.md', 'docs/product/overview.md',
  'docs/product/visual-tour.md', 'docs/assets/screenshots/manifest.json',
  'docs/assets/screenshots/workspace-overview.png',
  'docs/assets/demos/direct-workflow.webm',
];
const errors = [];
for (const file of required) if (!existsSync(path.join(root, file))) errors.push(`missing required file: ${file}`);

const markdownFiles = [];
function walk(dir) {
  if (!existsSync(dir)) return;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || entry.name === 'node_modules' || entry.name === 'DEV' || entry.name === 'superpowers' || entry.name === 'community-preview' || entry.name === 'engineering' || entry.name === 'design') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.isFile() && full.endsWith('.md')) markdownFiles.push(full);
  }
}
for (const file of ['README.md', 'README.en.md', 'README.es.md']) markdownFiles.push(path.join(root, file));
walk(path.join(root, 'docs'));
const linkPattern = /!?(?:\[[^\]]*\])\(([^)]+)\)/g;
for (const file of markdownFiles) {
  const text = readFileSync(file, 'utf8');
  let match;
  while ((match = linkPattern.exec(text))) {
    const target = match[1].trim().split(/\s+['"]?/)[0];
    if (!target || /^(?:https?:|mailto:|file:|#)/i.test(target)) continue;
    const clean = target.split('#')[0].replace(/:\d+$/, '');
    const resolved = path.resolve(path.dirname(file), clean);
    if (!existsSync(resolved)) errors.push(`broken local link: ${path.relative(root, file)} -> ${target}`);
    if (/^(?:\.\.\/)*\.?(?:DEV|\.omx)(?:\/|$)/i.test(clean)) errors.push(`public doc links internal path: ${path.relative(root, file)} -> ${target}`);
  }
}
const manifestPath = path.join(root, 'docs/assets/screenshots/manifest.json');
if (existsSync(manifestPath)) {
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  const items = Array.isArray(manifest) ? manifest : manifest.captures || [];
  const requiredVisualIds = Array.from({ length: 11 }, (_, index) => `VIS-${String(index + 1).padStart(3, '0')}`);
  const itemById = new Map();
  for (const item of items) {
    if (!item.id) errors.push('visual manifest item has no id');
    else if (itemById.has(item.id)) errors.push(`visual manifest duplicate id: ${item.id}`);
    else itemById.set(item.id, item);
    if (item.status !== 'PASS') errors.push(`visual manifest item is not PASS: ${item.id || '(unknown)'}`);
    for (const field of ['surface', 'scenario', 'viewport', 'capture_command', 'source_sha']) {
      if (!item[field]) errors.push(`visual manifest item ${item.id || '(unknown)'} missing ${field}`);
    }
    if (item.source_sha && !/^[0-9a-f]{40}$/i.test(item.source_sha)) {
      errors.push(`visual manifest item has invalid source_sha: ${item.id || '(unknown)'}`);
    }
  }
  for (const id of requiredVisualIds) if (!itemById.has(id)) errors.push(`visual manifest missing required capture: ${id}`);
  for (const item of items) {
    if (!item.file) errors.push('visual manifest item has no file');
    else if (!existsSync(path.join(root, 'docs/assets/screenshots', item.file))) errors.push(`manifest file missing: ${item.file}`);
  }

  if (manifest.source_sha && !/^[0-9a-f]{40}$/i.test(manifest.source_sha)) {
    errors.push('visual manifest has invalid source_sha');
  }
  if (manifest.source_sha) {
    const visualPaths = ['web/src', 'web/public', 'cmd/nexus-desktop', 'internal/desktop', 'web/scripts'];
    const diff = spawnSync('git', ['diff', '--name-only', `${manifest.source_sha}..HEAD`, '--', ...visualPaths], {
      cwd: root,
      encoding: 'utf8',
    });
    if (diff.status === 0 && (diff.stdout || '').trim()) {
      errors.push(`visual manifest is stale after visual changes since ${manifest.source_sha}: ${(diff.stdout || '').trim().replace(/\n/g, ', ')}`);
    }
  }
}
for (const file of ['README.md', 'README.en.md', 'README.es.md']) {
  const text = readFileSync(path.join(root, file), 'utf8');
  for (const phrase of ['TL;DR', 'docs/README.md', 'docs/assets/screenshots/workspace-overview.png']) {
    if (!text.includes(phrase)) errors.push(`${file} missing required phrase: ${phrase}`);
  }
}
if (errors.length) {
  console.error(errors.map((error) => `✗ ${error}`).join('\n'));
  process.exitCode = 1;
} else {
  console.log(`docs verification passed: ${markdownFiles.length} Markdown files, ${required.length} required assets/files`);
}
