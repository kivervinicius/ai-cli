import React, { useEffect, useMemo, useState } from 'react';
import { Bot, Play, ShieldAlert, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Dialog, EmptyState, Input, InlineAlert, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, Project } from '../../types';
import { buildDirectAgentName, directAccountTitle, directQuotaDisplay, eligibleDirectResources } from './directSessionModel';

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
  quota_view?: {
    status?: string;
    fetched_at?: string;
    model_groups?: Array<{ key?: string; name?: string; windows?: Array<{ kind?: string; remaining?: number }> }>;
  };
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
  const { t } = useTranslation();
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
      await nexus.applyAgentConfig(agent.id, {
        provider: selected.provider,
        profile: selected.profile,
        isolation: 'project',
        workspace: project.canonical_path || undefined,
      });
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

  const healthBadge = (health?: string) => {
    if (health === 'healthy') return <Badge tone="success">{t('overview.healthy')}</Badge>;
    if (health === 'degraded') return <Badge tone="warning">{t('status.DEGRADED')}</Badge>;
    return null;
  };

  return (
    <Dialog open={open} onClose={onClose} title={request?.mode === 'assisted' ? t('directSession.assistedTitle') : t('directSession.title')} wide>
      <div className="nx-form-stack">
        <Input value={name} onChange={setName} placeholder={t('directSession.namePlaceholder')} aria-label={t('directSession.name')} />

        {loading ? <Spinner label={t('directSession.loading')} /> : resources.length === 0 ? (
          <EmptyState
            icon={<Bot size={22} />}
            title={t('directSession.empty')}
            hint={t('directSession.emptyHint')}
          />
        ) : (
          <div className="nx-direct-resource-list" role="radiogroup" aria-label={t('directSession.chooseAccount')}>
            {resources.map((resource) => {
              const checked = resource.id === selectedID;
              const quota = directQuotaDisplay(resource);
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
                      <strong>{directAccountTitle(resource)}</strong>
                      {healthBadge(resource.health)}
                    </span>
                    <small>{quota === null ? t('directSession.quotaUnknown') : t('directSession.remainingLabel', { value: quota })}</small>
                  </span>
                </button>
              );
            })}
          </div>
        )}

        {request?.prompt && (
          <div className="nx-direct-prompt-preview">
            <strong>{t('directSession.prompt')}</strong>
            <pre>{request.prompt}</pre>
          </div>
        )}
        {error && <InlineAlert tone="danger" title={t('directSession.startError')}>{error}</InlineAlert>}
        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('directSession.cancel')}</Button>
          <Button tone="brand" disabled={!selected || starting || !name.trim()} onClick={() => void start()}>
            <Play size={14} /> {starting ? t('directSession.starting') : t('directSession.start')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
