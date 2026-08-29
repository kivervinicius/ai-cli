import { useCallback, useEffect, useState } from 'react';
import { api } from '../api';
import { nexus } from '../nexus/api';
import type { Agent, EventRecord, ProfileInfo, Project, ProviderInfo, RuntimeSession, Workspace } from '../types';

export function useNexusData() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [runtimes, setRuntimes] = useState<RuntimeSession[]>([]);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [profiles, setProfiles] = useState<ProfileInfo[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refreshGlobal = useCallback(async () => {
    try {
      const [projectList, workspaceList, runtimeList, providerList, profileList, eventList] = await Promise.all([
        nexus.listProjects(), api.getWorkspaces(), api.getRuntimes(), api.getProviders(), api.getProfiles(), api.getEvents(),
      ]);
      setProjects(projectList); setWorkspaces(workspaceList); setRuntimes(runtimeList); setProviders(providerList); setProfiles(profileList); setEvents(eventList); setError('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load Nexus data');
    } finally { setLoading(false); }
  }, []);

  const refreshAgents = useCallback(async (projectId?: string | null) => {
    if (!projectId) { setAgents([]); return; }
    try { setAgents(await nexus.listAgents(projectId)); } catch (cause) { setError(cause instanceof Error ? cause.message : 'Unable to load Agents'); }
  }, []);

  useEffect(() => { void refreshGlobal(); const timer = window.setInterval(() => void refreshGlobal(), 5000); return () => window.clearInterval(timer); }, [refreshGlobal]);

  return { projects, setProjects, agents, setAgents, workspaces, runtimes, providers, profiles, events, loading, error, refreshGlobal, refreshAgents };
}
