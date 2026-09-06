import { Workspace, RuntimeSession, ProviderInfo, ProfileInfo, EventRecord } from './types';
import { isDesktopApp, getPlatformBridge } from './platform';

let csrfToken = '';
let desktopAuthToken = '';
let desktopBaseUrl = '';

export type BrowserSession = {
  authenticated: boolean;
  csrf_token?: string;
  expires_at?: string;
  idle_timeout?: number;
};

export function setDesktopAuth(token: string, baseUrl?: string) {
  desktopAuthToken = token;
  if (baseUrl) {
    desktopBaseUrl = baseUrl.replace(/\/+$/, '');
  }
}

export function getDesktopAuthToken(): string {
  return desktopAuthToken;
}

export function getDesktopBaseUrl(): string {
  return desktopBaseUrl;
}

export function getWebSocketEndpoint(path: string): string {
  const token = getDesktopAuthToken();
  const baseUrl = getDesktopBaseUrl();
  let proto =
    typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  let host = typeof window !== 'undefined' ? window.location.host : 'localhost';

  if (baseUrl) {
    try {
      const parsed = new URL(baseUrl);
      proto = parsed.protocol === 'https:' ? 'wss:' : 'ws:';
      host = parsed.host;
    } catch {
      // ignore
    }
  }

  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  const url = `${proto}//${host}${cleanPath}`;
  if (!token) return url;

  const delimiter = url.includes('?') ? '&' : '?';
  return `${url}${delimiter}token=${encodeURIComponent(token)}`;
}

function notifySessionExpired() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('nexus:session-expired'));
  }
}

export async function initSession(): Promise<BrowserSession> {
  try {
    if (isDesktopApp()) {
      try {
        const bridge = getPlatformBridge();
        if (bridge.getBootstrapInfo) {
          const bootstrap = await bridge.getBootstrapInfo();
          if (bootstrap && bootstrap.sessionToken) {
            setDesktopAuth(bootstrap.sessionToken, bootstrap.serverUrl);
            if (bootstrap.csrfToken) {
              csrfToken = bootstrap.csrfToken;
            }
            const targetUrl = bootstrap.serverUrl
              ? `${bootstrap.serverUrl}/api/v1/session`
              : '/api/v1/session';
            try {
              const res = await fetch(targetUrl, {
                headers: {
                  Accept: 'application/json',
                  Authorization: `Bearer ${bootstrap.sessionToken}`,
                },
              });
              if (res.ok) {
                const data = await res.json();
                if (data.csrf_token) csrfToken = data.csrf_token;
                return {
                  authenticated: true,
                  csrf_token: csrfToken || bootstrap.csrfToken,
                  expires_at: data.expires_at,
                  idle_timeout: data.idle_timeout,
                };
              }
            } catch (verifyErr) {
              console.warn(
                'Desktop session verify probe error, using pre-authenticated bootstrap token:',
                verifyErr,
              );
            }
            return {
              authenticated: true,
              csrf_token: bootstrap.csrfToken,
            };
          }
        }
      } catch (bridgeErr) {
        console.warn('Desktop session bootstrap initialization failed:', bridgeErr);
      }
    }

    const headers: Record<string, string> = { Accept: 'application/json' };
    if (desktopAuthToken) {
      headers['Authorization'] = `Bearer ${desktopAuthToken}`;
    }
    const targetUrl = desktopBaseUrl ? `${desktopBaseUrl}/api/v1/session` : '/api/v1/session';
    const res = await fetch(targetUrl, { headers });
    if (!res.ok) return { authenticated: false };
    const data = await res.json();
    if (data.csrf_token) {
      csrfToken = data.csrf_token;
    }
    if (data.authenticated && !desktopAuthToken && isDesktopApp()) {
      try {
        const b = await getPlatformBridge().getBootstrapInfo?.();
        if (b && b.sessionToken) {
          setDesktopAuth(b.sessionToken, b.serverUrl);
        }
      } catch {}
    }
    return data;
  } catch {
    return { authenticated: false };
  }
}

