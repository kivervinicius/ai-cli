/* Nexus product API client (projects, agents, layouts, config). */

import { Project, Agent, AgentDetail, RuntimeSession, AgentConfig, ConfigImpact } from '../types';

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
  recoverAgent: (id: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/recover`, { method: 'POST', body: '{}' }),

  // Agent config (Gate 3)
  getAgentConfig: (id: string) =>
    request<{ config: AgentConfig; revision: string; revisions: any[] }>(`/api/v1/agents/${id}/config`),
  applyAgentConfig: (id: string, config: AgentConfig) =>
    request<{ impact: ConfigImpact }>(`/api/v1/agents/${id}/config/apply`, {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  previewAgentConfig: (id: string, config: AgentConfig) =>
    request<{ impact: ConfigImpact }>(`/api/v1/agents/${id}/config/impact`, {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  // Resource Scheduler (Gate 5)
  listResources: () =>
    request<{ accounts: any[]; policy: string }>('/api/v1/resources'),
  selectResource: (provider: string, policy: string, agentId?: string) =>
    request<{ decision: any }>('/api/v1/resources/select', {
      method: 'POST',
      body: JSON.stringify({ provider, policy, agent_id: agentId }),
    }),

  // Maestro Assist (Gate 6)
  getMaestroStatus: () => request<any>('/api/v1/maestro'),
  getMaestroAdvice: (projectId: string, agentId?: string, intent?: string) =>
    request<any>('/api/v1/maestro/advice', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, agent_id: agentId, intent }),
    }),

  // Missions (Gate 7 Beta)
  listMissions: (projectId: string) =>
    request<{ missions: any[] }>(`/api/v1/projects/${projectId}/missions`),
  createMission: (projectId: string, data: { name: string; description?: string; goal?: string; scope?: string; risk_level?: string }) =>
    request<any>(`/api/v1/projects/${projectId}/missions`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  getMission: (missionId: string) =>
    request<any>(`/api/v1/missions/${missionId}`),
  updateMission: (missionId: string, data: any) =>
    request<any>(`/api/v1/missions/${missionId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  deleteMission: (missionId: string) =>
    request<any>(`/api/v1/missions/${missionId}`, { method: 'DELETE' }),
  addMissionTask: (missionId: string, data: { name: string; description?: string; kind?: string; priority?: number }) =>
    request<any>(`/api/v1/missions/${missionId}/tasks`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  assignMissionAgent: (missionId: string, data: { task_id: string; agent_id: string }) =>
    request<any>(`/api/v1/missions/${missionId}/assign`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
};
