import type { ContextReadinessState } from '../../types';

export type ComposerGateAction = 'NONE' | 'PREPARE' | 'WAIT' | 'REFRESH' | 'RETRY';
export interface ComposerGate {
  canCompose: boolean;
  canFinalize: boolean;
  canMaterialize: boolean;
  canExecute: boolean;
  action: ComposerGateAction;
  /** i18n key under work.gate.* */
  reasonKey: string;
  /** @deprecated prefer reasonKey + t() */
  reason: string;
}

const REASONS: Record<string, string> = {
  ready: 'Durable project context matches the current source fingerprint.',
  missing: 'No durable context readiness checkpoint exists yet.',
  hydrating: 'Context readiness is being evaluated.',
  stale: 'Branch, HEAD, dirty state or Maestro version changed.',
  failed: 'Context readiness could not be established.',
};

export function composerNeedsGapConfirmation(brief: {
  open_questions?: string[];
  readiness?: { state?: string };
}): boolean {
  return brief.readiness?.state === 'BLOCKED' || (brief.open_questions?.length ?? 0) > 0;
}

/** Prefer a readable session label over keyboard-spam or raw IDs. */
export function formatComposerSessionTitle(
  title: string | undefined,
  id: string,
  createdAt?: string,
): string {
  const raw = String(title || '').trim();
  const repeatedKeySpam = raw.length <= 24 && /^(.)\1{4,}$/u.test(raw);
  const spam = raw.length < 2 || repeatedKeySpam || /^[çc]l?k+$/iu.test(raw);
  if (!raw || spam) {
    if (createdAt) {
      const ts = Date.parse(createdAt);
      if (Number.isFinite(ts)) {
        return new Intl.DateTimeFormat(undefined, {
          day: '2-digit',
          month: 'short',
          hour: '2-digit',
          minute: '2-digit',
        }).format(ts);
      }
    }
    return id.slice(-8).toUpperCase();
  }
  return raw.length > 52 ? `${raw.slice(0, 49)}…` : raw;
}

export function composerSessionStateKey(state: string | undefined): string {
  const normalized = String(state || 'UNKNOWN').toUpperCase();
  return `work.composer.sessionState.${normalized}`;
}

export function composerGateForReadiness(state: ContextReadinessState): ComposerGate {
  switch (state) {
    case 'READY':
      return {
        canCompose: true,
        canFinalize: true,
        canMaterialize: true,
        canExecute: true,
        action: 'NONE',
        reasonKey: 'work.gate.ready',
        reason: REASONS.ready,
      };
    case 'MISSING':
      return {
        canCompose: true,
        canFinalize: true,
        canMaterialize: false,
        canExecute: false,
        action: 'PREPARE',
        reasonKey: 'work.gate.missing',
        reason: REASONS.missing,
      };
    case 'HYDRATING':
      return {
        canCompose: true,
        canFinalize: true,
        canMaterialize: false,
        canExecute: false,
        action: 'WAIT',
        reasonKey: 'work.gate.hydrating',
        reason: REASONS.hydrating,
      };
    case 'STALE':
      return {
        canCompose: true,
        canFinalize: true,
        canMaterialize: false,
        canExecute: false,
        action: 'REFRESH',
        reasonKey: 'work.gate.stale',
        reason: REASONS.stale,
      };
    case 'FAILED':
    default:
      return {
        canCompose: true,
        canFinalize: true,
        canMaterialize: false,
        canExecute: false,
        action: 'RETRY',
        reasonKey: 'work.gate.failed',
        reason: REASONS.failed,
      };
  }
}
