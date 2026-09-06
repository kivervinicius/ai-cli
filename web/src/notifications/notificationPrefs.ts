export type NotificationPrefs = {
  notificationsEnabled: boolean;
  soundEnabled: boolean;
};

const STORAGE_KEY = 'iapro:nexus:notification-prefs:v1';

export const DEFAULT_NOTIFICATION_PREFS: NotificationPrefs = {
  notificationsEnabled: true,
  soundEnabled: true,
};

export function loadNotificationPrefs(): NotificationPrefs {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...DEFAULT_NOTIFICATION_PREFS };
    const parsed = JSON.parse(raw) as Partial<NotificationPrefs>;
    return {
      notificationsEnabled: parsed.notificationsEnabled !== false,
      soundEnabled: parsed.soundEnabled !== false,
    };
  } catch {
    return { ...DEFAULT_NOTIFICATION_PREFS };
  }
}

export function saveNotificationPrefs(prefs: NotificationPrefs): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // ignore quota / private mode
  }
}

let audioCtx: AudioContext | null = null;

export function playAttentionSound(): void {
  try {
    const Ctx =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Ctx) return;
    if (!audioCtx) audioCtx = new Ctx();
    const ctx = audioCtx;
    if (ctx.state === 'suspended') {
      void ctx.resume();
    }
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.type = 'sine';
    osc.frequency.value = 880;
    gain.gain.value = 0.0001;
    osc.connect(gain);
    gain.connect(ctx.destination);
    const now = ctx.currentTime;
    gain.gain.exponentialRampToValueAtTime(0.08, now + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.22);
    osc.start(now);
    osc.stop(now + 0.25);
  } catch {
    // browsers may block until a user gesture
  }
}
