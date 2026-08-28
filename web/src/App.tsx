import React, { useState, useEffect } from 'react';
import { api, initSession } from './api';
import { Workspace, RuntimeSession, ProviderInfo, ProfileInfo, EventRecord } from './types';
import { Sidebar } from './components/Sidebar';
import { Dashboard } from './components/Dashboard';
import { TerminalView } from './components/TerminalView';
import { ProvidersView } from './components/ProvidersView';
import { EventsView } from './components/EventsView';
import { StartModal } from './components/StartModal';
import { HandoffModal } from './components/HandoffModal';
import { ContinueModal } from './components/ContinueModal';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<string>('dashboard');
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [activeWorkspace, setActiveWorkspace] = useState<string>('');
  const [runtimes, setRuntimes] = useState<RuntimeSession[]>([]);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [profiles, setProfiles] = useState<ProfileInfo[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);

  const [activeTerminalId, setActiveTerminalId] = useState<string>('');
  const [showStartModal, setShowStartModal] = useState<boolean>(false);
  const [handoffRuntime, setHandoffRuntime] = useState<RuntimeSession | null>(null);
  const [continueRuntime, setContinueRuntime] = useState<RuntimeSession | null>(null);

  // Initial load & authentication
  useEffect(() => {
    initSession().then(() => {
      fetchStaticData();
      fetchDynamicData();
    });

    // 3-second background polling
    const interval = setInterval(fetchDynamicData, 3000);
    return () => clearInterval(interval);
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

  const handleAddWorkspace = async (path: string, name?: string) => {
    try {
      await api.addWorkspace(path, name);
      const ws = await api.getWorkspaces();
      setWorkspaces(ws);
      setActiveWorkspace(path);
    } catch (e) {
      console.error('Failed to add workspace', e);
    }
  };

  const handleRemoveWorkspace = async (path: string) => {
    try {
      await api.removeWorkspace(path);
      const ws = await api.getWorkspaces();
      setWorkspaces(ws);
      if (activeWorkspace === path && ws.length > 0) {
        setActiveWorkspace(ws[0].path);
      }
    } catch (e) {
      console.error('Failed to remove workspace', e);
    }
  };

  return (
    <div className="flex h-screen w-screen bg-slate-950 text-slate-100 overflow-hidden font-sans">
      {/* Left Sidebar */}
      <Sidebar
        currentTab={currentTab}
        onSelectTab={setCurrentTab}
        workspaces={workspaces}
        activeWorkspace={activeWorkspace}
        onSelectWorkspace={setActiveWorkspace}
        runtimeCount={runtimes.filter((r) => r.state === 'RUNNING' || r.state === 'STARTING').length}
        onAddWorkspace={handleAddWorkspace}
        onRemoveWorkspace={handleRemoveWorkspace}
      />

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col h-full overflow-hidden bg-slate-950">
        {/* Top Navbar */}
        <header className="h-12 border-b border-slate-800/80 px-6 flex items-center justify-between text-xs font-mono select-none bg-slate-950/40">
          <div className="flex items-center space-x-2">
            <span className="text-slate-500">Workspace:</span>
            <span className="text-slate-200 font-semibold">{activeWorkspace || 'Default'}</span>
          </div>

          <div className="flex items-center space-x-4">
            <span className="flex items-center space-x-1.5 text-emerald-400 font-medium">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <span>Control Core Healthy</span>
            </span>
          </div>
        </header>

        {/* View Switcher */}
        <div className="flex-1 p-4 overflow-y-auto">
          {currentTab === 'dashboard' && (
            <Dashboard
              runtimes={runtimes}
              providers={providers}
              workspaces={workspaces}
              onOpenTerminal={handleOpenTerminal}
              onOpenStartModal={() => setShowStartModal(true)}
              onOpenHandoffModal={(r) => setHandoffRuntime(r)}
              onOpenContinueModal={(r) => setContinueRuntime(r)}
              onStopRuntime={handleStopRuntime}
              onDeleteRuntime={handleDeleteRuntime}
              onCleanInactive={handleCleanInactive}
            />
          )}

          {currentTab === 'terminals' && (
            <TerminalView
              runtimes={runtimes.filter((r) => r.state === 'RUNNING' || r.state === 'STARTING')}
              activeRuntimeId={activeTerminalId}
              onSelectRuntime={setActiveTerminalId}
              onUpdateTitle={handleUpdateTitle}
            />
          )}

          {currentTab === 'runtimes' && (
            <Dashboard
              runtimes={runtimes}
              providers={providers}
              workspaces={workspaces}
              onOpenTerminal={handleOpenTerminal}
              onOpenStartModal={() => setShowStartModal(true)}
              onOpenHandoffModal={(r) => setHandoffRuntime(r)}
              onOpenContinueModal={(r) => setContinueRuntime(r)}
              onStopRuntime={handleStopRuntime}
              onDeleteRuntime={handleDeleteRuntime}
              onCleanInactive={handleCleanInactive}
            />
          )}

          {currentTab === 'providers' && <ProvidersView providers={providers} />}

          {currentTab === 'events' && <EventsView events={events} />}
        </div>
      </main>

      {/* Modals */}
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
    </div>
  );
};
