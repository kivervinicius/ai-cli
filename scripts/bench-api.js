const fs = require('fs');

(async () => {
  const log = fs.readFileSync('/tmp/nexus-web.log', 'utf8');
  const match = log.match(/Bootstrap:\s+(http:\/\/[^\s]+)/);
  const bootstrapUrl = match[1];

  const res = await fetch(bootstrapUrl, { redirect: 'manual' });
  const setCookie = res.headers.get('set-cookie');
  if (!setCookie) {
    console.error('No set-cookie returned!');
    process.exit(1);
  }
  const cookie = setCookie.split(';')[0];
  console.log('Session Cookie established:', cookie);

  const endpoints = [
    '/api/v1/session',
    '/api/v1/projects',
    '/api/v1/workspaces',
    '/api/v1/runtimes',
    '/api/v1/providers',
    '/api/v1/profiles',
    '/api/v1/events'
  ];

  for (const ep of endpoints) {
    const t0 = Date.now();
    try {
      const r = await fetch('http://127.0.0.1:3000' + ep, { headers: { Cookie: cookie } });
      const data = await r.json();
      const elapsed = Date.now() - t0;
      console.log(`[${r.status}] ${ep} took ${elapsed}ms -> count: ${Array.isArray(data) ? data.length : Object.keys(data).length}`);
    } catch (e) {
      console.log(`[ERR] ${ep} failed in ${Date.now() - t0}ms: ${e.message}`);
    }
  }
})();
