import React, { useEffect, useState } from 'react';
import { Activity, Check, Gauge, ShieldAlert } from 'lucide-react';
import { Badge, EmptyState, Progress, Spinner } from '../design-system';
import { nexus } from './api';
import { translateStatus } from '../i18n';
import { useTranslation } from 'react-i18next';

interface QuotaWindow {
  kind: string;
  label: string;
  remaining: number;
  reset_desc: string;
  status: string;
  bar: string;
}

interface QuotaModelGroup {
  name: string;
  windows: QuotaWindow[];
}

interface AvailReasons {
  exhausted_windows?: string[];
  rate_limited?: boolean;
  unknown_quota?: boolean;
  auth_required?: boolean;
  all_ok?: boolean;
}

interface QuotaView {
  provider: string;
  profile: string;
  account: string;
  plan: string;
  status: string;
  source: string;
  model_groups: QuotaModelGroup[];
  fetched_at: string;
  available: boolean;
  avail_reasons: AvailReasons;
}

interface ProviderAccount {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  authenticated: boolean;
  is_default: boolean;
  available: boolean;
  avail_reasons?: AvailReasons;
  quota_view?: QuotaView;
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
	agentId?: string;
	preferProvider?: string;
	onSelected?: (provider: string, profile: string) => void | Promise<void>;
}

const healthTone = (health: string) =>
  health === 'healthy' ? 'success' : health === 'degraded' ? 'warning' : health === 'unhealthy' ? 'danger' : 'default';

const QuotaBar: React.FC<{ w: QuotaWindow }> = ({ w }) => (
  <span className="nx-resource-account__quota">
    <span className="nx-resource-account__quota-label">{w.label}</span>
    <Progress value={w.remaining} label={`${Math.round(w.remaining)}%`} />
    {w.reset_desc && <span className="nx-resource-account__quota-reset">{w.reset_desc}</span>}
  </span>
);

export const ResourcePicker: React.FC<Props> = ({ agentId, preferProvider, onSelected }) => {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [decision, setDecision] = useState<SchedulerDecision | null>(null);
  const [loading, setLoading] = useState(true);
  const [selecting, setSelecting] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    void nexus.listResources().then(async (data) => {
      if (!mounted) return;
      setAccounts(data.accounts || []);
    }).catch(() => undefined).finally(() => mounted && setLoading(false));
    return () => { mounted = false; };
  }, [preferProvider]);

  const choose = async (account: ProviderAccount) => {
	if (!agentId) return;
	setSelecting(account.id);
    try {
	      const result = await nexus.selectResource(agentId, account.provider, account.profile);
      setDecision(result.decision);
	      await onSelected?.(account.provider, account.profile);
    } finally {
      setSelecting(null);
    }
  };

  if (loading) return <Spinner label={t('resources.loading')} />;
  if (accounts.length === 0) return <EmptyState icon={<Gauge size={20} />} title={t('resources.empty')} hint={t('resources.emptyHint')} />;

  return (
    <div className="nx-resource-picker">
      <div className="nx-resource-account-list">
        {accounts.map((account) => {
          const selected = decision?.selected?.id === account.id;
          const qv = account.quota_view;
          const hasGroups = qv && qv.model_groups && qv.model_groups.length > 0;
          const multiGroups = qv && qv.model_groups && qv.model_groups.length > 1;

          return (
            <button
              key={account.id}
              type="button"
              className="nx-resource-account"
              data-selected={selected}
              onClick={() => void choose(account)}
              disabled={!agentId || !account.authenticated || !account.available || selecting === account.id}
            >
              <span className="nx-resource-account__icon">
                {account.rate_limited ? <ShieldAlert size={17} /> : <Activity size={17} />}
              </span>
              <span className="nx-resource-account__main">
                <span className="nx-resource-account__title">
                  <strong>{account.display_name || account.provider}</strong>
                  <Badge>{account.profile}</Badge>
                  <Badge tone={healthTone(account.health)}>{translateStatus(account.health)}</Badge>
                  <Badge tone={account.available ? 'success' : 'danger'}>
                    {account.available ? 'DISPONIVEL' : account.avail_reasons?.rate_limited ? 'RATE LIMITED' : account.avail_reasons?.exhausted_windows ? 'QUOTA ESGOTADA' : 'INDISPONIVEL'}
                  </Badge>
                  {account.is_default && <Badge>{t('common.default')}</Badge>}
                  {selected && <Badge><Check size={12} /> {t('common.selected')}</Badge>}
                </span>
                <span className="nx-resource-account__quota">
                  {hasGroups ? (
                    qv!.model_groups.map((group, gi) => (
                      <span key={gi} className="nx-resource-account__group">
                        {multiGroups && group.name && (
                          <span className="nx-resource-account__group-name">{group.name}</span>
                        )}
                        {(group.windows || []).map((w) => (
                          <QuotaBar key={w.kind} w={w} />
                        ))}
                      </span>
                    ))
                  ) : (
                    <span className="nx-resource-account__quota-unknown">
                      {t('resources.unknown')}
                    </span>
                  )}
                </span>
              </span>
              {selecting === account.id && <Spinner />}
            </button>
          );
        })}
      </div>
      {decision && (
        <div className="nx-resource-decision">
          <span className="nx-resource-decision__label">{t('resources.decision')}</span>
          <span className="nx-resource-decision__reason">{decision.reason}</span>
        </div>
      )}
    </div>
  );
};
