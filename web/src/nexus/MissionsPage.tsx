import React, { useEffect, useState } from 'react';
import { nexus } from './api';
import { Button, Badge, Card, Spinner, EmptyState, Select } from '../ui/primitives';

interface Mission {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: string;
  goal: string;
  scope: string;
  risk_level: string;
  created_at: string;
  updated_at: string;
}

interface MissionTask {
  id: string;
  mission_id: string;
  name: string;
  description: string;
  status: string;
  kind: string;
  priority: number;
}

interface MissionStats {
  total: number;
  pending: number;
  active: number;
  completed: number;
  failed: number;
}

interface Props {
  projectId: string;
}

const statusTone = (s: string) =>
  s === 'COMPLETED' ? 'success' :
  s === 'ACTIVE' ? 'brand' :
  s === 'FAILED' ? 'danger' :
  s === 'CANCELLED' ? 'warning' :
  'default';

const kindTone = (k: string) =>
  k === 'verify' ? 'success' :
  k === 'security' ? 'danger' :
  k === 'config' ? 'warning' :
  'default';

export const MissionsPage: React.FC<Props> = ({ projectId }) => {
  const [missions, setMissions] = useState<Mission[]>([]);
  const [selected, setSelected] = useState<Mission | null>(null);
  const [tasks, setTasks] = useState<MissionTask[]>([]);
  const [stats, setStats] = useState<MissionStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');

  useEffect(() => {
    nexus.listMissions(projectId).then((data) => {
      setMissions(data.missions || []);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [projectId]);

  const selectMission = async (m: Mission) => {
    setSelected(m);
    try {
      const detail = await nexus.getMission(m.id);
      setTasks(detail.tasks || []);
      setStats(detail.stats || null);
    } catch {}
  };

  const createMission = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const m = await nexus.createMission(projectId, { name: newName, goal: newName });
      setMissions([m, ...missions]);
      setNewName('');
    } finally {
      setCreating(false);
    }
  };

  if (loading) return <Spinner label="Loading missions…" />;

  return (
    <div className="flex gap-4">
      <div className="w-72 flex-shrink-0">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-slate-200">Missions</h2>
          <Badge tone="warning">BETA</Badge>
        </div>
        <div className="flex gap-1.5 mb-3">
          <input
            className="flex-1 bg-slate-900 border border-slate-800 rounded px-2 py-1 text-xs text-slate-200 placeholder:text-slate-600"
            placeholder="New mission name…"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && createMission()}
          />
          <Button tone="brand" onClick={createMission} disabled={creating || !newName.trim()}>
            +
          </Button>
        </div>
        {missions.length === 0 ? (
          <EmptyState title="No missions" hint="Create one to get started." />
        ) : (
          <div className="flex flex-col gap-1">
            {missions.map((m) => (
              <div
                key={m.id}
                className={`p-2 rounded cursor-pointer transition ${
                  selected?.id === m.id ? 'bg-indigo-950/40 border border-indigo-700' : 'hover:bg-slate-800/50'
                }`}
                onClick={() => selectMission(m)}
              >
                <div className="flex items-center gap-1.5">
                  <span className="text-sm text-slate-100 truncate">{m.name}</span>
                  <Badge tone={statusTone(m.status)}>{m.status}</Badge>
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5">
                  {m.scope} · {m.risk_level}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex-1 min-w-0">
        {selected ? (
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <h3 className="text-lg font-semibold text-slate-100">{selected.name}</h3>
              <Badge tone={statusTone(selected.status)}>{selected.status}</Badge>
              <Badge tone="default">{selected.risk_level}</Badge>
            </div>
            {selected.description && (
              <p className="text-sm text-slate-400">{selected.description}</p>
            )}
            {selected.goal && (
              <Card className="p-3">
                <div className="text-xs text-slate-500 mb-1">Goal</div>
                <div className="text-sm text-slate-200">{selected.goal}</div>
              </Card>
            )}
            {stats && (
              <div className="flex gap-3">
                {(['total', 'pending', 'active', 'completed', 'failed'] as const).map((k) => (
                  <div key={k} className="text-center">
                    <div className="text-lg font-bold text-slate-100">{stats[k]}</div>
                    <div className="text-[11px] text-slate-500">{k}</div>
                  </div>
                ))}
              </div>
            )}
            {tasks.length > 0 && (
              <div className="flex flex-col gap-1.5">
                <h4 className="text-sm font-semibold text-slate-200">Tasks</h4>
                {tasks.map((t) => (
                  <div key={t.id} className="flex items-center gap-2 p-2 rounded bg-slate-900/40 border border-slate-800">
                    <Badge tone={kindTone(t.kind)}>{t.kind}</Badge>
                    <span className="text-sm text-slate-100">{t.name}</span>
                    <Badge tone={statusTone(t.status)}>{t.status}</Badge>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <EmptyState title="Select a mission" hint="View details and tasks." />
        )}
      </div>
    </div>
  );
};
