#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webDir, '..');
const binPath = path.resolve(repoRoot, 'nexus');
const screenshotDir = path.resolve(repoRoot, '.tempmediaStorage');

if (!existsSync(screenshotDir)) {
  mkdirSync(screenshotDir, { recursive: true });
}

console.log('=== Nexus E2E Hardening & Responsiveness Verification ===');
console.log('1. Checking nexus binary...');
if (!existsSync(binPath)) {
  console.log('Building nexus binary with make build...');
  const buildRes = spawnSync('make', ['build'], { cwd: repoRoot, stdio: 'inherit' });
  assert.equal(buildRes.status, 0, 'make build must succeed');
}

async function terminateProcess(proc) {
  if (!proc || !proc.pid) return;
  return new Promise((resolve) => {
    try {
      if (process.platform === 'win32') {
        spawn('taskkill', ['/pid', String(proc.pid), '/T', '/F']);
        resolve();
      } else {
        process.kill(-proc.pid, 'SIGTERM');
        const timeout = setTimeout(() => {
          try {
            process.kill(-proc.pid, 'SIGKILL');
          } catch (_) {}
          resolve();
        }, 1200);
        proc.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      }
    } catch (_) {
      resolve();
    }
  });
}

async function main() {
  console.log('2. Starting Nexus Web Server on ephemeral port (--port 0)...');
  const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
    cwd: repoRoot,
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  let bootstrapUrl = '';
  const portPromise = new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error('Timed out waiting for server bootstrap URL in stdout (12s)'));
    }, 12000);

    let output = '';
    server.stdout.on('data', (chunk) => {
      output += chunk.toString();
      const match = output.match(/Bootstrap:\s*(http:\/\/127\.0\.0\.1:\d+\/\?token=[a-f0-9]+)/i);
      if (match) {
        clearTimeout(timer);
        bootstrapUrl = match[1];
        resolve(bootstrapUrl);
      }
    });

    server.stderr.on('data', (chunk) => {
      output += chunk.toString();
    });

    server.on('exit', (code) => {
      clearTimeout(timer);
      reject(new Error(`Server exited prematurely with code ${code}. Output:\n${output}`));
    });
  });

  try {
    await portPromise;
    console.log(`Server bound successfully! Bootstrap URL: ${bootstrapUrl}`);

    console.log('3. Launching headless browser...');
    const browser = await chromium.launch({
      executablePath: '/usr/bin/google-chrome',
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
    });

    const viewports = [
      { width: 320, height: 568, name: 'mobile-se' },
      { width: 390, height: 844, name: 'mobile-iphone' },
      { width: 768, height: 1024, name: 'tablet' },
      { width: 1024, height: 768, name: 'laptop-compact' },
      { width: 1280, height: 800, name: 'laptop-critical' },
      { width: 1440, height: 900, name: 'desktop' },
    ];

    const context = await browser.newContext();
    const page = await context.newPage();

    console.log(`Navigating to ${bootstrapUrl}...`);
    await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });
    await page.waitForTimeout(1000);

    console.log('4. Testing Breakpoints and Create Menu Button...');
    for (const vp of viewports) {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.waitForTimeout(400);

      const createBtn = page.locator('[data-testid="topbar-create-menu-btn"]');
      const isVisible = await createBtn.isVisible();
      assert.equal(isVisible, true, `Create menu button must be visible at ${vp.width}x${vp.height}`);

      const box = await createBtn.boundingBox();
      assert.ok(box, `Create menu button must have bounding box at ${vp.width}px`);
      assert.ok(box.width >= 30, `Create menu button width (${box.width}) must be >= 30 at ${vp.width}px`);
      assert.ok(box.height >= 26, `Create menu button height (${box.height}) must be >= 26 at ${vp.width}px`);

      // Verifica ausência física de sobreposição (elementFromPoint)
      const elementAtPoint = await page.evaluate(
        ({ x, y }) => {
          const el = document.elementFromPoint(x, y);
          return Boolean(el && el.closest('[data-testid="topbar-create-menu-btn"]'));
        },
        { x: box.x + box.width / 2, y: box.y + box.height / 2 }
      );
      assert.equal(elementAtPoint, true, `Create menu button must be topmost element at ${vp.width}px`);

      const shotPath = path.join(screenshotDir, `e2e_bp_${vp.width}.png`);
      await page.screenshot({ path: shotPath });
      console.log(`  ✓ Breakpoint ${vp.width}x${vp.height} verified (Zero obstruction, shot: ${shotPath})`);
    }

    console.log('5. Testing Settings Surface, Accordion WAI-ARIA and Density Delta...');
    await page.setViewportSize({ width: 1280, height: 800 });
    const settingsBtn = page.locator('button:has-text("Settings")').first();
    if (await settingsBtn.isVisible()) {
      await settingsBtn.click();
      await page.waitForTimeout(800);
    }

    // Validação do Accordion
    const accordionHeader = page.locator('button[id^="theme-cat-hdr-"]').first();
    assert.equal(await accordionHeader.isVisible(), true, 'Accordion category header must be visible');
    const ariaExpanded = await accordionHeader.getAttribute('aria-expanded');
    assert.ok(ariaExpanded === 'true' || ariaExpanded === 'false', 'aria-expanded must be boolean string');

    // Validação dos Radios com Swatches
    const themeRadio = page.locator('div[role="radio"]').first();
    assert.equal(await themeRadio.isVisible(), true, 'Theme radio must be visible');
    const hasSwatches = await themeRadio.locator('.nx-theme-palette, span[title^="Fundo:"]').count();
    assert.ok(hasSwatches > 0, 'Theme radio must display color palette swatches');

    // Validação Quantitativa de Densidade (Compact vs Comfortable)
    const card = page.locator('.nx-settings-card').first();
    const densityCompBtn = page.locator('button:has-text("Compacta"), button:has-text("Compact")').first();
    const densityComfBtn = page.locator('button:has-text("Confortável"), button:has-text("Comfortable")').first();

    if ((await densityComfBtn.isVisible()) && (await densityCompBtn.isVisible())) {
      await densityComfBtn.click();
      await page.waitForTimeout(300);
      const boxComf = await card.boundingBox();

      await densityCompBtn.click();
      await page.waitForTimeout(300);
      const boxComp = await card.boundingBox();

      if (boxComf && boxComp) {
        console.log(`  Measured Card Height - Comfortable: ${boxComf.height.toFixed(1)}px, Compact: ${boxComp.height.toFixed(1)}px`);
        assert.ok(boxComf.height > boxComp.height, 'Comfortable card height must be strictly greater than Compact');
      }
    }

    const settingsShot = path.join(screenshotDir, 'e2e_settings_accordion.png');
    await page.screenshot({ path: settingsShot });
    console.log(`  ✓ Settings & Accordion verified (shot: ${settingsShot})`);

    await browser.close();
    console.log('✓ All E2E Hardening assertions PASSED successfully!');
  } finally {
    console.log('6. Terminating test server...');
    await terminateProcess(server);
  }
}

main().catch((err) => {
  console.error('E2E Verification FAILED:', err);
  process.exit(1);
});
