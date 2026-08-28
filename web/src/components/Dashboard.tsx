import React, { useState } from 'react';
import { RuntimeSession, ProviderInfo, Workspace } from '../types';
import { Cpu, Play, Square, ArrowRightLeft, FastForward, Trash2, Folder, Layers, History } from 'lucide-react';

interface DashboardProps {
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  workspaces: Workspace[];
  activeWorkspace?: string;
  onOpenTerminal: (id: string) => void;
  onOpenStartModal: () => void;
  onOpenHandoffModal: (runtime: RuntimeSession) => void;
  onOpenContinueModal: (runtime: RuntimeSession) => void;
  onStopRuntime: (id: string) => void;
  onDeleteRuntime?: (id: string) => void;
  onCleanInactive?: () => void;
}

export const Dashboard: React.FC<DashboardProps> = ({
  runtimes,
  providers,
  workspaces,
  activeWorkspace,
  onOpenTerminal,
  onOpenStartModal,
  onOpenHandoffModal,
  onOpenContinueModal,
  onStopRuntime,
  onDeleteRuntime,
  onCleanInactive,
}) => {
  const [showAllWorkspaces, setShowAllWorkspaces] = useState<boolean>(false);

  const normalize = (p?: string) => (p || '').trim().replace(/[\\/]+$/, '').toLowerCase();

  // Filter runtimes by active workspace unless "All Projects" is toggled
  const projectRuntimes = runtimes.filter((r) => {
    if (showAllWorkspaces || !activeWorkspace) return true;
    const normActive = normalize(activeWorkspace);
    const normR = normalize(r.workspace);
    return normR === normActive || normR.startsWith(normActive + '/');
  });

  // Only truly running or starting agents are "Live"
  const liveRuntimes = projectRuntimes.filter(
    (r) => r.state === 'RUNNING' || r.state === 'STARTING' || r.state === 'HANDOFF'
  );

  // Past/offline sessions go to Session History
  const pastRuntimes = projectRuntimes.filter(
    (r) => r.state !== 'RUNNING' && r.state !== 'STARTING' && r.state !== 'HANDOFF'
  );

  const activeTotalCount = runtimes.filter((r) => r.state === 'RUNNING' || r.state === 'STARTING').length;
  const installedCount = providers.filter((p) => p.installed).length;

  return (
    <div className="space-y-6">
      {/* Top Banner & Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <div className="text-xs font-mono text-slate-400">Supervised Runtimes</div>
          <div className="mt-2 text-2xl font-bold font-mono text-emerald-400">
            {liveRuntimes.length}
            {activeWorkspace && !showAllWorkspaces && (
              <span className="text-xs text-slate-500 font-normal ml-2">/ {activeTotalCount} total</span>
            )}
          </div>
          <div className="mt-1 text-[11px] text-slate-500">
            {activeWorkspace && !showAllWorkspaces ? 'Active in current project' : 'Active across all projects'}
          </div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <div className="text-xs font-mono text-slate-400">Available Providers</div>
          <div className="mt-2 text-2xl font-bold font-mono text-sky-400">
            {installedCount} / {providers.length}
          </div>
          <div className="mt-1 text-[11px] text-slate-500">Installed in local PATH</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <div className="text-xs font-mono text-slate-400">Registered Projects</div>
          <div className="mt-2 text-2xl font-bold font-mono text-indigo-400">{workspaces.length}</div>
          <div className="mt-1 text-[11px] text-slate-500">Workspaces in config</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4 flex flex-col justify-between">
          <div className="text-xs font-mono text-slate-400">Quick Launch</div>
          <button
            onClick={onOpenStartModal}
            className="mt-2 w-full flex items-center justify-center space-x-2 py-2 px-3 bg-sky-600 hover:bg-sky-500 text-white rounded text-xs font-semibold shadow-md shadow-sky-900/30 transition"
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>Launch Agent</span>
          </button>
        </div>
      </div>

      {/* Scope Filter Switcher */}
      {activeWorkspace && (
        <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-800 text-xs font-mono">
          <div className="flex items-center space-x-2 text-slate-400">
            <Folder className="w-4 h-4 text-sky-400" />
            <span>Scope:</span>
            <span className="text-slate-200 font-semibold truncate max-w-sm">{activeWorkspace}</span>
          </div>

          <div className="flex items-center space-x-1.5 bg-slate-950 p-1 rounded-md border border-slate-800/80">
            <button
              onClick={() => setShowAllWorkspaces(false)}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center space-x-1 ${
                !showAllWorkspaces
                  ? 'bg-sky-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <span>Project Only</span>
              <span className="ml-1 px-1.5 py-0.2 rounded-full bg-black/30 text-[10px]">
                {liveRuntimes.length}
              </span>
            </button>
            <button
              onClick={() => setShowAllWorkspaces(true)}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center space-x-1 ${
                showAllWorkspaces
                  ? 'bg-sky-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Layers className="w-3 h-3" />
              <span>All Projects</span>
              <span className="ml-1 px-1.5 py-0.2 rounded-full bg-black/30 text-[10px]">
                {runtimes.length}
              </span>
            </button>
          </div>
        </div>
      )}

      {/* LIVE RUNTIMES CARD (Active Only) */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-sm">
        <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2.5">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
            <h2 className="text-xs font-bold font-mono text-slate-200 uppercase tracking-wider">
              Live Runtimes ({liveRuntimes.length})
            </h2>
          </div>
          <span className="text-[11px] font-mono text-slate-500">Active coding processes</span>
        </div>

        {liveRuntimes.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-xs font-mono space-y-2">
            <div>No live agent runtimes active in this project.</div>
            <div className="text-slate-600 text-[11px]">
              Click <span className="text-sky-400">"Launch Agent"</span> above or run{' '}
              <code className="text-slate-400 bg-slate-950 px-1.5 py-0.5 rounded border border-slate-800">
                ai control start &lt;provider&gt;
              </code>{' '}
              to start one.
            </div>
          </div>
        ) : (
          <div className="divide-y divide-slate-800">
            {liveRuntimes.map((r) => {
              const prov = r.provider_id || r.provider || 'AI';
              const prof = r.profile_id || r.profile || 'default';
              const title = r.title || `${prov.toUpperCase()} (${prof})`;
              return (
                <div
                  key={r.runtime_id}
                  className="p-4 flex items-center justify-between hover:bg-slate-800/30 transition"
                >
                  <div className="flex items-center space-x-3">
                    <div className="w-3 h-3 rounded-full bg-emerald-500 animate-pulse" />
                    <div>
                      <div className="flex items-center space-x-2">
                        <span className="font-bold text-sm text-slate-100 font-sans">{title}</span>
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-sky-950 text-sky-300 border border-sky-800 font-mono uppercase">
                          {prov}
                        </span>
                        <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                          {prof}
                        </span>
                        <span className="text-xs font-mono text-slate-500">ID: {r.runtime_id}</span>
                        {r.handoff_type && (
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-950 text-indigo-300 border border-indigo-800 font-mono">
                            {r.handoff_type}
                          </span>
                        )}
                      </div>
                      <div className="mt-1 text-xs text-slate-400 font-mono truncate max-w-md">
                        {r.workspace}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => onOpenTerminal(r.runtime_id)}
                      className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded text-xs font-medium transition flex items-center space-x-1.5 shadow-sm"
                    >
                      <Cpu className="w-3.5 h-3.5" />
                      <span>Terminal</span>
                    </button>

                    <button
                      onClick={() => onOpenHandoffModal(r)}
                      className="px-2.5 py-1.5 bg-slate-800 hover:bg-indigo-950 text-slate-300 hover:text-indigo-200 rounded text-xs font-medium transition flex items-center space-x-1"
                      title="Account Handoff"
                    >
                      <ArrowRightLeft className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">Handoff</span>
                    </button>

                    <button
                      onClick={() => onOpenContinueModal(r)}
                      className="px-2.5 py-1.5 bg-slate-800 hover:bg-emerald-950 text-slate-300 hover:text-emerald-200 rounded text-xs font-medium transition flex items-center space-x-1"
                      title="Continue With AI"
                    >
                      <FastForward className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">Continue</span>
                    </button>

                    <button
                      onClick={() => onStopRuntime(r.runtime_id)}
                      className="px-2.5 py-1.5 bg-slate-800 hover:bg-rose-950 text-slate-300 hover:text-rose-300 rounded text-xs font-medium transition"
                      title="Stop Process"
                    >
                      <Square className="w-3.5 h-3.5 fill-current" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* SESSION HISTORY CARD (Past / Offline Sessions) */}
      {pastRuntimes.length > 0 && (
        <div className="bg-slate-900 border border-slate-800/80 rounded-lg overflow-hidden opacity-95">
          <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
            <div className="flex items-center space-x-2.5">
              <History className="w-4 h-4 text-slate-400" />
              <h2 className="text-xs font-bold font-mono text-slate-300 uppercase tracking-wider">
                Session History ({pastRuntimes.length})
              </h2>
            </div>
            {onCleanInactive && (
              <button
                onClick={onCleanInactive}
                className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-slate-200 text-[10px] font-mono transition"
                title="Clear all inactive/stale sessions"
              >
                Clear Inactive
              </button>
            )}
          </div>

          <div className="divide-y divide-slate-800/70">
            {pastRuntimes.map((r) => {
              const prov = r.provider_id || r.provider || 'AI';
              const prof = r.profile_id || r.profile || 'default';
              const title = r.title || `${prov.toUpperCase()} (${prof})`;
              return (
                <div
                  key={r.runtime_id}
                  className="p-3.5 flex items-center justify-between hover:bg-slate-800/20 transition"
                >
                  <div className="flex items-center space-x-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-slate-600" />
                    <div>
                      <div className="flex items-center space-x-2">
                        <span className="font-semibold text-sm text-slate-300 font-sans">{title}</span>
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-950 text-slate-400 border border-slate-800 font-mono uppercase">
                          {prov}
                        </span>
                        <span className="text-xs px-2 py-0.5 rounded bg-slate-800/80 text-slate-400 font-mono">
                          {prof}
                        </span>
                        <span className="text-xs font-mono text-slate-500">ID: {r.runtime_id}</span>
                        <span
                          className={`text-[10px] px-1.5 py-0.5 rounded font-mono uppercase ${
                            r.state === 'STOPPED'
                              ? 'bg-slate-800 text-slate-400'
                              : r.state === 'FAILED'
                              ? 'bg-rose-950/60 text-rose-400 border border-rose-900/60'
                              : 'bg-amber-950/60 text-amber-400 border border-amber-900/60'
                          }`}
                        >
                          {r.state}
                        </span>
                      </div>
                      <div className="mt-0.5 text-xs text-slate-500 font-mono truncate max-w-md">
                        {r.workspace}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => onOpenContinueModal(r)}
                      className="px-2.5 py-1 bg-slate-800 hover:bg-emerald-950 text-slate-300 hover:text-emerald-200 rounded text-xs font-medium transition flex items-center space-x-1"
                      title="Resume conversation with new session"
                    >
                      <FastForward className="w-3.5 h-3.5" />
                      <span>Resume</span>
                    </button>

                    {onDeleteRuntime && (
                      <button
                        onClick={() => onDeleteRuntime(r.runtime_id)}
                        className="p-1 text-slate-500 hover:text-rose-400 hover:bg-slate-800 rounded transition"
                        title="Delete Session Record"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

