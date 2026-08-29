import React, { useEffect, useMemo, useState } from 'react';
import { Accessibility, ArrowUpCircle, MonitorCog, Palette, RefreshCw, RotateCcw, Sparkles } from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Input, Segmented, Switch } from '../../design-system';
import { useTheme, type ThemeAccent, type ThemeDensity, type ThemeScheme } from '../../design-system';
import { useWorkspace } from '../../workspace/WorkspaceProvider';
import { nexus } from '../../nexus/api';
import type { IntelligenceMode, IntelligenceStatus, ProviderAccount } from '../../types';
import { useTranslation } from 'react-i18next';
import { normalizeLanguage, supportedLanguages, type SupportedLanguage } from '../../i18n';

export const SettingsSurface: React.FC<{ onTour: () => void }> = ({ onTour }) => {
  const theme = useTheme();
  const { t, i18n } = useTranslation();
  const workspace = useWorkspace();
  const [updateInfo, setUpdateInfo] = useState<{ nexus_version: string; maestro_version: string; maestro_available: boolean } | null>(null);
  const [updating, setUpdating] = useState(false);
  const [updateSuccess, setUpdateSuccess] = useState(false);
  const [updateError, setUpdateError] = useState('');
  const [intelligence, setIntelligence] = useState<IntelligenceStatus | null>(null);
  const [intelligenceDraft, setIntelligenceDraft] = useState<IntelligenceStatus>({ mode: 'OFF', available: false });
  const [intelligenceResources, setIntelligenceResources] = useState<ProviderAccount[]>([]);
  const [intelligenceSaving, setIntelligenceSaving] = useState(false);
  const [intelligenceError, setIntelligenceError] = useState('');

  const checkUpdates = async () => {
    try {
      setUpdateError('');
      const data = await nexus.getSystemUpdates();
      setUpdateInfo(data);
    } catch (error) {
      setUpdateInfo(null);
      setUpdateError(error instanceof Error ? error.message : String(error));
    }
  };

  const loadIntelligence = async () => {
    try {
      setIntelligenceError('');
      const [status, resources] = await Promise.all([nexus.getIntelligence(), nexus.listResources()]);
      setIntelligence(status);
      setIntelligenceDraft(status);
      setIntelligenceResources(resources.accounts || []);
    } catch (error) {
      setIntelligenceError(error instanceof Error ? error.message : String(error));
    }
  };

  useEffect(() => {
    void checkUpdates();
    void loadIntelligence();
  }, []);

  const handleUpdate = async () => {
    setUpdating(true);
    setUpdateSuccess(false);
    try {
      const res = await nexus.performSystemUpdate();
      if (res.nexus_updated || res.maestro_updated) {
        setUpdateSuccess(true);
        void checkUpdates();
      }
    } catch (error) {
      setUpdateError(error instanceof Error ? error.message : String(error));
    } finally {
      setUpdating(false);
    }
  };

  const cliResources = useMemo(() => intelligenceResources.filter((account) =>
    account.available && account.authenticated &&
    account.capabilities?.headless === 'SUPPORTED' &&
    account.capabilities?.submit_prompt === 'SUPPORTED'
  ), [intelligenceResources]);

  const selectedCLI = `${intelligenceDraft.provider || ''}:${intelligenceDraft.profile || ''}`;

  const saveIntelligence = async () => {
    setIntelligenceSaving(true);
    setIntelligenceError('');
    try {
      const payload = {
        mode: intelligenceDraft.mode as IntelligenceMode,
        provider: intelligenceDraft.provider,
        profile: intelligenceDraft.profile,
        base_url: intelligenceDraft.base_url,
        model: intelligenceDraft.model,
        api_key_env: intelligenceDraft.api_key_env,
        api_key_file: intelligenceDraft.api_key_file,
      };
      const next = await nexus.updateIntelligence(payload);
      setIntelligence(next);
      setIntelligenceDraft(next);
    } catch (error) {
      setIntelligenceError(error instanceof Error ? error.message : String(error));
    } finally {
      setIntelligenceSaving(false);
    }
  };

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('settings.eyebrow')}</span>
          <h1>{t('settings.title')}</h1>
          <p>{t('settings.intro')}</p>
        </div>
      </div>
      <div className="nx-settings-grid">
        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <Palette size={17} />
            <div>
              <strong>{t('settings.theme')}</strong>
              <small>{t('settings.themeDescription')}</small>
            </div>
          </div>
          <Segmented
            ariaLabel={t('settings.colorScheme')}
            value={theme.scheme}
            onChange={(value) => theme.setScheme(value as ThemeScheme)}
            options={[
              { value: 'system', label: t('settings.system') },
              { value: 'dark', label: t('settings.dark') },
              { value: 'light', label: t('settings.light') },
              { value: 'high-contrast', label: t('settings.highContrast') },
            ]}
          />
          <Segmented
            ariaLabel={t('settings.accent')}
            value={theme.accent}
            onChange={(value) => theme.setAccent(value as ThemeAccent)}
            options={[
              { value: 'purple', label: t('settings.purple') },
              { value: 'blue', label: t('settings.blue') },
              { value: 'cyan', label: t('settings.cyan') },
              { value: 'neutral', label: t('settings.neutral') },
            ]}
          />
        </Card>

        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <Accessibility size={17} />
            <div>
              <strong>{t('settings.accessibility')}</strong>
              <small>{t('settings.accessibilityDescription')}</small>
            </div>
          </div>
          <Segmented
            ariaLabel={t('settings.density')}
            value={theme.density}
            onChange={(value) => theme.setDensity(value as ThemeDensity)}
            options={[
              { value: 'compact', label: t('settings.compact') },
              { value: 'comfortable', label: t('settings.comfortable') },
            ]}
          />
          <Switch
            checked={theme.reducedMotion}
            onChange={theme.setReducedMotion}
            label={t('settings.reduceMotion')}
            description={t('settings.reduceMotionDescription')}
          />
          <Button onClick={onTour}>{t('settings.replayTour')}</Button>
        </Card>

        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <Sparkles size={17} />
            <div><strong>{t('language.title')}</strong><small>{t('language.description')}</small></div>
          </div>
          <Segmented ariaLabel={t('language.title')} value={normalizeLanguage(i18n.language)} onChange={(value) => void i18n.changeLanguage(value as SupportedLanguage)} options={supportedLanguages.map((value) => ({ value, label: t(value === 'pt-BR' ? 'language.ptBR' : value === 'en' ? 'language.en' : 'language.es') }))} />
        </Card>

        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <Sparkles size={17} />
            <div>
              <strong>Nexus Intelligence</strong>
              <small>Optional semantic planning. Direct AI work remains available when Intelligence is OFF.</small>
            </div>
          </div>
          <Segmented
            ariaLabel="Intelligence mode"
            value={intelligenceDraft.mode}
            onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, mode: value as IntelligenceMode }))}
            options={[
              { value: 'OFF', label: 'Off' },
              { value: 'CLI', label: 'Coding CLI' },
              { value: 'OPENAI_COMPATIBLE', label: 'API' },
            ]}
          />
          {intelligenceDraft.mode === 'CLI' && (
            <label style={{ display: 'grid', gap: 6 }}>
              <span>Provider profile</span>
              <select
                value={selectedCLI}
                onChange={(event) => {
                  const account = cliResources.find((item) => `${item.provider}:${item.profile}` === event.target.value);
                  setIntelligenceDraft((prev) => ({ ...prev, provider: account?.provider || '', profile: account?.profile || '' }));
                }}
              >
                <option value="">Choose a verified headless profile…</option>
                {cliResources.map((account) => (
                  <option key={account.id} value={`${account.provider}:${account.profile}`}>{account.display_name || account.provider} · {account.profile}</option>
                ))}
              </select>
            </label>
          )}
          {intelligenceDraft.mode === 'OPENAI_COMPATIBLE' && (
            <>
              <Input value={intelligenceDraft.base_url || ''} onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, base_url: value }))} placeholder="Base URL (OpenAI-compatible)" />
              <Input value={intelligenceDraft.api_key_env || ''} onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, api_key_env: value }))} placeholder="Environment variable containing API key" />
              <Input value={intelligenceDraft.api_key_file || ''} onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, api_key_file: value }))} placeholder="Or secret file path" />
            </>
          )}
          {intelligenceDraft.mode !== 'OFF' && (
            <Input value={intelligenceDraft.model || ''} onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, model: value }))} placeholder="Model override (optional)" />
          )}
          {intelligence && (
            <InlineAlert tone={intelligence.available ? 'success' : intelligence.mode === 'OFF' ? 'info' : 'warning'} title={intelligence.available ? 'Intelligence provider ready' : intelligence.mode === 'OFF' ? 'Intelligence is optional' : 'Provider not ready'}>
              {intelligence.error || (intelligence.mode === 'OFF' ? 'Direct sessions and manual WorkPlans continue to work normally.' : `${intelligence.provider || 'Provider'} ${intelligence.profile || ''}`)}
            </InlineAlert>
          )}
          {intelligenceError && <InlineAlert tone="danger" title="Intelligence configuration error">{intelligenceError}</InlineAlert>}
          <Button tone="brand" onClick={() => void saveIntelligence()} disabled={intelligenceSaving}>
            {intelligenceSaving ? 'Saving…' : 'Save Intelligence'}
          </Button>
        </Card>

        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <ArrowUpCircle size={17} />
            <div>
              <strong>{t('settings.updates')}</strong>
              <small>{t('settings.updatesDescription')}</small>
            </div>
          </div>
          {updateInfo && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, fontSize: '12px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>IAPro Nexus:</span>
                <Badge tone="success">v{updateInfo.nexus_version}</Badge>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>Orquestrador Maestro:</span>
                <Badge tone={updateInfo.maestro_available ? 'brand' : 'warning'}>
                  v{updateInfo.maestro_version}
                </Badge>
              </div>
            </div>
          )}
          {updateError && <InlineAlert tone="danger" title="Update status unavailable">{updateError}</InlineAlert>}
          {updateSuccess && (
            <InlineAlert tone="success" title={t('settings.updated')}>
              {t('settings.upToDate')}
            </InlineAlert>
          )}
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <Button tone="brand" onClick={handleUpdate} disabled={updating}>
              <RefreshCw size={14} className={updating ? 'nx-spin' : ''} />
              {updating ? t('settings.updating') : t('settings.update')}
            </Button>
          </div>
        </Card>

        <Card className="nx-settings-card">
          <div className="nx-settings-card__title">
            <MonitorCog size={17} />
            <div>
              <strong>{t('settings.workspace')}</strong>
              <small>{t('settings.workspaceDescription')}</small>
            </div>
          </div>
          <Button tone="warning" onClick={workspace.reset}>
            <RotateCcw size={14} /> {t('settings.reset')}
          </Button>
        </Card>
      </div>
    </div>
  );
};
