import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';
import { en, es, ptBR } from './resources';

export const supportedLanguages = ['pt-BR', 'en', 'es'] as const;
export type SupportedLanguage = typeof supportedLanguages[number];
export const languageStorageKey = 'iapro:nexus:language:v1';

export function normalizeLanguage(value?: string | null): SupportedLanguage {
  const tag = (value || '').trim().replace('_', '-').toLowerCase();
  if (tag === 'pt' || tag.startsWith('pt-')) return 'pt-BR';
  if (tag === 'es' || tag.startsWith('es-')) return 'es';
  if (tag === 'en' || tag.startsWith('en-')) return 'en';
  return 'en';
}

const initialization = i18n.use(LanguageDetector).use(initReactI18next).init({
  resources: { en: { translation: en }, 'pt-BR': { translation: ptBR }, es: { translation: es } },
  supportedLngs: [...supportedLanguages],
  fallbackLng: 'en',
  load: 'currentOnly',
  interpolation: { escapeValue: false },
  detection: { order: ['localStorage', 'navigator'], lookupLocalStorage: languageStorageKey, caches: ['localStorage'], convertDetectedLanguage: normalizeLanguage },
  react: { useSuspense: false },
});

const syncDocumentLanguage = (language: string) => {
  const normalized = normalizeLanguage(language);
  if (typeof document !== 'undefined') {
    document.documentElement.lang = normalized;
    document.documentElement.dir = 'ltr';
  }
  if (language !== normalized) void i18n.changeLanguage(normalized);
};

i18n.on('languageChanged', syncDocumentLanguage);
void initialization.then(() => syncDocumentLanguage(i18n.language));

export function translateStatus(status?: string): string {
  if (!status) return i18n.t('common.unknown');
  const key = `status.${status.toUpperCase()}`;
  return i18n.exists(key) ? i18n.t(key) : status;
}

export function formatResourcePolicy(policy?: string | null): string {
  if (!policy || policy === '{}' || policy === 'null' || policy.trim() === '') {
    return i18n.t('policy.balanced');
  }
  try {
    const parsed = JSON.parse(policy);
    if (typeof parsed === 'object' && parsed !== null) {
      const mode = parsed.mode || parsed.policy || parsed.name;
      if (mode) return formatResourcePolicy(mode);
      return i18n.t('policy.balanced');
    }
  } catch {}
  const upper = policy.toUpperCase();
  if (upper === 'BALANCED') return i18n.t('policy.balanced');
  if (upper === 'PERFORMANCE') return i18n.t('policy.performance');
  if (upper === 'COST') return i18n.t('policy.cost');
  return policy;
}

export function translateContinuityStatus(status?: string): string {
  if (!status) return i18n.t('continuity.PENDING');
  const key = `continuity.${status}`;
  return i18n.exists(key) ? i18n.t(key) : status.replace(/_/g, ' ');
}

export function translateIsolation(isolation?: string): string {
  if (!isolation || isolation === 'project') return i18n.t('isolation.project');
  if (isolation === 'worktree') return i18n.t('isolation.worktree');
  if (isolation === 'none') return i18n.t('isolation.none');
  if (isolation === 'developer') return i18n.t('isolation.developer');
  return isolation;
}

export default i18n;
