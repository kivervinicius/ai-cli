import React from 'react';
import { RuntimeSession, ProviderInfo, Workspace } from '../types';
import { Cpu, Play, Square, ArrowRightLeft, FastForward, Trash2 } from 'lucide-react';

interface DashboardProps {
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  workspaces: Workspace[];
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
  onOpenTerminal,
  onOpenStartModal,
  onOpenHandoffModal,
  onOpenContinueModal,
  onStopRuntime,
  onDeleteRuntime,
  onCleanInactive,
}) => {
  const activeCount = runtimes.filter((r) => r.state === 'RUNNING' || r.state === 'STARTING').length;
  const installedCount = providers.filter((p) => p.installed).length;

  return (
    <div className="space-y-6">
      {/* Top Banner & Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <div className="text-xs font-mono text-slate-400">Supervised Runtimes</div>
          <div className="mt-2 text-2xl font-bold font-mono text-emerald-400">{activeCount}</div>
          <div className="mt-1 text-[11px] text-slate-500">Active and managed</div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <div className="text-xs font-mono text-slate-400">Available Providers</div>
          <div className="mt-2 text-2xl font-bold font-mono text-sky-400">{installedCount} / {providers.length}</div>
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

      {/* Supervised Runtimes Section */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <h2 className="text-xs font-bold font-mono text-slate-200 uppercase tracking-wider">
              Live Runtimes
            </h2>
            {onCleanInactive && runtimes.some((r) => r.state === 'STOPPED' || r.state === 'FAILED' || r.state === 'STALE') && (
              <button
                onClick={onCleanInactive}
                className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-slate-200 text-[10px] font-mono transition"
              >
                Clear Inactive
              </button>
            )}
          </div>
          <span className="text-xs font-mono text-slate-500">{runtimes.length} total sessions</span>
        </div>

        {runtimes.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-xs font-mono">
            No active runtimes. Start one with "Launch Agent" or from the CLI with `ai control start &lt;provider&gt;`.
          </div>
        ) : (
          <div className="divide-y divide-slate-800">
            {runtimes.map((r) => {
              const isRunning = r.state === 'RUNNING' || r.state === 'STARTING';
              return (
                <div key={r.runtime_id} className="p-4 flex items-center justify-between hover:bg-slate-800/30 transition">
                  <div className="flex items-center space-x-3">
                    <div
                      className={`w-3 h-3 rounded-full ${
                        isRunning ? 'bg-emerald-500 animate-pulse' : 'bg-slate-600'
                      }`}
                    />
                    <div>
                      <div className="flex items-center space-x-2">
                        <span className="font-bold text-sm text-slate-100 uppercase font-mono">
                          {r.provider}
                        </span>
                        <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                          {r.profile}
                        </span>
                        <span className="text-xs font-mono text-slate-500">ID: {r.runtime_id}</span>
                        {r.handoff_type && (
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-950 text-indigo-300 border border-indigo-800 font-mono">
                            {r.handoff_type}
                          </span>
                        )}
                        {!isRunning && (
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
                            {r.state}
                          </span>
                        )}
                      </div>
                      <div className="mt-1 text-xs text-slate-400 font-mono truncate max-w-md">
                        {r.workspace}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    {isRunning ? (
                      <button
                        onClick={() => onOpenTerminal(r.runtime_id)}
                        className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded text-xs font-medium transition flex items-center space-x-1.5"
                      >
                        <Cpu className="w-3.5 h-3.5" />
                        <span>Terminal</span>
                      </button>
                    ) : (
                      <span className="px-2 py-1 text-[11px] font-mono text-slate-500 bg-slate-950 rounded border border-slate-800">
                        OFFLINE
                      </span>
                    )}

                    {isRunning && (
                      <>
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
                      </>
                    )}

                    {!isRunning && onDeleteRuntime && (
                      <button
                        onClick={() => onDeleteRuntime(r.runtime_id)}
                        className="p-1.5 text-slate-500 hover:text-rose-400 hover:bg-slate-800 rounded transition"
                        title="Delete Stale Record"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
