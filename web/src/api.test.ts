import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from './api';

describe('api request layer', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
  });

  it('throws on non-ok responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      json: () => Promise.resolve({ error: 'authentication required' }),
    }));

    await expect(api.getWorkspaces()).rejects.toThrow('authentication required');
  });

  it('sends JSON bodies on state-changing calls', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'stopping' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await api.stopRuntime('rt-1');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/runtimes/rt-1/stop');
    expect(init.method).toBe('POST');
    expect(init.headers.get('Content-Type')).toBe('application/json');
  });

  it('includes CSRF token header after session bootstrap', async () => {
    const sessionFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'tok-123' }),
    });
    vi.stubGlobal('fetch', sessionFetch);
    const { initSession } = await import('./api');

    await initSession();

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await api.stopRuntime('rt-2');

    const [, init] = fetchMock.mock.calls[0];
    expect(init.headers.get('X-CSRF-Token')).toBe('tok-123');
  });
});

it('rotates session and uses the fresh CSRF token for subsequent mutations', async () => {
  const fetchMock = vi.fn()
    .mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'fresh-token' }),
    })
    .mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });
  vi.stubGlobal('fetch', fetchMock);
  const { rotateSession, api } = await import('./api');

  await rotateSession();
  await api.stopRuntime('rt-rotate');

  expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/session/rotate');
  expect(fetchMock.mock.calls[1][1].headers.get('X-CSRF-Token')).toBe('fresh-token');
});

it('logout clears the local CSRF token', async () => {
  const bootstrapFetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ authenticated: true, csrf_token: 'old-token' }),
  });
  vi.stubGlobal('fetch', bootstrapFetch);
  const { initSession, logoutSession, api } = await import('./api');
  await initSession();

  const fetchMock = vi.fn()
    .mockResolvedValueOnce({ ok: true, status: 204, json: () => Promise.resolve({}) })
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ status: 'ok' }) });
  vi.stubGlobal('fetch', fetchMock);
  await logoutSession();
  await api.stopRuntime('rt-after-logout');

  expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/session/logout');
  expect(fetchMock.mock.calls[1][1].headers.get('X-CSRF-Token')).toBeNull();
});
