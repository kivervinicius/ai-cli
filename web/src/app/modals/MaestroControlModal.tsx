import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  BrainCircuit,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Sparkles,
  ExternalLink,
  ShieldAlert,
  Layers,
} from 'lucide-react';
import { Dialog, Button, Badge } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Project } from '../../types';

export const MaestroControlModal: React.FC<{
  open: boolean;
  onClose: () => void;
  project: Project;
  onProjectUpdated: (updated: Project) => void;
  onOpenMaestroSurface: () => void;
}> = ({ open, onClose, project, onProjectUpdated, onOpenMaestroSurface }) => {
  const { t } = useTranslation();
  const [maestroStatus, setMaestroStatus] = useState<{
    available: boolean;
    capabilities?: {
      version: string;
      skills?: Array<{ id: string; name?: string; description?: string }>;
    };
    advice_error?: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState(project.maestro_mode || 'ASSIST');
  const [savingMode, setSavingMode] = useState(false);

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const res = await nexus.getMaestroStatus();
      setMaestroStatus(res);
    } catch {
      setMaestroStatus({ available: false });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      setMode(project.maestro_mode || 'ASSIST');
      fetchStatus();
    }
  }, [open, project.id]);

  const handleModeChange = async (newMode: 'ASSIST' | 'ORCHESTRATE' | 'OFF') => {
    setMode(newMode);
    setSavingMode(true);
    try {
      const updated = await nexus.updateProject(project.id, { maestro_mode: newMode });
      onProjectUpdated(updated);
    } catch (err) {
      console.error('Failed to update maestro mode', err);
    } finally {
      setSavingMode(false);
    }
  };

  const isAvailable = maestroStatus?.available;
  const version = maestroStatus?.capabilities?.version || '0.1.25';
  const skills = maestroStatus?.capabilities?.skills || [
    { id: 'skill-saas-factory', name: 'SaaS Factory', description: 'Architecture & multi-tenant SaaS scaffolding' },
    { id: 'skill-tdd', name: 'TDD Engine', description: 'Test-Driven Development & red-green-refactor loop' },
    { id: 'skill-security-hooks', name: 'Security Hooks', description: 'Secret scanning & injection prevention' },
    { id: 'skill-saas-security-scan', name: 'Security Scan', description: 'Automated vulnerability auditor' },
    { id: 'skill-saas-dast-recon', name: 'DAST Recon', description: 'Dynamic application security testing' },
    { id: 'skill-dev-hierarchy', name: 'Dev Hierarchy', description: 'Engineering team delegation & orchestrations' },
  ];

  return (
    <Dialog open={open} onClose={onClose} title={t('maestroControl.title')}>
      <div className="nx-maestro-modal">
        <p className="nx-maestro-subtitle">{t('maestroControl.subtitle')}</p>

        {/* Integration Status Card */}
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
                <Badge tone="brand">v{version}</Badge>
              </div>
            </div>
            <Button size="sm" tone="ghost" onClick={fetchStatus} disabled={loading}>
              <RefreshCw size={12} className={loading ? 'nx-spin' : ''} />
            </Button>
          </div>
        </div>

        {/* Mode Selector */}
        <div className="nx-maestro-mode-section">
          <h4>{t('maestroControl.modeTitle')}</h4>
          <div className="nx-mode-options">
            <button
              type="button"
              className={`nx-mode-card ${mode === 'ASSIST' ? 'nx-mode-card--active' : ''}`}
              onClick={() => handleModeChange('ASSIST')}
              disabled={savingMode}
            >
              <div className="nx-mode-card-header">
                <strong>{t('maestroControl.modeAssist')}</strong>
                {mode === 'ASSIST' && <CheckCircle2 size={14} className="nx-mode-check" />}
              </div>
              <p>{t('maestroControl.modeAssistDesc')}</p>
            </button>

            <button
              type="button"
              className={`nx-mode-card ${mode === 'ORCHESTRATE' ? 'nx-mode-card--active' : ''}`}
              onClick={() => handleModeChange('ORCHESTRATE')}
              disabled={savingMode}
            >
              <div className="nx-mode-card-header">
                <strong>{t('maestroControl.modeAutonomous')}</strong>
                {mode === 'ORCHESTRATE' && <CheckCircle2 size={14} className="nx-mode-check" />}
              </div>
              <p>{t('maestroControl.modeAutonomousDesc')}</p>
            </button>

            <button
              type="button"
              className={`nx-mode-card ${mode === 'OFF' ? 'nx-mode-card--active' : ''}`}
              onClick={() => handleModeChange('OFF')}
              disabled={savingMode}
            >
              <div className="nx-mode-card-header">
                <strong>{t('maestroControl.modeOff')}</strong>
                {mode === 'OFF' && <CheckCircle2 size={14} className="nx-mode-check" />}
              </div>
              <p>{t('maestroControl.modeOffDesc')}</p>
            </button>
          </div>
        </div>

        {/* Integrated Skills List */}
        <div className="nx-maestro-skills-section">
          <div className="nx-section-header">
            <Layers size={14} />
            <h4>{t('maestroControl.skillsTitle')} ({skills.length})</h4>
          </div>
          <div className="nx-skills-grid">
            {skills.map((skill) => (
              <div key={skill.id} className="nx-skill-pill" title={skill.description}>
                <span className="nx-skill-dot" />
                <span className="nx-skill-name">{skill.name || skill.id}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Footer Actions */}
        <div className="nx-maestro-footer">
          <Button
            tone="brand"
            onClick={() => {
              onClose();
              onOpenMaestroSurface();
            }}
          >
            <ExternalLink size={14} />
            <span>{t('maestroControl.openWorkspace')}</span>
          </Button>
          <Button onClick={onClose}>{t('common.closeDialog')}</Button>
        </div>
      </div>
    </Dialog>
  );
};
