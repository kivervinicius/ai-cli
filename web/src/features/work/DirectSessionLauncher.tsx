import React, { useEffect, useMemo, useState } from 'react';
import { Bot, Play, ShieldAlert, TerminalSquare } from 'lucide-react';
import { Badge, Button, Dialog, EmptyState, Input, InlineAlert, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, Project } from '../../types';
import { buildDirectAgentName, eligibleDirectResources } from './directSessionModel';

interface DirectResource {
  id: string;
  provider: string;
  profile: string;
  display_name?: string;
  authenticated: boolean;
  available: boolean;
  health?: string;
  rate_limited?: boolean;
  quota_remaining?: number;
  avail_reasons?: { unknown_quota?: boolean; exhausted_windows?: string[]; rate_limited?: boolean };
  quota_view?: { status?: string; model_groups?: Array<{ windows?: Array<{ remaining?: number }> }> };
}

export interface DirectSessionRequest {
  mode: 'direct' | 'assisted';
  prompt: string;
}

export const DirectSessionLauncher: React.FC<{
  open: boolean;
  project: Project;
  request: DirectSessionRequest | null;
  onClose: () => void;
  onStarted: (agent: Agent, prompt: string) => void | Promise<void>;
  refreshAgents: () => Promise<void>;
}> = ({ open, project, request, onClose, onStarted, refreshAgents }) => {
  const [resources, setResources] = useState<DirectResource[]>([]);
  const [selectedID, setSelectedID] = useState('');
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) return;
    let mounted = true;
    setLoading(true);
    setError('');
    void nexus.listResources()
      .then((result) => {
        if (!mounted) return;
        const eligible = eligibleDirectResources((result.accounts || []) as DirectResource[]);
        setResources(eligible);
        setSelectedID(eligible[0]?.id || '');
        if (eligible[0]) setName(buildDirectAgentName(request?.prompt || '', eligible[0].provider));
      })
      .catch((err) => mounted && setError(err instanceof Error ? err.message : String(err)))
      .finally(() => mounted && setLoading(false));
    return () => { mounted = false; };
  }, [open, request?.prompt]);

  const selected = useMemo(() => resources.find((resource) => resource.id === selectedID) || null, [resources, selectedID]);

  useEffect(() => {
    if (!selected || name.trim()) return;
    setName(buildDirectAgentName(request?.prompt || '', selected.provider));
  }, [selected, request?.prompt, name]);

  const choose = (resource: DirectResource) => {
    setSelectedID(resource.id);
    setName(buildDirectAgentName(request?.prompt || '', resource.provider));
  };

  const start = async () => {
    if (!selected || !request) return;
    setStarting(true);
    setError('');
    try {
      const agent = await nexus.createAgent(project.id, name.trim() || buildDirectAgentName(request.prompt, selected.provider), 'developer');
      await nexus.selectResource(agent.id, selected.provider, selected.profile, 'MANUAL');
      await nexus.startAgent(agent.id);
      await refreshAgents();
      await onStarted(agent, request.prompt);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setStarting(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title={request?.mode === 'assisted' ? 'Start assisted AI session' : 'Start direct AI session'} wide>
      <div className="nx-form-stack">
        <InlineAlert tone={request?.mode === 'assisted' ? 'info' : 'success'} title={request?.mode === 'assisted' ? 'Assisted work remains optional' : 'No Mission required'}>
          {request?.mode === 'assisted'
            ? 'The selected provider session remains directly controllable. This launcher does not silently rewrite the prompt; use AI Planning when you want Intelligence to analyze and structure the work.'
            : 'This creates a Persistent Agent and starts the selected coding CLI directly inside this Project.'}
        </InlineAlert>

        <Input value={name} onChange={setName} placeholder="Agent/session name" aria-label="Agent/session name" />

        {loading ? <Spinner label="Loading provider accounts…" /> : resources.length === 0 ? (
          <EmptyState
            icon={<Bot size={22} />}
            title="No usable AI provider account"
            hint="Install and authenticate Codex, Claude, Gemini, AGY, OpenCode or another supported provider before starting a direct session."
          />
        ) : (
          <div className="nx-direct-resource-list" role="radiogroup" aria-label="Choose AI provider and profile">
            {resources.map((resource) => {
              const checked = resource.id === selectedID;
              const quotaKnown = !resource.avail_reasons?.unknown_quota && resource.quota_view?.status !== 'UNKNOWN';
              return (
                <button
                  type="button"
                  role="radio"
                  aria-checked={checked}
                  data-selected={checked ? 'true' : 'false'}
                  className="nx-direct-resource"
                  key={resource.id}
                  onClick={() => choose(resource)}
                >
                  <span className="nx-direct-resource__icon">{resource.rate_limited ? <ShieldAlert size={17} /> : <TerminalSquare size={17} />}</span>
                  <span className="nx-direct-resource__body">
                    <span className="nx-direct-resource__title">
                      <strong>{resource.display_name || resource.provider}</strong>
                      <Badge>{resource.provider}</Badge>
                      <Badge>{resource.profile}</Badge>
                      {resource.health && <Badge tone={resource.health === 'healthy' ? 'success' : resource.health === 'degraded' ? 'warning' : 'default'}>{resource.health}</Badge>}
                    </span>
                    <small>{quotaKnown && Number.isFinite(resource.quota_remaining) ? `${Math.round((resource.quota_remaining ?? 0) * ((resource.quota_remaining ?? 0) <= 1 ? 100 : 1))}% available` : 'Quota UNKNOWN — never assumed as best'}</small>
                  </span>
                </button>
              );
            })}
          </div>
        )}

        {request?.prompt && (
          <div className="nx-direct-prompt-preview">
            <strong>Initial prompt</strong>
            <pre>{request.prompt}</pre>
          </div>
        )}
        {error && <InlineAlert tone="danger" title="Could not start session">{error}</InlineAlert>}
        <div className="nx-dialog-actions">
          <Button onClick={onClose}>Cancel</Button>
          <Button tone="brand" disabled={!selected || starting || !name.trim()} onClick={() => void start()}>
            <Play size={14} /> {starting ? 'Starting…' : 'Start AI session'}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
