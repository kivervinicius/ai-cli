import React from 'react';
import { useTranslation } from 'react-i18next';
import i18n from '../i18n';
import { EventRecord } from '../types';

interface EventsViewProps {
  events: EventRecord[];
}

export const EventsView: React.FC<EventsViewProps> = ({ events }) => {
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
          {t('legacy.auditEventLog', 'Runtime Audit Event Log')}
        </h2>
        <span className="text-xs font-mono" style={{ color: 'var(--nx-muted)' }}>
          {t('legacy.eventsRecorded', {
            count: events.length,
            defaultValue: `${events.length} events recorded`,
          })}
        </span>
      </div>

      {events.length === 0 ? (
        <div className="p-8 text-center text-xs font-mono" style={{ color: 'var(--nx-muted)' }}>
          {t('legacy.noAuditEvents', 'No audit events recorded yet.')}
        </div>
      ) : (
        <div className="font-mono text-xs max-h-[600px] overflow-y-auto">
          {events.map((ev) => (
            <div
              key={ev.id}
              className="p-3 transition hover:bg-[var(--nx-surface-2)]"
              style={{ borderBottom: '1px solid var(--nx-border)' }}
            >
              <div
                className="flex items-center justify-between"
                style={{ color: 'var(--nx-text-soft)' }}
              >
                <div className="flex items-center space-x-2">
                  <span
                    className="text-[10px] px-1.5 py-0.5 rounded font-bold"
                    style={{
                      background: 'var(--nx-surface-2)',
                      color: 'var(--nx-accent-text)',
                      border: '1px solid var(--nx-border)',
                    }}
                  >
                    {ev.type}
                  </span>
                  <span className="font-bold uppercase" style={{ color: 'var(--nx-text)' }}>
                    {ev.provider_id || ev.provider || 'system'}
                  </span>
                  <span style={{ color: 'var(--nx-muted)' }}>[{ev.runtime_id}]</span>
                </div>
                <span className="text-[10px]" style={{ color: 'var(--nx-muted)' }}>
                  {new Date(ev.timestamp).toLocaleTimeString(i18n.language)}
                </span>
              </div>
              <div className="mt-1" style={{ color: 'var(--nx-text-soft)' }}>
                {ev.summary}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
