import { useCallback, useEffect, useRef, useState } from 'react';
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
  const refreshInFlight = useRef<Promise<void> | null>(null);

  const refreshGlobal = useCallback(async () => {
    if (refreshInFlight.current) return refreshInFlight.current;

    const request = (async () => {
      try {
        const [projectList, workspaceList, runtimeList, providerList, profileList, eventList] = await Promise.all([
          nexus.listProjects(), api.getWorkspaces(), api.getRuntimes(), api.getProviders(), api.getProfiles(), api.getEvents(),
        ]);
        setProjects(Array.isArray(projectList) ? projectList : []);
        setWorkspaces(Array.isArray(workspaceList) ? workspaceList : []);
        setRuntimes(Array.isArray(runtimeList) ? runtimeList : []);
        setProviders(Array.isArray(providerList) ? providerList : []);
        setProfiles(Array.isArray(profileList) ? profileList : []);
        setEvents(Array.isArray(eventList) ? eventList : []);
        setError('');
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Unable to load Nexus data');
      } finally {
        setLoading(false);
        refreshInFlight.current = null;
      }
    })();
    refreshInFlight.current = request;
    return request;
  }, []);

  const refreshAgents = useCallback(async (projectId?: string | null) => {
    if (!projectId) { setAgents([]); return; }
    try {
      const list = await nexus.listAgents(projectId);
      setAgents(Array.isArray(list) ? list : []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load Agents');
    }
  }, []);

  useEffect(() => {
    void refreshGlobal();
    const refreshWhenVisible = () => {
      if (!document.hidden) void refreshGlobal();
    };
    const timer = window.setInterval(refreshWhenVisible, 5000);
    document.addEventListener('visibilitychange', refreshWhenVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
    };
  }, [refreshGlobal]);

  return { projects, setProjects, agents, setAgents, workspaces, runtimes, providers, profiles, events, loading, error, refreshGlobal, refreshAgents };
}
