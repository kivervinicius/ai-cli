const { chromium } = require('/home/desenvolvedor/.nvm/versions/node/v22.17.0/lib/node_modules/@opengsd/gsd-pi/node_modules/playwright');
const fs = require('fs');

(async () => {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/google-chrome',
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const context = await browser.newContext();
  const page = await context.newPage();
  
  const consoleMsgs = [];
  const pageErrors = [];
  const networkEvents = [];
  
  page.on('console', msg => {
    consoleMsgs.push({ type: msg.type(), text: msg.text() });
    console.log(`[CONSOLE ${msg.type().toUpperCase()}] ${msg.text()}`);
  });
  page.on('pageerror', err => {
    pageErrors.push(err.toString());
    console.log(`[PAGE ERROR] ${err.toString()}`);
  });
  page.on('request', req => {
    networkEvents.push({ type: 'req', url: req.url(), method: req.method() });
  });
  page.on('response', res => {
    networkEvents.push({ type: 'res', url: res.url(), status: res.status() });
    console.log(`[HTTP ${res.status()}] ${res.request().method()} ${res.url()}`);
  });
  
  const log = fs.readFileSync('/tmp/nexus-web.log', 'utf8');
  const match = log.match(/Bootstrap:\s+(http:\/\/[^\s]+)/);
  const bootstrapUrl = match ? match[1] : 'http://127.0.0.1:3000/';

  console.log('Navigating to:', bootstrapUrl);
  await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded' });
  
  console.log('Waiting for content to load...');
  try {
    // Wait up to 5 seconds for spinner to disappear
    await page.waitForSelector('.nx-app-loading', { state: 'detached', timeout: 5000 });
    console.log('App loading spinner detached successfully!');
  } catch (e) {
    console.log('Loading spinner did not detach within 5s:', e.message);
  }

  await page.waitForTimeout(1000);
  
  const bodyText = await page.evaluate(() => document.body.innerText);
  console.log('=== Body Text ===\n', bodyText);
  
  await page.screenshot({ path: '/tmp/nexus-screen-dashboard.png' });
  console.log('Screenshot saved to /tmp/nexus-screen-dashboard.png');

  // Let's inspect buttons and links
  const buttons = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('button')).map(b => ({
      text: b.innerText.trim().replace(/\n/g, ' '),
      aria: b.getAttribute('aria-label') || '',
      className: b.className
    }));
  });
  console.log('Buttons count:', buttons.length);
  console.log('Buttons:', buttons);

  console.log('--- Final Page Errors ---', pageErrors);
  console.log('--- Final Console Errors ---', consoleMsgs.filter(m => m.type === 'error'));

  await browser.close();
})();
