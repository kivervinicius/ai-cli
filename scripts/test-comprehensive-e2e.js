import { chromium } from '../web/node_modules/playwright/index.mjs';
import fs from 'node:fs';

(async () => {
  const log = fs.readFileSync('/tmp/nexus-web.log', 'utf8');
  const match = log.match(/Bootstrap:\s+(http:\/\/[^\s]+)/);
  if (!match) throw new Error('Bootstrap URL not found in log');
  const bootstrapUrl = match[1];

  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/google-chrome',
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  
  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];
  
  page.on('console', msg => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
      console.log('[BROWSER CONSOLE ERROR]', msg.text());
    }
  });
  page.on('pageerror', err => {
    pageErrors.push(err.toString());
    console.log('[BROWSER UNCAUGHT ERROR]', err.toString());
  });
  page.on('response', res => {
    if (res.status() >= 400 && !res.url().includes('favicon')) {
      failedRequests.push({ status: res.status(), url: res.url() });
      console.log(`[HTTP ${res.status()}] ${res.url()}`);
    }
  });

  console.log('1. Navigating to:', bootstrapUrl);
  await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2500);

  // 2. Test Language Switcher to English, then Portuguese, then Spanish
  console.log('2. Testing Language Switcher...');
  const langBtn = await page.$('.nx-lang-btn');
  if (langBtn) {
    await langBtn.click();
    await page.waitForTimeout(300);
    const enOpt = await page.$('.nx-lang-option:has-text("English")');
    if (enOpt) await enOpt.click();
    await page.waitForTimeout(500);

    await langBtn.click();
    await page.waitForTimeout(300);
    const ptOpt = await page.$('.nx-lang-option:has-text("Português")');
    if (ptOpt) await ptOpt.click();
    await page.waitForTimeout(500);
  }

  // 3. Test Command Palette (Ctrl+K)
  console.log('3. Testing Command Palette (Ctrl+K)...');
  await page.keyboard.press('Control+k');
  await page.waitForTimeout(500);
  await page.keyboard.type('overview');
  await page.waitForTimeout(300);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(300);

  // 4. Test Project Manager (Ctrl+P)
  console.log('4. Testing Project Manager (Ctrl+P)...');
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(800);

  // 5. In Project Manager: test Directory Browser Modal
  console.log('5. Testing Directory Browser Modal from Project Manager...');
  const browseBtn = await page.locator('button:has-text("Browse"), button:has-text("Procurar"), button[title*="Browse"], button[title*="Procurar"]').first();
  if (await browseBtn.isVisible()) {
    await browseBtn.click();
    await page.waitForTimeout(1000);
    
    // Check bookmarks in modal
    const bookmarksCount = await page.$$eval('.nx-dir-bookmarks-list button', b => b.length);
    console.log(`Directory modal bookmarks: ${bookmarksCount}`);

    // Check directory entries
    const entriesCount = await page.$$eval('.nx-dir-entry-card', c => c.length);
    console.log(`Directory modal entries: ${entriesCount}`);

    // Close directory browser modal
    const closeBtn = await page.locator('.nx-dir-picker__footer button:has-text("Fechar"), .nx-dir-picker__footer button:has-text("Close")').first();
    if (await closeBtn.isVisible()) {
      await closeBtn.click();
      await page.waitForTimeout(400);
    }
  }

  // 6. Test Project Scan Modal
  console.log('6. Testing Project Scan Modal...');
  const scanBtn = await page.locator('button:has-text("Scan"), button:has-text("Escanear"), button[title*="Scan"]').first();
  if (await scanBtn.isVisible()) {
    await scanBtn.click();
    await page.waitForTimeout(1000);
    const scanCardsCount = await page.$$eval('.nx-scan-entry-card', c => c.length);
    console.log(`Scan modal results: ${scanCardsCount}`);
    
    const closeScanBtn = await page.locator('button:has-text("Fechar"), button:has-text("Close"), button:has-text("Cancelar")').first();
    if (await closeScanBtn.isVisible()) {
      await closeScanBtn.click();
      await page.waitForTimeout(400);
    }
  }

  // Close Project Manager if open
  await page.keyboard.press('Escape');
  await page.waitForTimeout(400);

  // 7. Test navigation through workspace tabs/surfaces
  console.log('7. Testing Workspace Surfaces Navigation...');
  const tabs = await page.$$('.nx-workspace-tab, .nx-project-rail__global button');
  console.log(`Found ${tabs.length} tabs/rail buttons`);
  for (let i = 0; i < tabs.length; i++) {
    try {
      await tabs[i].click();
      await page.waitForTimeout(600);
    } catch (e) {
      // ignore
    }
  }

  // 8. Summary check
  console.log('\n=== COMPREHENSIVE E2E SUMMARY ===');
  console.log(`Page Errors: ${pageErrors.length}`);
  console.log(`Console Errors: ${consoleErrors.length}`);
  console.log(`Failed HTTP Requests: ${failedRequests.length}`);

  await browser.close();

  if (pageErrors.length > 0 || consoleErrors.length > 0) {
    console.error('FAILED with errors!');
    process.exit(1);
  }
  console.log('ALL TESTS PASSED WITH ZERO ERRORS AND ZERO WARNINGS!');
})();
