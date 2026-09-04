import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { AlertTriangle, CheckCircle2, Terminal, XCircle } from 'lucide-react';
import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import { IconButton } from '../design-system';
import { notificationFromRuntime, type InAppNotification } from './inAppNotificationModel';
import { loadNotificationPrefs } from './notificationPrefs';

const DISMISS_AFTER_MS = 7_000;

export const InAppNotificationCenter: React.FC<{
  runtimes: RuntimeSession[];
  focusedProjectId?: string;
  onFocusRuntime: (runtimeId: string) => void;
}> = ({ runtimes, focusedProjectId, onFocusRuntime }) => {
  const [notifications, setNotifications] = useState<InAppNotification[]>([]);
  const observed = useRef(new Set<string>());
  const initialized = useRef(false);

  useEffect(() => {
    if (!loadNotificationPrefs().notificationsEnabled) return;
    const scoped = focusedProjectId
      ? runtimes.filter((runtime) => runtime.project_id === focusedProjectId)
      : [];
    const next = scoped
      .filter((runtime) => (runtime.provider_id || runtime.provider || '').toLowerCase() !== 'shell')
      .map(notificationFromRuntime)
      .filter((notification): notification is InAppNotification => notification !== null);

    if (!initialized.current) {
      next.forEach((notification) => observed.current.add(notification.id));
      initialized.current = true;
      return;
    }

    const fresh = next.filter((notification) => !observed.current.has(notification.id));
    if (fresh.length === 0) return;
    fresh.forEach((notification) => observed.current.add(notification.id));
    setNotifications((current) => [...fresh, ...current].slice(0, 3));
  }, [runtimes, focusedProjectId]);

  useEffect(() => {
    if (notifications.length === 0) return;
    const timer = window.setTimeout(() => setNotifications((current) => current.slice(0, -1)), DISMISS_AFTER_MS);
    return () => window.clearTimeout(timer);
  }, [notifications]);

  const dismiss = (id: string) => setNotifications((current) => current.filter((notification) => notification.id !== id));

  if (notifications.length === 0) return null;

  return createPortal(
    <section className="nx-notification-center" aria-label="Notificações recentes" aria-live="polite">
      {notifications.map((notification) => (
        <article className="nx-notification-toast" data-tone={notification.tone} key={notification.id} role="status">
          {notification.tone === 'success' ? (
            <CheckCircle2 size={18} />
          ) : notification.tone === 'warning' ? (
            <AlertTriangle size={18} />
          ) : (
            <XCircle size={18} />
          )}
          <div className="nx-notification-toast__body">
            <strong>{notification.title}</strong>
            <span>
              {sanitizeAttentionText(notification.projectName, 'Projeto')} ·{' '}
              {sanitizeAttentionText(notification.message, 'Sem detalhes adicionais.')}
            </span>
          </div>
          <IconButton label="Abrir terminal" onClick={() => onFocusRuntime(notification.runtimeId)}>
            <Terminal size={15} />
          </IconButton>
          <IconButton label="Fechar notificação" onClick={() => dismiss(notification.id)}>
            <XCircle size={15} />
          </IconButton>
        </article>
      ))}
    </section>,
    document.body,
  );
};
