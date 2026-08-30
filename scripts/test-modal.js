const { chromium } = require('/home/desenvolvedor/.nvm/versions/node/v22.17.0/lib/node_modules/@opengsd/gsd-pi/node_modules/playwright');
const fs = require('fs');

(async () => {
  await new Promise(r => setTimeout(r, 1000));
  
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

  console.log('1. Navigating to fresh bootstrap URL:', bootstrapUrl);
  await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded' });
  
  // Wait for loading spinner to detach
  try {
    await page.waitForSelector('.nx-app-loading', { state: 'detached', timeout: 8000 });
  } catch (e) {
    console.log('Wait note:', e.message);
  }
  await page.waitForTimeout(1500);

  // Click on 'Browse Folders on OS...' / 'Procurar Pastas no SO...'
  console.log('2. Clicking Browse Folders button...');
  const browseBtn = await page.locator('.nx-hub-os-actions button, .nx-input-with-action button, button:has-text("Browse"), button:has-text("Procurar")').first();
  await browseBtn.click();
  await page.waitForTimeout(2000);

  await page.screenshot({ path: '/tmp/nexus-browser-modal-fixed.png' });
  console.log('Modal screenshot saved to /tmp/nexus-browser-modal-fixed.png');

  // Let's inspect modal content
  const bookmarkButtons = await page.$$('.nx-dir-bookmarks-list button');
  const bookmarks = [];
  for (const b of bookmarkButtons) {
    bookmarks.push((await b.innerText()).trim());
  }
  console.log('Bookmarks rendered:', bookmarks);

  const crumbButtons = await page.$$('.nx-dir-breadcrumbs-list button');
  const breadcrumbs = [];
  for (const b of crumbButtons) {
    breadcrumbs.push((await b.innerText()).trim());
  }
  console.log('Breadcrumbs rendered:', breadcrumbs);

  const entryCards = await page.$$('.nx-dir-entry-card');
  console.log('Directory Entry Cards count:', entryCards.length);
  const sampleEntries = [];
  for (let i = 0; i < Math.min(entryCards.length, 10); i++) {
    sampleEntries.push((await entryCards[i].innerText()).replace(/\n/g, ' '));
  }
  console.log('Sample entries:', sampleEntries);

  const currentPathEl = await page.$('.nx-dir-selected-info code');
  const currentPath = currentPathEl ? await currentPathEl.innerText() : 'N/A';
  console.log('Current Path in footer:', currentPath);

  console.log('--- Page Errors ---', pageErrors);
  console.log('--- Console Errors ---', consoleErrors);

  await browser.close();
  
  if (entryCards.length === 0 || bookmarks.length === 0) {
    console.error('FAIL: Directory entries or bookmarks are empty!');
    process.exit(1);
  }
  console.log('SUCCESS: Directory browser modal is fully functional with live directories!');
})();
