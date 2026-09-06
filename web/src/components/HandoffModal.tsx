import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RuntimeSession, ProfileInfo } from '../types';
import { asArray } from '../lib/safeArray';
import { api } from '../api';
import { Button, Dialog, Select } from '../design-system';

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
  const { t } = useTranslation();
  const safeProfiles = asArray<ProfileInfo>(profiles);

  const runtimeProvider = runtime.provider_id || runtime.provider || '';
  const runtimeProfile = runtime.profile_id || runtime.profile || '';
  const availableProfiles = safeProfiles.filter(
    (p) => p.provider === runtimeProvider && p.name !== runtimeProfile,
  );
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : '',
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
    <Dialog open={open} onClose={onClose} title={t('legacy.handoffTitle', 'Account Handoff')}>
      <div className="space-y-4 font-mono">
        <div className="text-xs space-y-2" style={{ color: 'var(--nx-text-soft)' }}>
          <div
            className="p-3 rounded"
            style={{
              background: 'var(--nx-bg)',
              border: '1px solid var(--nx-border)',
            }}
          >
            <div>
              <span style={{ color: 'var(--nx-muted)' }}>
                {t('legacy.handoffSource', 'Source:')}
              </span>{' '}
              <span className="uppercase font-bold" style={{ color: 'var(--nx-text)' }}>
                {runtimeProvider}
              </span>{' '}
              ({runtimeProfile})
            </div>
            <div className="mt-1">
              <span style={{ color: 'var(--nx-muted)' }}>
                {t('legacy.handoffRuntime', 'Runtime:')}
              </span>{' '}
              {runtime.runtime_id}
            </div>
          </div>
          <p className="text-[11px]" style={{ color: 'var(--nx-muted)' }}>
            {t(
              'legacy.handoffDesc',
              'Switches active execution to another account while resuming the exact same conversation thread.',
            )}
          </p>
        </div>

        {availableProfiles.length === 0 ? (
          <div
            className="p-4 rounded text-xs"
            style={{
              background: 'var(--nx-surface-2)',
              border: '1px solid var(--nx-warning)',
              color: 'var(--nx-warning)',
            }}
          >
            {t('legacy.noOtherProfiles', {
              provider: runtimeProvider,
              defaultValue: `No alternative profiles found for provider ${runtimeProvider}. Add more accounts via nexus add ${runtimeProvider}.`,
            })}
          </div>
        ) : (
          <div className="space-y-2 text-xs">
            <label
              className="text-xs"
              style={{ color: 'var(--nx-text-soft)', display: 'block', marginBottom: 4 }}
            >
              {t('legacy.targetProfile', 'Target Profile:')}
            </label>
            <Select
              value={selectedProfile}
              onChange={(val) => setSelectedProfile(val)}
              options={availableProfiles.map((p) => ({
                value: p.name,
                label:
                  `${p.name} ${p.account_email ? `(${p.account_email})` : ''} ${p.plan ? `[${p.plan}]` : ''}`.trim(),
              }))}
            />
          </div>
        )}

        {error && (
          <div
            className="p-3 rounded text-xs"
            style={{
              background: 'var(--nx-surface-2)',
              border: '1px solid var(--nx-danger)',
              color: 'var(--nx-danger)',
            }}
          >
            {error}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('directSession.cancel', 'Cancel')}</Button>
          <Button
            tone="brand"
            disabled={loading || availableProfiles.length === 0}
            onClick={handleHandoff}
          >
            {loading
              ? t('legacy.executingHandoff', 'Performing Handoff...')
              : t('legacy.executeHandoff', 'Execute Handoff')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
