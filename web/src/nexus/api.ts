/* Nexus product API client (projects, agents, layouts). */

import { Project, Agent, AgentDetail, RuntimeSession } from '../types';

let csrf = '';
export function setNexusCSRF(token: string) {
  csrf = token;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');
  if (options.method && options.method !== 'GET' && options.method !== 'HEAD') {
    headers.set('Content-Type', 'application/json');
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const res = await fetch(path, { ...options, headers });
  if (!res.ok) {
    const errBody = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errBody.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const nexus = {
  listProjects: () => request<Project[]>('/api/v1/projects'),
  createProject: (path: string, name?: string) =>
    request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify({ path, name }) }),
  getProject: (id: string) => request<{ project: Project; layout: string }>(`/api/v1/projects/${id}`),
  deleteProject: (id: string) => request<{ status: string }>(`/api/v1/projects/${id}`, { method: 'DELETE' }),
  getLayout: (projectId: string) => request<{ layout: string }>(`/api/v1/projects/${projectId}/layout`),
  saveLayout: (projectId: string, layout: string) =>
    request<{ status: string }>(`/api/v1/projects/${projectId}/layout`, {
      method: 'PUT',
      body: JSON.stringify({ layout }),
    }),

  listAgents: (projectId: string) => request<Agent[]>(`/api/v1/projects/${projectId}/agents`),
  createAgent: (projectId: string, name: string, role?: string) =>
    request<Agent>(`/api/v1/projects/${projectId}/agents`, {
      method: 'POST',
      body: JSON.stringify({ name, role }),
    }),
  getAgent: (id: string) => request<AgentDetail>(`/api/v1/agents/${id}`),
  deleteAgent: (id: string) => request<{ status: string }>(`/api/v1/agents/${id}`, { method: 'DELETE' }),
  startAgent: (id: string, provider?: string, profile?: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/start`, {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  stopAgent: (id: string) =>
    request<{ status: string }>(`/api/v1/agents/${id}/stop`, { method: 'POST', body: '{}' }),
};
