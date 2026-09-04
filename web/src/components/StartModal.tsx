import React, { useState } from 'react';
import { ProviderInfo, ProfileInfo, RuntimeSession, Workspace } from '../types';
import { api } from '../api';
import { Play, FolderGit2 } from 'lucide-react';
import { Button, Dialog } from '../design-system';

interface StartModalProps {
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  workspace: string;
  workspaces: Workspace[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
  open?: boolean;
}

export const StartModal: React.FC<StartModalProps> = ({
  providers,
  profiles,
  workspace,
  workspaces,
  onClose,
  onSuccess,
  open = true,
}) => {
  const installedProviders = providers.filter((p) => p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    installedProviders.length > 0 ? installedProviders[0].id : 'agy'
  );

  const availableProfiles = profiles.filter((p) => p.provider === selectedProvider);
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : 'default'
  );

  const [selectedWorkspace, setSelectedWorkspace] = useState<string>(workspace || (workspaces.length > 0 ? workspaces[0].path : ''));
  const [isCustomWorkspace, setIsCustomWorkspace] = useState(false);
  const [customWorkspace, setCustomWorkspace] = useState('');
  const [sessionTitle, setSessionTitle] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleProviderChange = (prov: string) => {
    setSelectedProvider(prov);
    const profs = profiles.filter((p) => p.provider === prov);
    setSelectedProfile(profs.length > 0 ? profs[0].name : 'default');
  };

  const handleWorkspaceSelect = (val: string) => {
    if (val === '__custom__') {
      setIsCustomWorkspace(true);
    } else {
      setIsCustomWorkspace(false);
      setSelectedWorkspace(val);
    }
  };

  const handleStart = async () => {
    setLoading(true);
    setError('');
    const targetWs = isCustomWorkspace ? customWorkspace.trim() : selectedWorkspace;
    if (!targetWs) {
      setError('Workspace path cannot be empty');
      setLoading(false);
      return;
    }

    try {
      const res = await api.startRuntime(
        selectedProvider,
        selectedProfile,
        targetWs,
        [],
        sessionTitle.trim() || undefined
      );
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to start runtime');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Launch Agent Runtime">
      <div className="space-y-4 font-mono">
        <div className="space-y-3 text-xs">
          {/* Target Project / Workspace */}
          <div>
            <label className="text-slate-400 flex items-center space-x-1.5">
              <FolderGit2 className="w-3.5 h-3.5 text-sky-400" />
              <span>Target Project / Workspace:</span>
            </label>
            <select
              value={isCustomWorkspace ? '__custom__' : selectedWorkspace}
              onChange={(e) => handleWorkspaceSelect(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 focus:outline-none focus:border-sky-500"
            >
              {workspaces.map((ws) => (
                <option key={ws.path} value={ws.path}>
                  {ws.name} ({ws.path})
                </option>
              ))}
              <option value="__custom__">+ Enter Custom Path...</option>
            </select>
            {isCustomWorkspace && (
              <input
                type="text"
                placeholder="/absolute/path/to/project"
                value={customWorkspace}
                onChange={(e) => setCustomWorkspace(e.target.value)}
                className="mt-2 w-full bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-slate-100 focus:outline-none focus:border-sky-500"
                autoFocus
              />
            )}
          </div>

          {/* Session Title / Goal */}
          <div>
            <label className="text-slate-400">Session Title / Objective (Optional):</label>
            <input
              type="text"
              placeholder="e.g. Refactor Auth, Debug Database..."
              value={sessionTitle}
              onChange={(e) => setSessionTitle(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500"
            />
          </div>

          {/* Coding Provider */}
          <div>
            <label className="text-slate-400">Coding Provider:</label>
            <select
              value={selectedProvider}
              onChange={(e) => handleProviderChange(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 uppercase focus:outline-none focus:border-sky-500"
            >
              {installedProviders.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.id} {p.version ? `(${p.version})` : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Profile / Account */}
          <div>
            <label className="text-slate-400">Profile / Account:</label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 focus:outline-none focus:border-sky-500"
            >
              {availableProfiles.length > 0 ? (
                availableProfiles.map((p) => (
                  <option key={p.name} value={p.name}>
                    {p.name} {p.account_email ? `(${p.account_email})` : ''}
                  </option>
                ))
              ) : (
                <option value="default">default</option>
              )}
            </select>
          </div>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/50 border border-rose-800 rounded text-rose-300 text-xs">
            {error}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>Cancel</Button>
          <Button
            tone="brand"
            disabled={loading || installedProviders.length === 0}
            onClick={handleStart}
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>{loading ? 'Starting...' : 'Start Runtime'}</span>
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
