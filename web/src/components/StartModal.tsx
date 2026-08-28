import React, { useState } from 'react';
import { ProviderInfo, ProfileInfo, RuntimeSession } from '../types';
import { api } from '../api';
import { X, Play } from 'lucide-react';

interface StartModalProps {
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  workspace: string;
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
}

export const StartModal: React.FC<StartModalProps> = ({
  providers,
  profiles,
  workspace,
  onClose,
  onSuccess,
}) => {
  const installedProviders = providers.filter((p) => p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    installedProviders.length > 0 ? installedProviders[0].id : 'agy'
  );

  const availableProfiles = profiles.filter((p) => p.provider === selectedProvider);
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : 'default'
  );

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleProviderChange = (prov: string) => {
    setSelectedProvider(prov);
    const profs = profiles.filter((p) => p.provider === prov);
    setSelectedProfile(profs.length > 0 ? profs[0].name : 'default');
  };

  const handleStart = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.startRuntime(selectedProvider, selectedProfile, workspace);
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to start runtime');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4 font-mono">
        <div className="flex items-center justify-between border-b border-slate-800 pb-3">
          <div className="flex items-center space-x-2">
            <Play className="w-4 h-4 text-sky-400 fill-current" />
            <h3 className="text-sm font-bold text-slate-100 uppercase tracking-wider">
              Launch Agent Runtime
            </h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="space-y-3 text-xs">
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

          <div>
            <label className="text-slate-400">Workspace Path:</label>
            <div className="mt-1 p-2 bg-slate-950 border border-slate-800 rounded text-slate-300 truncate">
              {workspace}
            </div>
          </div>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/50 border border-rose-800 rounded text-rose-300 text-xs">
            {error}
          </div>
        )}

        <div className="flex items-center justify-end space-x-3 pt-3 border-t border-slate-800">
          <button
            onClick={onClose}
            className="px-3 py-1.5 rounded text-xs font-medium text-slate-400 hover:text-white"
          >
            Cancel
          </button>
          <button
            disabled={loading || installedProviders.length === 0}
            onClick={handleStart}
            className="px-4 py-1.5 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded text-xs font-semibold shadow-md shadow-sky-900/30 transition flex items-center space-x-1.5"
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>{loading ? 'Starting...' : 'Start Runtime'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
