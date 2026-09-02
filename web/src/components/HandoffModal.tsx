import React, { useState } from 'react';
import { RuntimeSession, ProfileInfo } from '../types';
import { api } from '../api';
import { X, ArrowRightLeft } from 'lucide-react';

interface HandoffModalProps {
  runtime: RuntimeSession;
  profiles: ProfileInfo[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
}

export const HandoffModal: React.FC<HandoffModalProps> = ({
  runtime,
  profiles,
  onClose,
  onSuccess,
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
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
        <div className="flex items-center justify-between border-b border-slate-800 pb-3">
          <div className="flex items-center space-x-2">
            <ArrowRightLeft className="w-5 h-5 text-sky-400" />
            <h3 className="text-sm font-bold text-slate-100 uppercase tracking-wider font-mono">
              Account Handoff
            </h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="text-xs text-slate-300 space-y-2 font-mono">
          <div className="bg-slate-950 p-3 rounded border border-slate-800">
            <div><span className="text-slate-500">Source:</span> <span className="uppercase font-bold text-slate-200">{runtimeProvider}</span> ({runtimeProfile})</div>
            <div className="mt-1"><span className="text-slate-500">Runtime:</span> {runtime.runtime_id}</div>
          </div>
          <p className="text-[11px] text-slate-400">
            Switches active execution to another account while resuming the exact same conversation thread.
          </p>
        </div>

        {availableProfiles.length === 0 ? (
          <div className="p-4 bg-amber-950/40 border border-amber-800/60 rounded text-amber-300 text-xs font-mono">
            No alternative profiles found for provider {runtimeProvider}. Add more accounts via <code>nexus add {runtimeProvider}</code>.
          </div>
        ) : (
          <div className="space-y-2">
            <label className="text-xs font-mono text-slate-400">Target Profile:</label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-sky-500"
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
            disabled={loading || availableProfiles.length === 0}
            onClick={handleHandoff}
            className="px-4 py-1.5 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded text-xs font-semibold shadow-md shadow-sky-900/30 transition"
          >
            {loading ? 'Performing Handoff...' : 'Execute Handoff'}
          </button>
        </div>
      </div>
    </div>
  );
};
