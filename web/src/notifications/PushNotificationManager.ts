export interface PushNotificationPayload {
  runtimeId: string;
  projectName?: string;
  reason: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR' | string;
  context: string;
  dynamicTitle?: string;
  fingerprint?: string;
  promptKind?: string;
  onClick?: () => void;
}

class PushNotificationService {
  private permission: NotificationPermission =
    typeof window !== 'undefined' && 'Notification' in window ? Notification.permission : 'default';
  private notifiedFingerprints = new Set<string>();

  public isSupported(): boolean {
    return typeof window !== 'undefined' && 'Notification' in window;
  }

  public getPermission(): NotificationPermission {
    if (this.isSupported()) {
      this.permission = Notification.permission;
    }
    return this.permission;
  }

  public async requestPermission(): Promise<NotificationPermission> {
    if (!this.isSupported()) return 'denied';
    try {
      const perm = await Notification.requestPermission();
      this.permission = perm;
      return perm;
    } catch {
      return this.permission;
    }
  }

  public confirmEnabled(projectName?: string): void {
    if (!this.isSupported() || this.getPermission() !== 'granted') return;
    try {
      new Notification(`Nexus · ${projectName || 'Projeto'}`, {
        body: 'Notificações do navegador ativadas. Com a aba fechada, o aviso honesto sai do processo nexus.',
        icon: '/logo.png',
        tag: 'nexus-permission-ok',
      });
    } catch (e) {
      console.warn('[PushNotificationService] Failed to show browser notification:', e);
    }
  }

  public sendPush(payload: PushNotificationPayload): void {
    if (!this.isSupported() || this.getPermission() !== 'granted') return;

    // Fail-closed: never push TASK_COMPLETED / WORKING / empty context as "attention".
    if (payload.reason !== 'QUESTION' && payload.reason !== 'APPROVAL' && payload.reason !== 'ERROR') {
      return;
    }
    const context = (payload.context || '').trim();
    if (!context) return;
    if (payload.promptKind === 'none') return;
    // Never treat the generic fallback as a real question.
    if (/o agente requer atenção/i.test(context) || /atenção necessária no terminal/i.test(context)) {
      return;
    }

    const fingerprint =
      payload.fingerprint ||
      `${payload.runtimeId}|${payload.promptKind || ''}|${context.toLowerCase().replace(/\s+/g, ' ')}`;
    if (this.notifiedFingerprints.has(fingerprint)) {
      return;
    }
    this.notifiedFingerprints.add(fingerprint);

    const proj = payload.projectName || 'Projeto';
    const title = payload.reason === 'ERROR' ? `Nexus · ${proj} · erro` : `Nexus · ${proj}`;

    try {
      const notif = new Notification(title, {
        body: context,
        icon: '/logo.png',
        tag: `nexus-${fingerprint}`,
        silent: false,
      });

      notif.onclick = () => {
        try {
          window.focus();
        } catch {
          // Browsers may block focus changes triggered by notifications.
        }
        notif.close();
        if (payload.onClick) {
          payload.onClick();
        }
      };
    } catch (e) {
      console.warn('[PushNotificationService] Failed to show browser notification:', e);
    }
  }

  /** Test helper / recovery when the wait clears. */
  public clearFingerprint(fingerprint: string): void {
    this.notifiedFingerprints.delete(fingerprint);
  }
}

export const pushNotifications = new PushNotificationService();
