import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  BrainCircuit,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Layers,
  ArrowUpCircle,
} from 'lucide-react';
import { Dialog, Button, Badge, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';

type MaestroSkill = { id: string; name?: string; description?: string };

type UpdateInfo = {
  nexus_version: string;
  maestro_version: string;
  maestro_latest_version?: string;
  maestro_available: boolean;
  update_available: boolean;
};

export const MaestroControlModal: React.FC<{
  open: boolean;
  onClose: () => void;
}> = ({ open, onClose }) => {
  const { t } = useTranslation();
  const [maestroStatus, setMaestroStatus] = useState<{
    available: boolean;
    capabilities?: {
      version: string;
      skills?: MaestroSkill[];
    };
  } | null>(null);
  const [updates, setUpdates] = useState<UpdateInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [updateError, setUpdateError] = useState('');
  const [updateResult, setUpdateResult] = useState<{
    maestro_updated: boolean;
    maestro_version: string;
    nexus_version: string;
    nexus_updated: boolean;
    error?: string;
  } | null>(null);

  const fetchStatus = async () => {
    setLoading(true);
    setUpdateError('');
    try {
      const [status, info] = await Promise.all([nexus.getMaestroStatus(), nexus.getSystemUpdates()]);
      setMaestroStatus(status);
      setUpdates(info);
    } catch {
      setMaestroStatus({ available: false });
      setUpdates(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      setUpdateResult(null);
      void fetchStatus();
    }
  }, [open]);

  const handleUpdate = async () => {
    setUpdating(true);
    setUpdateError('');
    setUpdateResult(null);
    try {
      const result = await nexus.performSystemUpdate();
      setUpdateResult(result);
      window.dispatchEvent(new CustomEvent('nexus:system-updates'));
      await fetchStatus();
    } catch (err) {
      setUpdateError(err instanceof Error ? err.message : String(err));
    } finally {
      setUpdating(false);
    }
  };

  const isAvailable = maestroStatus?.available;
  const version = maestroStatus?.capabilities?.version || updates?.maestro_version;
  const latest = updates?.maestro_latest_version;
  const skills = maestroStatus?.capabilities?.skills || [];
  const nexusVersion = updates?.nexus_version;

  return (
    <Dialog open={open} onClose={onClose} title={t('maestroControl.title')}>
      <div className="nx-maestro-modal">
        <p className="nx-maestro-subtitle">{t('maestroControl.subtitle')}</p>

        <div className="nx-maestro-status-card">
          <div className="nx-maestro-status-header">
            <div className="nx-maestro-status-icon">
              <BrainCircuit size={22} className={isAvailable ? 'nx-icon--brand' : 'nx-icon--warning'} />
            </div>
            <div>
              <strong>{t('maestroControl.status')}</strong>
              <div className="nx-maestro-badges">
                <Badge tone={isAvailable ? 'success' : 'warning'}>
                  {isAvailable ? <CheckCircle2 size={12} /> : <AlertTriangle size={12} />}
                  {isAvailable ? t('maestroControl.available') : t('maestroControl.degraded')}
                </Badge>
                {version && <Badge tone="brand">v{version}</Badge>}
                {updates?.update_available && latest && (
                  <Badge tone="warning">{t('maestroControl.latest')}: v{latest}</Badge>
                )}
              </div>
            </div>
            <Button size="sm" tone="ghost" onClick={() => void fetchStatus()} disabled={loading || updating}>
              <RefreshCw size={12} className={loading ? 'nx-spin' : ''} />
            </Button>
          </div>
          {nexusVersion && (
            <p className="nx-maestro-nexus-version">
              {t('maestroControl.nexusVersion')}: v{nexusVersion}
            </p>
          )}
        </div>

        {updateResult && (
          <InlineAlert tone={updateResult.maestro_updated ? 'success' : 'info'} title={t('maestroControl.updateResult')}>
            {updateResult.maestro_updated
              ? t('maestroControl.updateMaestroDone', { version: updateResult.maestro_version })
              : t('maestroControl.updateMaestroSame', { version: updateResult.maestro_version })}
            {' '}
            {t('maestroControl.nexusBinaryNote', { version: updateResult.nexus_version })}
          </InlineAlert>
        )}
        {updateError && (
          <InlineAlert tone="danger" title={t('maestroControl.updateMaestroFailed')}>
            {updateError}
          </InlineAlert>
        )}

        <div className="nx-maestro-skills-section">
          <div className="nx-section-header">
            <Layers size={14} />
            <h4>{t('maestroControl.skillsTitle')} ({skills.length})</h4>
          </div>
          {skills.length === 0 ? (
            <p className="nx-maestro-subtitle">{t('maestroControl.noSkills')}</p>
          ) : (
            <div className="nx-skills-grid">
              {skills.map((skill) => (
                <div key={skill.id} className="nx-skill-pill" title={skill.description}>
                  <span className="nx-skill-dot" />
                  <span className="nx-skill-name">{skill.name || skill.id}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="nx-maestro-footer">
          <Button tone="brand" onClick={() => void handleUpdate()} disabled={updating}>
            <ArrowUpCircle size={14} className={updating ? 'nx-spin' : ''} />
            <span>{updating ? t('maestroControl.updating') : t('maestroControl.updateNow')}</span>
          </Button>
          <Button onClick={onClose}>{t('common.closeDialog')}</Button>
        </div>
      </div>
    </Dialog>
  );
};
