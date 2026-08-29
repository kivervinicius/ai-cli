import React, { useEffect, useState } from 'react';
import { nexus } from './api';
import { Agent, AgentConfig, ConfigImpact } from '../types';
import { Drawer, Button, Badge, Input, Select, Spinner } from '../ui/primitives';

interface Props {
  agent: Agent;
  open: boolean;
  onClose: () => void;
  onApplied: () => void;
}

const PROVIDERS = [
  { value: 'claude', label: 'Claude (Anthropic)' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini (Google)' },
  { value: 'cursor', label: 'Cursor' },
];

const ISOLATION_MODES = [
  { value: '', label: 'Default (project)' },
  { value: 'project', label: 'Project isolation' },
  { value: 'global', label: 'Global' },
  { value: 'none', label: 'None' },
];

const MAESTRO_MODES = [
  { value: '', label: 'Inherit from project' },
  { value: 'OFF', label: 'Off' },
  { value: 'ASSIST', label: 'Assist' },
  { value: 'ORCHESTRATE', label: 'Orchestrate (beta)' },
];

const CONTINUITY_POLICIES = [
  { value: '', label: 'Auto' },
  { value: 'auto', label: 'Auto' },
  { value: 'native', label: 'Native resume only' },
  { value: 'new_session', label: 'Always new session' },
];

const impactTone = (mode: string) =>
  mode === 'NEW_SESSION' ? 'warning' : mode === 'RESTART_RUNTIME' ? 'info' : 'success';

const impactLabel = (mode: string) =>
  mode === 'NEW_SESSION'
    ? 'New provider session required'
    : mode === 'RESTART_RUNTIME'
      ? 'Runtime restart required'
      : 'Applied live';

export const AgentConfigurationDrawer: React.FC<Props> = ({ agent, open, onClose, onApplied }) => {
  const [config, setConfig] = useState<AgentConfig>({ provider: '', profile: 'default' });
  const [impact, setImpact] = useState<ConfigImpact | null>(null);
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);

  useEffect(() => {
    if (!open || !agent.id) return;
    setLoading(true);
    nexus
      .getAgentConfig(agent.id)
      .then((data) => {
        setConfig(data.config || { provider: '', profile: 'default' });
        setImpact(null);
        setError('');
      })
      .catch((e: any) => setError(e.message))
      .finally(() => setLoading(false));
  }, [open, agent.id]);

  const previewImpact = async () => {
    try {
      const res = await nexus.previewAgentConfig(agent.id, config);
      setImpact(res.impact);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const applyConfig = async () => {
    setApplying(true);
    setError('');
    try {
      const res = await nexus.applyAgentConfig(agent.id, config);
      setImpact(res.impact);
      onApplied();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setApplying(false);
    }
  };

  const update = <K extends keyof AgentConfig>(key: K, value: AgentConfig[K]) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
    setImpact(null); // clear stale preview
  };

  return (
    <Drawer open={open} onClose={onClose} title={`Configure — ${agent.name}`} wide>
      {loading ? (
        <Spinner label="Loading configuration…" />
      ) : (
        <div className="flex flex-col gap-5">
          {error && <p className="text-xs text-red-400 bg-red-950/30 rounded px-3 py-2">{error}</p>}

          {/* Provider & Profile */}
          <section>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Provider</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[11px] text-slate-500 mb-1 block">Provider</label>
                <Select
                  value={config.provider}
                  onChange={(v) => update('provider', v)}
                  options={PROVIDERS}
                  placeholder="Select provider"
                />
              </div>
              <div>
                <label className="text-[11px] text-slate-500 mb-1 block">Profile</label>
                <Input
                  value={config.profile}
                  onChange={(v) => update('profile', v)}
                  placeholder="default"
                  mono
                />
              </div>
            </div>
          </section>

          {/* Model */}
          <section>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Model</h3>
            <Input
              value={config.model || ''}
              onChange={(v) => update('model', v || undefined)}
              placeholder="e.g. sonnet, gpt-4o, gemini-pro"
              mono
            />
            <p className="text-[11px] text-slate-600 mt-1">Leave empty for provider default.</p>
          </section>

          {/* Workspace */}
          <section>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Workspace</h3>
            <Input
              value={config.workspace || ''}
              onChange={(v) => update('workspace', v || undefined)}
              placeholder="Override project workspace path"
              mono
            />
          </section>

          {/* Isolation & Maestro */}
          <section>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Behavior</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[11px] text-slate-500 mb-1 block">Isolation</label>
                <Select
                  value={config.isolation || ''}
                  onChange={(v) => update('isolation', v || undefined)}
                  options={ISOLATION_MODES}
                />
              </div>
              <div>
                <label className="text-[11px] text-slate-500 mb-1 block">Maestro</label>
                <Select
                  value={config.maestro_mode || ''}
                  onChange={(v) => update('maestro_mode', v || undefined)}
                  options={MAESTRO_MODES}
                />
              </div>
            </div>
          </section>

          {/* Continuity */}
          <section>
            <label className="text-[11px] text-slate-500 mb-1 block">Continuity Policy</label>
            <Select
              value={config.continuity_policy || ''}
              onChange={(v) => update('continuity_policy', v || undefined)}
              options={CONTINUITY_POLICIES}
            />
          </section>

          {/* Advanced toggle */}
          <button
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="text-xs text-indigo-400 hover:text-indigo-300 text-left"
          >
            {showAdvanced ? '▾ Hide advanced' : '▸ Show advanced'}
          </button>

          {showAdvanced && (
            <>
              {/* Environment */}
              <section>
                <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Environment</h3>
                <Input
                  value={config.environment ? JSON.stringify(config.environment) : ''}
                  onChange={(v) => {
                    try {
                      update('environment', v ? JSON.parse(v) : undefined);
                    } catch {}
                  }}
                  placeholder='{"KEY":"value"}'
                  mono
                />
                <p className="text-[11px] text-slate-600 mt-1">JSON key-value pairs for environment variables.</p>
              </section>

              {/* Allocation */}
              <section>
                <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Allocation</h3>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] text-slate-500 mb-1 block">Prefer Provider</label>
                    <Select
                      value={config.allocation?.prefer_provider || ''}
                      onChange={(v) =>
                        update('allocation', { ...config.allocation, prefer_provider: v || undefined })
                      }
                      options={PROVIDERS}
                      placeholder="Any"
                    />
                  </div>
                  <div>
                    <label className="text-[11px] text-slate-500 mb-1 block">Max Concurrent</label>
                    <Input
                      value={String(config.allocation?.max_concurrent || '')}
                      onChange={(v) =>
                        update('allocation', {
                          ...config.allocation,
                          max_concurrent: v ? parseInt(v, 10) : undefined,
                        })
                      }
                      placeholder="Unlimited"
                    />
                  </div>
                </div>
              </section>
            </>
          )}

          {/* Impact preview */}
          {impact && (
            <div className="rounded-[var(--nx-radius-sm)] border border-slate-800 bg-slate-950/50 p-3">
              <div className="flex items-center gap-2 mb-1">
                <Badge tone={impactTone(impact.mode)}>{impactLabel(impact.mode)}</Badge>
              </div>
              {impact.changed_fields.length > 0 && (
                <p className="text-[11px] text-slate-400">
                  Changed: {impact.changed_fields.join(', ')}
                </p>
              )}
              {impact.warnings && impact.warnings.length > 0 && (
                <div className="mt-2">
                  {impact.warnings.map((w, i) => (
                    <p key={i} className="text-[11px] text-amber-400">
                      {w}
                    </p>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center gap-2 pt-2 border-t border-slate-800">
            <Button tone="default" onClick={previewImpact} disabled={!config.provider}>
              Preview Impact
            </Button>
            <Button tone="brand" onClick={applyConfig} disabled={!config.provider || applying}>
              {applying ? 'Applying…' : 'Safe Apply'}
            </Button>
            <div className="flex-1" />
            <Button tone="default" onClick={onClose}>
              Close
            </Button>
          </div>
        </div>
      )}
    </Drawer>
  );
};
