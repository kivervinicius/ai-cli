import React, { useState } from 'react';
import { RuntimeSession, ProfileInfo } from '../types';
import { api } from '../api';
import { Button, Dialog } from '../design-system';

interface HandoffModalProps {
  runtime: RuntimeSession;
  profiles: ProfileInfo[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
  open?: boolean;
}

export const HandoffModal: React.FC<HandoffModalProps> = ({
  runtime,
  profiles,
  onClose,
  onSuccess,
  open = true,
}) => {
  const runtimeProvider = runtime.provider_id || runtime.provider || '';
  const runtimeProfile = runtime.profile_id || runtime.profile || '';
  const availableProfiles = profiles.filter(
    (p) => p.provider === runtimeProvider && p.name !== runtimeProfile
  );
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : ''
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleHandoff = async () => {
    if (!selectedProfile) return;
    setLoading(true);
    setError('');
    try {
      const target = `${runtimeProvider}:${selectedProfile}`;
      const res = await api.accountHandoff(runtime.runtime_id, target);
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Account handoff failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Account Handoff">
      <div className="space-y-4 font-mono">
        <div className="text-xs text-slate-300 space-y-2">
          <div className="bg-slate-950 p-3 rounded border border-slate-800">
            <div><span className="text-slate-500">Source:</span> <span className="uppercase font-bold text-slate-200">{runtimeProvider}</span> ({runtimeProfile})</div>
            <div className="mt-1"><span className="text-slate-500">Runtime:</span> {runtime.runtime_id}</div>
          </div>
          <p className="text-[11px] text-slate-400">
            Switches active execution to another account while resuming the exact same conversation thread.
          </p>
        </div>

        {availableProfiles.length === 0 ? (
          <div className="p-4 bg-amber-950/40 border border-amber-800/60 rounded text-amber-300 text-xs">
            No alternative profiles found for provider {runtimeProvider}. Add more accounts via <code>nexus add {runtimeProvider}</code>.
          </div>
        ) : (
          <div className="space-y-2 text-xs">
            <label className="text-xs text-slate-400">Target Profile:</label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-sky-500"
            >
              {availableProfiles.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name} {p.account_email ? `(${p.account_email})` : ''} {p.plan ? `[${p.plan}]` : ''}
                </option>
              ))}
            </select>
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
            disabled={loading || availableProfiles.length === 0}
            onClick={handleHandoff}
          >
            {loading ? 'Performing Handoff...' : 'Execute Handoff'}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
