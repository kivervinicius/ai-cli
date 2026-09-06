// Manual local browser diagnostic. Never embed a bootstrap URL or print cookies.
// Usage: NEXUS_BOOTSTRAP_URL='http://127.0.0.1:3000/?token=...' node scripts/test-frontend.js
const { chromium } = require('playwright');

const bootstrapURL = process.env.NEXUS_BOOTSTRAP_URL;
if (!bootstrapURL) {
  throw new Error('NEXUS_BOOTSTRAP_URL is required');
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const pageErrors = [];
  page.on('pageerror', (err) => pageErrors.push(err.message));

  await page.goto(bootstrapURL, { waitUntil: 'domcontentloaded' });
  await page.locator('body').waitFor({ state: 'visible' });

  if (pageErrors.length) {
    throw new Error(`browser page errors: ${pageErrors.join('; ')}`);
  }
  console.log('NEXUS_BROWSER_DIAGNOSTIC_PASS');
  await browser.close();
}

main().catch((err) => {
  console.error(`NEXUS_BROWSER_DIAGNOSTIC_FAIL: ${err.message}`);
  process.exitCode = 1;
});
