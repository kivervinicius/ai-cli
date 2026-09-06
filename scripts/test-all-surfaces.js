import { chromium } from '../web/node_modules/playwright/index.mjs';
import fs from 'node:fs';

(async () => {
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
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
      console.log('[BROWSER CONSOLE ERROR]', msg.text());
    }
  });
  page.on('pageerror', err => {
    pageErrors.push(err.toString());
    console.log('[BROWSER UNCAUGHT ERROR]', err.toString());
  });

  const log = fs.readFileSync('/tmp/nexus-web.log', 'utf8');
  const match = log.match(/Bootstrap:\s+(http:\/\/[^\s]+)/);
  if (!match) throw new Error('Bootstrap URL not found in log');
  const bootstrapUrl = match[1];

  console.log('1. Authenticating via:', bootstrapUrl);
  await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Click on "Add Project" form if on ProjectHub
  const addProjectBtn = await page.getByRole('button', { name: /Add Project|Adicionar Projeto/i }).first();
  if (await addProjectBtn.isVisible()) {
    console.log('On ProjectHub. Filling project path...');
    const input = await page.locator('input').first();
    await input.fill('/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager');
    await addProjectBtn.click();
    await page.waitForTimeout(2500);
  }

  await page.screenshot({ path: '/tmp/nexus-project-main.png' });
  console.log('Main project view screenshot saved');

  // Let's list all buttons and click through surfaces
  const topButtons = await page.$$('.nx-topbar button, .nx-project-rail button, .nx-taskbar button');
  console.log(`Found ${topButtons.length} toolbar/rail buttons in active workspace`);

  // Open Command Palette
  console.log('Opening Command Palette (Ctrl+K)...');
  await page.keyboard.press('Control+k');
  await page.waitForTimeout(500);
  await page.screenshot({ path: '/tmp/nexus-palette.png' });
  await page.keyboard.press('Escape');
  await page.waitForTimeout(300);

  // Open Project Manager (Ctrl+P)
  console.log('Opening Project Manager (Ctrl+P)...');
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(800);
  await page.screenshot({ path: '/tmp/nexus-project-manager.png' });
  await page.keyboard.press('Escape');
  await page.waitForTimeout(300);

  console.log('=== TEST SUMMARY ===');
  console.log('Page Errors count:', pageErrors.length);
  console.log('Console Errors count:', consoleErrors.length);

  await browser.close();
  
  if (pageErrors.length > 0 || consoleErrors.length > 0) {
    process.exit(1);
  }
  console.log('PASSED CLEANLY!');
})();
