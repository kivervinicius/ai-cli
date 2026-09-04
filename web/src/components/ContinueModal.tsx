import React, { useState } from 'react';
import { RuntimeSession, ProviderInfo, ProfileInfo } from '../types';
import { api } from '../api';
import { AlertTriangle } from 'lucide-react';
import { Button, Dialog } from '../design-system';

interface ContinueModalProps {
  runtime: RuntimeSession;
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
  open?: boolean;
}

export const ContinueModal: React.FC<ContinueModalProps> = ({
  runtime,
  providers,
  profiles,
  onClose,
  onSuccess,
  open = true,
}) => {
  const runtimeProvider = runtime.provider_id || runtime.provider;
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
    <Dialog open={open} onClose={onClose} title="Continue with Another AI">
      <div className="space-y-4 font-mono">
        <div className="p-3 bg-indigo-950/40 border border-indigo-800/60 rounded flex items-start space-x-2.5 text-xs text-indigo-300">
          <AlertTriangle className="w-4 h-4 text-indigo-400 flex-shrink-0 mt-0.5" />
          <div>
            <strong>A NEW SESSION WILL BE CREATED.</strong>
            <p className="mt-1 text-[11px] text-indigo-300/80">
              Captures current git status, diff stats, and modified files with sensitive secrets redacted, and boots a clean agent thread in the target CLI.
            </p>
          </div>
        </div>

        {otherProviders.length === 0 ? (
          <div className="p-4 bg-amber-950/40 border border-amber-800/60 rounded text-amber-300 text-xs">
            No other installed coding CLIs detected. Install Claude, Codex, Gemini, or OpenCode to continue across providers.
          </div>
        ) : (
          <div className="space-y-3 text-xs">
            <div>
              <label className="text-xs text-slate-400">Destination Provider:</label>
              <select
                value={selectedProvider}
                onChange={(e) => handleProviderChange(e.target.value)}
                className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500 uppercase"
              >
                {otherProviders.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.id} {p.version ? `(${p.version})` : ''}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs text-slate-400">Profile / Account:</label>
              <select
                value={selectedProfile}
                onChange={(e) => setSelectedProfile(e.target.value)}
                className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
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
          <div className="p-3 bg-rose-950/50 border border-rose-800 rounded text-rose-300 text-xs">
            {error}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>Cancel</Button>
          <Button
            tone="brand"
            disabled={loading || otherProviders.length === 0}
            onClick={handleContinue}
          >
            {loading ? 'Continuing...' : 'Launch Target Session'}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
