import React, { useEffect, useState } from 'react';
import { Accessibility, ArrowUpCircle, MonitorCog, Palette, RefreshCw, RotateCcw, Sparkles } from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Segmented, Switch } from '../../design-system';
import { useTheme, type ThemeAccent, type ThemeDensity, type ThemeScheme } from '../../design-system';
import { useWorkspace } from '../../workspace/WorkspaceProvider';
import { nexus } from '../../nexus/api';
import { useTranslation } from 'react-i18next';
import { normalizeLanguage, supportedLanguages, type SupportedLanguage } from '../../i18n';

export const SettingsSurface: React.FC<{ onTour: () => void }> = ({ onTour }) => {
  const theme = useTheme();
  const { t, i18n } = useTranslation();
  const workspace = useWorkspace();
  const [updateInfo, setUpdateInfo] = useState<{ nexus_version: string; maestro_version: string; maestro_available: boolean } | null>(null);
  const [updating, setUpdating] = useState(false);
  const [updateSuccess, setUpdateSuccess] = useState(false);

  const checkUpdates = async () => {
    try {
      const data = await nexus.getSystemUpdates();
      setUpdateInfo(data);
    } catch {
      setUpdateInfo({ nexus_version: '0.4.1', maestro_version: '0.1.22', maestro_available: true });
    }
  };

  useEffect(() => {
    void checkUpdates();
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
    } catch {
      // Fallback
    } finally {
      setUpdating(false);
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
