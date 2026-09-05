import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ProviderInfo, ProfileInfo, RuntimeSession, Workspace } from '../types';
import { asArray } from '../lib/safeArray';
import { api } from '../api';
import { Play, FolderGit2 } from 'lucide-react';
import { Button, Dialog } from '../design-system';

interface StartModalProps {
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  workspace: string;
  workspaces: Workspace[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
  open?: boolean;
}

export const StartModal: React.FC<StartModalProps> = ({
  providers,
  profiles,
  workspace,
  workspaces,
  onClose,
  onSuccess,
  open = true,
}) => {
  const { t } = useTranslation();
  const safeProviders = asArray<ProviderInfo>(providers);
  const safeProfiles = asArray<ProfileInfo>(profiles);
  const safeWorkspaces = asArray<Workspace>(workspaces);

  const installedProviders = safeProviders.filter((p) => p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    installedProviders.length > 0 ? installedProviders[0].id : 'agy'
  );

  const availableProfiles = safeProfiles.filter((p) => p.provider === selectedProvider);
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : 'default'
  );

  const [selectedWorkspace, setSelectedWorkspace] = useState<string>(workspace || (safeWorkspaces.length > 0 ? safeWorkspaces[0].path : ''));
  const [isCustomWorkspace, setIsCustomWorkspace] = useState(false);
  const [customWorkspace, setCustomWorkspace] = useState('');
  const [sessionTitle, setSessionTitle] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleProviderChange = (prov: string) => {
    setSelectedProvider(prov);
    const profs = safeProfiles.filter((p) => p.provider === prov);
    setSelectedProfile(profs.length > 0 ? profs[0].name : 'default');
  };

  const handleWorkspaceSelect = (val: string) => {
    if (val === '__custom__') {
      setIsCustomWorkspace(true);
    } else {
      setIsCustomWorkspace(false);
      setSelectedWorkspace(val);
    }
  };

  const handleStart = async () => {
    setLoading(true);
    setError('');
    const targetWs = isCustomWorkspace ? customWorkspace.trim() : selectedWorkspace;
    if (!targetWs) {
      setError('Workspace path cannot be empty');
      setLoading(false);
      return;
    }

    try {
      const res = await api.startRuntime(
        selectedProvider,
        selectedProfile,
        targetWs,
        [],
        sessionTitle.trim() || undefined
      );
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to start runtime');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title={t('legacy.startRuntimeBtn', 'Start Agent Runtime')}>
      <div className="space-y-4 font-mono">
        <div className="space-y-3 text-xs">
          {/* Target Project / Workspace */}
          <div>
            <label className="flex items-center space-x-1.5" style={{ color: 'var(--nx-text-soft)' }}>
              <FolderGit2 className="w-3.5 h-3.5" style={{ color: 'var(--nx-accent-text)' }} />
              <span>{t('legacy.targetWorkspace', 'Target Project / Workspace:')}</span>
            </label>
            <select
              value={isCustomWorkspace ? '__custom__' : selectedWorkspace}
              onChange={(e) => handleWorkspaceSelect(e.target.value)}
              className="mt-1 w-full rounded px-3 py-2 focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
            >
              {safeWorkspaces.map((ws) => (
                <option key={ws.path} value={ws.path}>
                  {ws.name} ({ws.path})
                </option>
              ))}
              <option value="__custom__">{t('legacy.enterCustomPath', '+ Enter Custom Path...')}</option>
            </select>
            {isCustomWorkspace && (
              <input
                type="text"
                placeholder={t('legacy.customPathPlaceholder', '/absolute/path/to/project')}
                value={customWorkspace}
                onChange={(e) => setCustomWorkspace(e.target.value)}
                className="mt-2 w-full rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
                style={{
                  background: 'var(--nx-bg)',
                  border: '1px solid var(--nx-border)',
                  color: 'var(--nx-text)',
                }}
                autoFocus
              />
            )}
          </div>

          {/* Session Title / Goal */}
          <div>
            <label style={{ color: 'var(--nx-text-soft)' }}>
              {t('legacy.sessionObjective', 'Session Title / Objective (Optional):')}
            </label>
            <input
              type="text"
              placeholder={t('legacy.sessionPlaceholder', 'e.g. Refactor Auth, Debug Database...')}
              value={sessionTitle}
              onChange={(e) => setSessionTitle(e.target.value)}
              className="mt-1 w-full rounded px-3 py-2 focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
            />
          </div>

          {/* Coding Provider */}
          <div>
            <label style={{ color: 'var(--nx-text-soft)' }}>
              {t('legacy.codingProvider', 'Coding Provider:')}
            </label>
            <select
              value={selectedProvider}
              onChange={(e) => handleProviderChange(e.target.value)}
              className="mt-1 w-full rounded px-3 py-2 uppercase focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
            >
              {installedProviders.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.id} {p.version ? `(${p.version})` : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Profile / Account */}
          <div>
            <label style={{ color: 'var(--nx-text-soft)' }}>
              {t('legacy.profileAccount', 'Profile / Account:')}
            </label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="mt-1 w-full rounded px-3 py-2 focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
            >
              {availableProfiles.length > 0 ? (
                availableProfiles.map((p) => (
                  <option key={p.name} value={p.name}>
                    {p.name} {p.account_email ? `(${p.account_email})` : ''}
                  </option>
                ))
              ) : (
                <option value="default">default</option>
              )}
            </select>
          </div>
        </div>

        {error && (
          <div
            className="p-3 rounded text-xs"
            style={{
              background: 'var(--nx-surface-2)',
              border: '1px solid var(--nx-danger)',
              color: 'var(--nx-danger)',
            }}
          >
            {error}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('directSession.cancel', 'Cancel')}</Button>
          <Button
            tone="brand"
            disabled={loading || installedProviders.length === 0}
            onClick={handleStart}
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>{loading ? t('legacy.startingRuntime', 'Starting Runtime...') : t('legacy.startRuntimeBtn', 'Start Agent Runtime')}</span>
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
