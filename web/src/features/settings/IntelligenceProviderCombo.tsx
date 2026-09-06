import React, { useMemo, useState } from 'react';
import { Check, ChevronDown, Gauge, ShieldAlert } from 'lucide-react';
import { Badge, DropdownMenu } from '../../design-system';
import type { ProviderAccount } from '../../types';
import {
  accountKey,
  intelligenceModelChoices,
  isIntelligenceCLICapable,
  quotaLabel,
} from './intelligenceProfiles';

export const IntelligenceProviderCombo: React.FC<{
  accounts: ProviderAccount[];
  provider?: string;
  profile?: string;
  model?: string;
  onChange: (next: { provider: string; profile: string; model?: string }) => void;
}> = ({ accounts, provider, profile, model, onChange }) => {
  const [open, setOpen] = useState(false);
  const selectedKey = provider && profile ? `${provider}:${profile}` : '';
  const selected = accounts.find((account) => accountKey(account) === selectedKey);
  const models = useMemo(() => intelligenceModelChoices(selected), [selected]);

  const summary = selected
    ? `${selected.display_name || selected.provider} · ${selected.profile}`
    : 'Selecionar provedor e perfil…';
  const remaining = selected ? quotaLabel(selected) : null;

  return (
    <div className="nx-intel-combo">
      <span className="nx-intel-combo__label">Provedor e perfil</span>
      <DropdownMenu.Root open={open} onOpenChange={setOpen}>
        <DropdownMenu.Trigger asChild>
          <button type="button" className="nx-intel-combo__trigger" aria-expanded={open}>
            <span>
              <strong>{summary}</strong>
              {selected && (
                <small>
                  {remaining == null ? 'quota desconhecida' : `${remaining} restante`}
                  {selected.quota_view?.plan ? ` · ${selected.quota_view.plan}` : ''}
                </small>
              )}
            </span>
            <ChevronDown size={16} />
          </button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="nx-intel-combo__menu" align="start" sideOffset={4}>
            {accounts.length === 0 && (
              <p className="nx-intel-combo__empty">
                Nenhum provedor descoberto. Autentique um CLI em Usage.
              </p>
            )}
            {accounts.map((account) => {
              const key = accountKey(account);
              const active = key === selectedKey;
              const pct = quotaLabel(account);
              const capable = isIntelligenceCLICapable(account);
              return (
                <DropdownMenu.Item
                  key={account.id || key}
                  asChild
                  onSelect={() => {
                    onChange({ provider: account.provider, profile: account.profile, model: '' });
                  }}
                >
                  <button
                    type="button"
                    className="nx-intel-combo__option"
                    data-active={active ? 'true' : 'false'}
                  >
                    <span className="nx-intel-combo__option-title">
                      {active ? <Check size={14} /> : <Gauge size={14} />}
                      <strong>{account.display_name || account.provider}</strong>
                      <Badge>{account.profile}</Badge>
                      <Badge
                        tone={
                          account.health === 'healthy'
                            ? 'success'
                            : account.health === 'degraded'
                              ? 'warning'
                              : 'default'
                        }
                      >
                        {account.health || 'unknown'}
                      </Badge>
                      {account.rate_limited && (
                        <Badge tone="danger">
                          <ShieldAlert size={11} /> rate limit
                        </Badge>
                      )}
                      <Badge tone={capable ? 'success' : 'warning'}>
                        {capable ? 'headless' : 'sem headless'}
                      </Badge>
                    </span>
                    <small>
                      {pct == null ? 'Quota desconhecida' : `${pct} restante`}
                      {account.authenticated ? '' : ' · precisa autenticar'}
                      {account.quota_view?.account ? ` · ${account.quota_view.account}` : ''}
                    </small>
                  </button>
                </DropdownMenu.Item>
              );
            })}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      {models.length > 0 && (
        <div className="nx-intel-combo__models" role="group" aria-label="Modelos do perfil">
          <span>Modelos</span>
          <div>
            {models.map((choice) => {
              const active = (model || '') === choice.id;
              return (
                <button
                  key={choice.id}
                  type="button"
                  className="nx-intel-combo__model"
                  data-active={active ? 'true' : 'false'}
                  onClick={() =>
                    onChange({
                      provider: provider || selected?.provider || '',
                      profile: profile || selected?.profile || '',
                      model: active ? '' : choice.id,
                    })
                  }
                >
                  {choice.label}
                  {choice.remaining != null ? ` · ${Math.round(choice.remaining)}%` : ''}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
