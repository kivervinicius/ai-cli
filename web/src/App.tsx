import React, { useState, useEffect } from 'react';
import { api, initSession } from './api';
import { setNexusCSRF } from './nexus/api';
import { AppShell, CommandPalette, PlaceholderPage } from './nexus/AppShell';
import { ProjectsPage } from './nexus/ProjectsPage';
import { Workspace, RuntimeSession, ProviderInfo, ProfileInfo, EventRecord } from './types';
import { Dashboard } from './components/Dashboard';
import { TerminalView } from './components/TerminalView';
import { ProvidersView } from './components/ProvidersView';
import { EventsView } from './components/EventsView';
import { StartModal } from './components/StartModal';
import { HandoffModal } from './components/HandoffModal';
import { ContinueModal } from './components/ContinueModal';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<string>('overview');
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [activeWorkspace, setActiveWorkspace] = useState<string>('');
  const [runtimes, setRuntimes] = useState<RuntimeSession[]>([]);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [profiles, setProfiles] = useState<ProfileInfo[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [paletteOpen, setPaletteOpen] = useState(false);

  const [activeTerminalId, setActiveTerminalId] = useState<string>('');
  const [showStartModal, setShowStartModal] = useState<boolean>(false);
  const [handoffRuntime, setHandoffRuntime] = useState<RuntimeSession | null>(null);
  const [continueRuntime, setContinueRuntime] = useState<RuntimeSession | null>(null);

  useEffect(() => {
    initSession().then((sess) => {
      if (sess.csrf_token) setNexusCSRF(sess.csrf_token);
      fetchStaticData();
      fetchDynamicData();
    });
    const interval = setInterval(fetchDynamicData, 3000);
    return () => clearInterval(interval);
  }, []);

  // Command palette shortcut: Ctrl/Cmd + K
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const fetchStaticData = async () => {
    try {
      const [ws, provs, profs] = await Promise.all([
        api.getWorkspaces(),
        api.getProviders(),
        api.getProfiles(),
      ]);
      setWorkspaces(ws);
      if (ws.length > 0) setActiveWorkspace(ws[0].path);
      setProviders(provs);
      setProfiles(profs);
    } catch (e) {
      console.error('Failed to load static data', e);
    }
  };

  const fetchDynamicData = async () => {
    try {
      const [rts, evs] = await Promise.all([api.getRuntimes(), api.getEvents()]);
      setRuntimes(rts);
      setEvents(evs);
    } catch (e) {
      console.error('Failed to load dynamic data', e);
    }
  };

  const handleOpenTerminal = (runtimeId: string) => {
    setActiveTerminalId(runtimeId);
    setCurrentTab('terminals');
  };

  const handleStopRuntime = async (id: string) => {
    try {
      await api.stopRuntime(id);
      fetchDynamicData();
    } catch (e) {
      console.error('Failed to stop runtime', e);
    }
  };

  const handleDeleteRuntime = async (id: string) => {
    try {
      await api.deleteRuntime(id);
      fetchDynamicData();
    } catch (e) {
      console.error('Failed to delete runtime', e);
    }
  };

  const handleCleanInactive = async () => {
    try {
      await api.cleanRuntimes();
      fetchDynamicData();
    } catch (e) {
      console.error('Failed to clean inactive runtimes', e);
    }
  };

  const handleUpdateTitle = async (id: string, title: string) => {
    try {
      await api.updateRuntimeTitle(id, title);
      fetchDynamicData();
    } catch (e) {
      console.error('Failed to update runtime title', e);
    }
  };

  return (
    <AppShell current={currentTab} onNavigate={setCurrentTab} onCommandPalette={() => setPaletteOpen(true)}>
      {currentTab === 'overview' && <ProjectsPage />}
      {currentTab === 'projects' && <ProjectsPage />}
      {currentTab === 'agents' && <ProjectsPage />}
      {currentTab === 'resources' && (
        <PlaceholderPage
          title="Resources"
          hint="Providers, accounts, quotas and the smart Resource Scheduler arrive in Gate 5."
        />
      )}
      {currentTab === 'maestro' && (
        <PlaceholderPage title="Maestro Assist" hint="Process, skill and verification recommendations arrive in Gate 6." />
      )}
      {currentTab === 'sessions' && (
        <PlaceholderPage title="Sessions" hint="Session continuity and lineage view arrives with persistent agents." />
      )}
      {currentTab === 'settings' && (
        <PlaceholderPage title="Settings" hint="Nexus settings arrive with the configuration drawer (Gate 3)." />
      )}

      {currentTab === 'runtimes' && (
        <Dashboard
          runtimes={runtimes}
          providers={providers}
          workspaces={workspaces}
          activeWorkspace={activeWorkspace}
          onOpenTerminal={handleOpenTerminal}
          onOpenStartModal={() => setShowStartModal(true)}
          onOpenHandoffModal={(r) => setHandoffRuntime(r)}
          onOpenContinueModal={(r) => setContinueRuntime(r)}
          onStopRuntime={handleStopRuntime}
          onDeleteRuntime={handleDeleteRuntime}
          onCleanInactive={handleCleanInactive}
        />
      )}

      <div className={currentTab === 'terminals' ? 'h-full flex flex-col' : 'hidden'}>
        <TerminalView
          runtimes={runtimes.filter((r) => r.state === 'RUNNING' || r.state === 'STARTING')}
          activeRuntimeId={activeTerminalId}
          onSelectRuntime={setActiveTerminalId}
          onUpdateTitle={handleUpdateTitle}
        />
      </div>

      {currentTab === 'providers' && <ProvidersView providers={providers} />}
      {currentTab === 'events' && <EventsView events={events} />}

      {showStartModal && (
        <StartModal
          providers={providers}
          profiles={profiles}
          workspace={activeWorkspace}
          workspaces={workspaces}
          onClose={() => setShowStartModal(false)}
          onSuccess={(newSession) => {
            fetchDynamicData();
            handleOpenTerminal(newSession.runtime_id);
          }}
        />
      )}

      {handoffRuntime && (
        <HandoffModal
          runtime={handoffRuntime}
          profiles={profiles}
          onClose={() => setHandoffRuntime(null)}
          onSuccess={() => {
            fetchDynamicData();
          }}
        />
      )}

      {continueRuntime && (
        <ContinueModal
          runtime={continueRuntime}
          providers={providers}
          profiles={profiles}
          onClose={() => setContinueRuntime(null)}
          onSuccess={(newSession) => {
            fetchDynamicData();
            handleOpenTerminal(newSession.runtime_id);
          }}
        />
      )}

      {paletteOpen && <CommandPalette onClose={() => setPaletteOpen(false)} onNavigate={setCurrentTab} />}
    </AppShell>
  );
};
