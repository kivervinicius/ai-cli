import React from 'react';
import { useTranslation } from 'react-i18next';
import { ProviderInfo } from '../types';
import { CheckCircle2, XCircle } from 'lucide-react';

interface ProvidersViewProps {
  providers: ProviderInfo[];
}

export const ProvidersView: React.FC<ProvidersViewProps> = ({ providers }) => {
  const { t } = useTranslation();

  return (
    <div
      className="rounded-lg overflow-hidden"
      style={{
        background: 'var(--nx-surface)',
        border: '1px solid var(--nx-border)',
      }}
    >
      <div
        className="px-4 py-3 flex items-center justify-between"
        style={{ borderBottom: '1px solid var(--nx-border)' }}
      >
        <h2
          className="text-xs font-bold font-mono uppercase tracking-wider"
          style={{ color: 'var(--nx-text)' }}
        >
          {t('legacy.truthfulProviders', 'Truthful Provider Capabilities')}
        </h2>
        <span className="text-xs font-mono" style={{ color: 'var(--nx-muted)' }}>
          {t('legacy.liveEvidence', 'Live Evidence Engine')}
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead
            style={{
              background: 'var(--nx-bg-elevated)',
              color: 'var(--nx-text-soft)',
              borderBottom: '1px solid var(--nx-border)',
            }}
          >
            <tr>
              <th className="px-4 py-3">{t('legacy.provider', 'Provider')}</th>
              <th className="px-4 py-3">{t('legacy.installed', 'Installed')}</th>
              <th className="px-4 py-3">{t('legacy.version', 'Version')}</th>
              <th className="px-4 py-3">{t('legacy.controlLevel', 'Control Level')}</th>
              <th className="px-4 py-3">{t('legacy.terminal', 'Terminal')}</th>
              <th className="px-4 py-3">{t('legacy.events', 'Events')}</th>
              <th className="px-4 py-3">{t('legacy.resume', 'Resume')}</th>
            </tr>
          </thead>
          <tbody>
            {providers.map((p) => {
              return (
                <tr
                  key={p.id}
                  className="transition hover:bg-[var(--nx-surface-2)]"
                  style={{ borderBottom: '1px solid var(--nx-border)' }}
                >
                  <td className="px-4 py-3 font-bold uppercase" style={{ color: 'var(--nx-text)' }}>
                    {p.id}
                  </td>
                  <td className="px-4 py-3">
                    {p.installed ? (
                      <span
                        className="inline-flex items-center font-semibold"
                        style={{ color: 'var(--nx-success)' }}
                      >
                        <CheckCircle2 className="w-4 h-4 mr-1" /> {t('legacy.yes', 'Yes')}
                      </span>
                    ) : (
                      <span
                        className="inline-flex items-center"
                        style={{ color: 'var(--nx-muted)' }}
                      >
                        <XCircle className="w-4 h-4 mr-1" /> {t('legacy.no', 'No')}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3" style={{ color: 'var(--nx-text-soft)' }}>
                    {p.version || '—'}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className="px-2 py-0.5 rounded font-bold text-[11px]"
                      style={{
                        background: 'var(--nx-surface-2)',
                        color: 'var(--nx-accent-text)',
                        border: '1px solid var(--nx-border)',
                      }}
                    >
                      {p.control_level}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className="font-medium"
                      style={{
                        color:
                          (p.capabilities?.terminal?.status || 'SUPPORTED') === 'SUPPORTED'
                            ? 'var(--nx-success)'
                            : 'var(--nx-muted)',
                      }}
                    >
                      {t(
                        `status.${p.capabilities?.terminal?.status || 'SUPPORTED'}`,
                        p.capabilities?.terminal?.status || 'SUPPORTED',
                      )}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span style={{ color: 'var(--nx-muted)' }}>
                      {t(
                        `status.${p.capabilities?.structured_events?.status || 'UNSUPPORTED'}`,
                        p.capabilities?.structured_events?.status || 'UNSUPPORTED',
                      )}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className="font-medium"
                      style={{
                        color:
                          (p.capabilities?.resume?.status || 'SUPPORTED') === 'SUPPORTED'
                            ? 'var(--nx-success)'
                            : 'var(--nx-muted)',
                      }}
                    >
                      {t(
                        `status.${p.capabilities?.resume?.status || 'SUPPORTED'}`,
                        p.capabilities?.resume?.status || 'SUPPORTED',
                      )}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
