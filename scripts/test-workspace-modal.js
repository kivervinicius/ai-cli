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
  
  page.on('console', msg => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  page.on('pageerror', err => pageErrors.push(err.toString()));

  console.log('1. Navigating to:', bootstrapUrl);
  await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2500);

  // Take screenshot of main workspace
  await page.screenshot({ path: '/tmp/nexus-main-workspace.png' });
  console.log('Saved main workspace screenshot to /tmp/nexus-main-workspace.png');

  // Open Project Manager (Ctrl+P)
  console.log('2. Opening Project Manager (Ctrl+P)...');
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(1000);

  // In Project Manager, click 'Browse Folders' or 'Adicionar Projeto'
  console.log('3. Finding Browse Folders / Procurar no SO buttons...');
  const browseBtn = await page.locator('button:has-text("Browse"), button:has-text("Procurar"), button[title*="Browse"], button[title*="Procurar"]').first();
  if (await browseBtn.isVisible()) {
    console.log('Clicking browse button...');
    await browseBtn.click();
    await page.waitForTimeout(1500);
    await page.screenshot({ path: '/tmp/nexus-dir-modal-live.png' });
    console.log('Saved modal screenshot to /tmp/nexus-dir-modal-live.png');
  }

  // Also test clicking the Left Rail "+ Add" button
  const addRailBtn = await page.locator('.nx-project-rail button:has-text("Add"), .nx-project-rail button[aria-label*="Add"], .nx-project-rail button[aria-label*="Adicionar"]').first();
  if (await addRailBtn.isVisible()) {
    console.log('Found Rail Add button');
  }

  console.log('--- Page Errors ---', pageErrors);
  console.log('--- Console Errors ---', consoleErrors);

  await browser.close();
  console.log('FINISHED TEST SUCCESSFULLY!');
})();
