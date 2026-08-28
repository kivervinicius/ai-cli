import React, { useState } from 'react';
import { RuntimeSession, ProviderInfo, ProfileInfo } from '../types';
import { api } from '../api';
import { X, FastForward, AlertTriangle } from 'lucide-react';

interface ContinueModalProps {
  runtime: RuntimeSession;
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
}

export const ContinueModal: React.FC<ContinueModalProps> = ({
  runtime,
  providers,
  profiles,
  onClose,
  onSuccess,
}) => {
  const runtimeProvider = runtime.provider_id || runtime.provider;
  const runtimeProfile = runtime.profile_id || runtime.profile;
  const otherProviders = providers.filter((p) => p.id !== runtimeProvider && p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    otherProviders.length > 0 ? otherProviders[0].id : ''
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

  const handleContinue = async () => {
    if (!selectedProvider) return;
    setLoading(true);
    setError('');
    try {
      const res = await api.contextContinue(runtime.runtime_id, selectedProvider, selectedProfile);
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Context continue failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
        <div className="flex items-center justify-between border-b border-slate-800 pb-3">
          <div className="flex items-center space-x-2">
            <FastForward className="w-5 h-5 text-emerald-400" />
            <h3 className="text-sm font-bold text-slate-100 uppercase tracking-wider font-mono">
              Continue with Another AI
            </h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-3 bg-indigo-950/40 border border-indigo-800/60 rounded flex items-start space-x-2.5 text-xs text-indigo-300 font-mono">
          <AlertTriangle className="w-4 h-4 text-indigo-400 flex-shrink-0 mt-0.5" />
          <div>
            <strong>A NEW SESSION WILL BE CREATED.</strong>
            <p className="mt-1 text-[11px] text-indigo-300/80">
              Captures current git status, diff stats, and modified files with sensitive secrets redacted, and boots a clean agent thread in the target CLI.
            </p>
          </div>
        </div>

        {otherProviders.length === 0 ? (
          <div className="p-4 bg-amber-950/40 border border-amber-800/60 rounded text-amber-300 text-xs font-mono">
            No other installed coding CLIs detected. Install Claude, Codex, Gemini, or OpenCode to continue across providers.
          </div>
        ) : (
          <div className="space-y-3">
            <div>
              <label className="text-xs font-mono text-slate-400">Destination Provider:</label>
              <select
                value={selectedProvider}
                onChange={(e) => handleProviderChange(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500 uppercase"
              >
                {otherProviders.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.id} {p.version ? `(${p.version})` : ''}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs font-mono text-slate-400">Profile / Account:</label>
              <select
                value={selectedProfile}
                onChange={(e) => setSelectedProfile(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
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
        )}

        {error && (
          <div className="p-3 bg-rose-950/50 border border-rose-800 rounded text-rose-300 text-xs font-mono">
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
            disabled={loading || otherProviders.length === 0}
            onClick={handleContinue}
            className="px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded text-xs font-semibold shadow-md shadow-emerald-900/30 transition"
          >
            {loading ? 'Continuing...' : 'Launch Target Session'}
          </button>
        </div>
      </div>
    </div>
  );
};
