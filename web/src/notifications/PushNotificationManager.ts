export interface PushNotificationPayload {
  runtimeId: string;
  projectName?: string;
  reason: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR' | string;
  context: string;
  dynamicTitle?: string;
  onClick?: () => void;
}

class PushNotificationService {
  private permission: NotificationPermission = typeof window !== 'undefined' && 'Notification' in window ? Notification.permission : 'default';
  private lastNotifiedKey = '';
  private lastNotifiedTime = 0;

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

  public sendPush(payload: PushNotificationPayload): void {
    if (!this.isSupported() || this.getPermission() !== 'granted') return;

    // Throttle duplicate notifications within 3 seconds
    const key = `${payload.runtimeId}:${payload.reason}:${payload.context}`;
    const now = Date.now();
    if (this.lastNotifiedKey === key && now - this.lastNotifiedTime < 3000) {
      return;
    }
    this.lastNotifiedKey = key;
    this.lastNotifiedTime = now;

    const proj = payload.projectName || 'Projeto';
    let title = `[Nexus | ${proj}] Notificação`;
    if (payload.reason === 'QUESTION') {
      title = `[Nexus | ${proj}] ❓ Pergunta Requer Atenção`;
    } else if (payload.reason === 'APPROVAL') {
      title = `[Nexus | ${proj}] ⚠️ Aprovação Necessária`;
    } else if (payload.reason === 'TASK_COMPLETED') {
      title = `[Nexus | ${proj}] ✅ Tarefa Concluída!`;
    }

    try {
      const notif = new Notification(title, {
        body: payload.context || (payload.reason === 'TASK_COMPLETED' ? 'Task completada com sucesso.' : 'O terminal aguarda sua interação.'),
        icon: '/logo.png',
        tag: `nexus-${payload.runtimeId}`,
        silent: false,
      });

      notif.onclick = () => {
        try {
          window.focus();
        } catch {}
        notif.close();
        if (payload.onClick) {
          payload.onClick();
        }
      };
    } catch (e) {
      console.warn('[PushNotificationService] Failed to show browser notification:', e);
    }
  }
}

export const pushNotifications = new PushNotificationService();
