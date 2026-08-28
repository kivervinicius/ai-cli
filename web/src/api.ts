import { Workspace, RuntimeSession, ProviderInfo, ProfileInfo, EventRecord } from './types';

let csrfToken = '';

export async function initSession(): Promise<{ authenticated: boolean; csrf_token?: string }> {
  try {
    const res = await fetch('/api/v1/session');
    if (!res.ok) return { authenticated: false };
    const data = await res.json();
    if (data.csrf_token) {
      csrfToken = data.csrf_token;
    }
    return data;
  } catch {
    return { authenticated: false };
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');

  if (options.method && options.method !== 'GET' && options.method !== 'HEAD') {
    headers.set('Content-Type', 'application/json');
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken);
    }
  }

  const res = await fetch(path, { ...options, headers });
  if (!res.ok) {
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
  getRuntime: (id: string) => request<{ session: RuntimeSession; capabilities: any }>(`/api/v1/runtimes/${id}`),
  startRuntime: (provider: string, profile: string, workspace: string, args: string[] = [], title?: string) =>
    request<RuntimeSession>('/api/v1/runtimes', {
      method: 'POST',
      body: JSON.stringify({ provider, profile, workspace, args, title }),
    }),
  updateRuntimeTitle: (id: string, title: string) =>
    request<{ status: string; title: string }>(`/api/v1/runtimes/${id}/title`, {
      method: 'POST',
      body: JSON.stringify({ title }),
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