export async function rotateSession(): Promise<BrowserSession> {
  const headers: Record<string, string> = {};
  if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
  if (desktopAuthToken) headers['Authorization'] = `Bearer ${desktopAuthToken}`;
  const targetUrl = desktopBaseUrl
    ? `${desktopBaseUrl}/api/v1/session/rotate`
    : '/api/v1/session/rotate';
  const res = await fetch(targetUrl, {
    method: 'POST',
    headers,
  });
  if (!res.ok) {
    notifySessionExpired();
    return { authenticated: false };
  }
  const data = (await res.json()) as BrowserSession;
  if (data.csrf_token) csrfToken = data.csrf_token;
  return data;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');

  if (desktopAuthToken) {
    headers.set('Authorization', `Bearer ${desktopAuthToken}`);
  }

  if (options.method && options.method !== 'GET' && options.method !== 'HEAD') {
    headers.set('Content-Type', 'application/json');
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken);
    }
  }

  const targetUrl = desktopBaseUrl && path.startsWith('/') ? `${desktopBaseUrl}${path}` : path;
  const res = await fetch(targetUrl, { ...options, headers });
  if (!res.ok) {
    if (res.status === 401) notifySessionExpired();
    const errBody = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errBody.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  getWorkspaces: () => request<Workspace[]>('/api/v1/workspaces'),
  addWorkspace: (path: string, name?: string) =>
    request<Workspace>('/api/v1/workspaces', {
      method: 'POST',
      body: JSON.stringify({ path, name }),
    }),
  removeWorkspace: (pathOrId: string) =>
    request<{ status: string }>(`/api/v1/workspaces?path=${encodeURIComponent(pathOrId)}`, {
      method: 'DELETE',
    }),
  getRuntimes: () => request<RuntimeSession[]>('/api/v1/runtimes'),
  getRuntime: (id: string) =>
    request<{ session: RuntimeSession; capabilities: any }>(`/api/v1/runtimes/${id}`),
  startRuntime: (
    provider: string,
    profile: string,
    workspace: string,
    args: string[] = [],
    title?: string,
  ) =>
    request<RuntimeSession>('/api/v1/runtimes', {
      method: 'POST',
      body: JSON.stringify({ provider, profile, workspace, args, title }),
    }),
  updateRuntimeTitle: (id: string, title: string) =>
    request<{ status: string; title: string }>(`/api/v1/runtimes/${id}/title`, {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
  respondRuntime: (id: string, input: string) =>
    request<{ status: string; sent: string }>(`/api/v1/runtimes/${id}/respond`, {
      method: 'POST',
      body: JSON.stringify({ input }),
    }),
  stopRuntime: (id: string) =>
    request<{ status: string }>(`/api/v1/runtimes/${id}/stop`, { method: 'POST' }),
  accountHandoff: (id: string, target: string) =>
    request<RuntimeSession>(`/api/v1/runtimes/${id}/handoff`, {
      method: 'POST',
      body: JSON.stringify({ target }),
    }),
  contextContinue: (id: string, provider: string, profile: string) =>
    request<RuntimeSession>(`/api/v1/runtimes/${id}/continue`, {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  getProviders: () => request<ProviderInfo[]>('/api/v1/providers'),
  getProfiles: () => request<ProfileInfo[]>('/api/v1/profiles'),
  getEvents: (runtimeId?: string) =>
    request<EventRecord[]>(`/api/v1/events${runtimeId ? `?runtime_id=${runtimeId}` : ''}`),
  deleteRuntime: (id: string) =>
    request<{ status: string }>(`/api/v1/runtimes/${id}`, { method: 'DELETE' }),
  cleanRuntimes: () =>
    request<{ cleaned: number; purged: number }>('/api/v1/runtimes', { method: 'DELETE' }),
};
