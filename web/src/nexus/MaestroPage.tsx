import React, { useEffect, useState } from 'react';
import { nexus } from './api';
import { Button, Badge, Card, Spinner, EmptyState } from '../ui/primitives';

interface MaestroStatus {
  available: boolean;
  mode: string;
  capabilities?: {
    version: string;
    modes: string[];
    skills: string[];
  };
  error?: string;
}

interface Recommendation {
  id: string;
  type: string;
  title: string;
  description: string;
  apply: string;
  why: string;
  risk: string;
}

interface Props {
  projectId: string;
}

export const MaestroPage: React.FC<Props> = ({ projectId }) => {
  const [status, setStatus] = useState<MaestroStatus | null>(null);
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [advising, setAdvising] = useState(false);

  useEffect(() => {
    nexus
      .getMaestroStatus()
      .then(setStatus)
      .catch(() => setStatus({ available: false, mode: 'OFF', error: 'failed to check status' }))
      .finally(() => setLoading(false));
  }, []);

  const requestAdvice = async () => {
    setAdvising(true);
    try {
      const resp = await nexus.getMaestroAdvice(projectId, undefined, 'general review');
      if (resp.degraded) {
        setRecommendations([]);
        return;
      }
      const all = [
        ...(resp.required || []),
        ...(resp.recommended || []),
        ...(resp.optional || []),
      ];
      setRecommendations(all);
    } catch {
      setRecommendations([]);
    } finally {
      setAdvising(false);
    }
  };

  if (loading) return <Spinner label="Checking Maestro…" />;

  const degraded = !status?.available;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-100">Maestro Assist</h2>
          <p className="text-xs text-slate-500">
            AI-powered project guidance and recommendations
          </p>
        </div>
        <Badge tone={degraded ? 'warning' : 'success'}>
          {degraded ? 'DEGRADED' : status?.mode || 'OFF'}
        </Badge>
      </div>

      {degraded ? (
        <Card className="p-4">
          <div className="flex items-center gap-2 mb-2">
            <Badge tone="warning">Maestro Unavailable</Badge>
          </div>
          <p className="text-sm text-slate-400">
            {status?.error || 'Maestro binary not found. Install the Orquestrador to enable AI-assisted project guidance.'}
          </p>
          <p className="text-xs text-slate-600 mt-2">
            Agents, terminals, and runtime control remain fully functional without Maestro.
          </p>
        </Card>
      ) : (
        <>
          <Card className="p-4">
            <div className="flex items-center gap-2 mb-2">
              <Badge tone="success">Maestro Connected</Badge>
              <Badge tone="default">v{status?.capabilities?.version || '?'}</Badge>
            </div>
            <p className="text-sm text-slate-400">
              Mode: {status?.mode} · Skills: {status?.capabilities?.skills?.length || 0}
            </p>
          </Card>

          <div className="flex gap-2">
            <Button tone="brand" onClick={requestAdvice} disabled={advising}>
              {advising ? 'Analyzing…' : 'Request Advice'}
            </Button>
          </div>

          {recommendations.length > 0 && (
            <div className="flex flex-col gap-2">
              <h3 className="text-sm font-semibold text-slate-200">Recommendations</h3>
              {recommendations.map((rec) => (
                <Card key={rec.id} className="p-3">
                  <div className="flex items-center gap-2 mb-1">
                    <Badge tone={rec.risk === 'high' ? 'danger' : rec.risk === 'medium' ? 'warning' : 'success'}>
                      {rec.risk}
                    </Badge>
                    <span className="text-sm font-medium text-slate-100">{rec.title}</span>
                  </div>
                  <p className="text-xs text-slate-400">{rec.description}</p>
                  {rec.why && <p className="text-[11px] text-slate-600 mt-1">Why: {rec.why}</p>}
                </Card>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
};
