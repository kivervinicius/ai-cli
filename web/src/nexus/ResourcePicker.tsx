import React, { useEffect, useState } from 'react';
import { nexus } from './api';
import { Button, Badge, Spinner } from '../ui/primitives';

interface ProviderAccount {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  authenticated: boolean;
  is_default: boolean;
  quota_remaining: number;
  quota_total: number;
  rate_limited: boolean;
  health: string;
}

interface SchedulerDecision {
  selected: ProviderAccount | null;
  policy: string;
  reason: string;
  score: number;
  rejected: { account: ProviderAccount; reason: string }[];
  explain_path: string[];
}

interface Props {
  preferProvider?: string;
  onSelect: (provider: string, profile: string) => void;
}

const healthTone = (h: string) =>
  h === 'healthy' ? 'success' : h === 'degraded' ? 'warning' : h === 'unhealthy' ? 'danger' : 'default';

export const ResourcePicker: React.FC<Props> = ({ preferProvider, onSelect }) => {
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [decision, setDecision] = useState<SchedulerDecision | null>(null);
  const [loading, setLoading] = useState(true);
  const [selecting, setSelecting] = useState(false);

  useEffect(() => {
    nexus
      .listResources()
      .then((data) => {
        setAccounts(data.accounts || []);
        if (preferProvider) {
          return nexus.selectResource(preferProvider, 'BALANCED');
        }
      })
      .then((data) => {
        if (data) setDecision(data.decision);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [preferProvider]);

  const handleSelect = async (provider: string, profile: string) => {
    setSelecting(true);
    try {
      const res = await nexus.selectResource(provider, 'BALANCED');
      setDecision(res.decision);
      onSelect(provider, profile);
    } finally {
      setSelecting(false);
    }
  };

  if (loading) return <Spinner label="Loading providers…" />;

  if (accounts.length === 0) {
    return (
      <div className="text-xs text-slate-500 py-2">
        No provider accounts available. Install and authenticate a provider first.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1.5">
        {accounts.map((acc) => (
          <div
            key={acc.id}
            className={`flex items-center justify-between gap-2 rounded-[var(--nx-radius-sm)] border px-3 py-2 cursor-pointer transition ${
              decision?.selected?.id === acc.id
                ? 'border-indigo-600 bg-indigo-950/30'
                : 'border-slate-800 bg-slate-900/40 hover:border-slate-700'
            }`}
            onClick={() => handleSelect(acc.provider, acc.profile)}
          >
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                <span className="text-sm font-medium text-slate-100">{acc.display_name || acc.provider}</span>
                <Badge tone="default">{acc.profile}</Badge>
                <Badge tone={healthTone(acc.health)}>{acc.health}</Badge>
                {acc.rate_limited && <Badge tone="danger">rate limited</Badge>}
              </div>
              <div className="text-[11px] text-slate-500 mt-0.5">
                {acc.provider}/{acc.profile}
                {acc.quota_total > 0 && (
                  <span className="ml-2">
                    quota: {Math.round((acc.quota_remaining / acc.quota_total) * 100)}%
                  </span>
                )}
              </div>
            </div>
            {decision?.selected?.id === acc.id && (
              <Badge tone="brand">selected</Badge>
            )}
          </div>
        ))}
      </div>

      {decision && decision.explain_path.length > 0 && (
        <div className="text-[11px] text-slate-600 mt-1">
          <span className="text-slate-500">Decision:</span> {decision.reason}
        </div>
      )}
    </div>
  );
};
