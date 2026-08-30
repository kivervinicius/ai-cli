import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ProviderInfo, ProfileInfo, RuntimeSession, Workspace } from '../types';
import { api } from '../api';
import { X, Play, FolderGit2 } from 'lucide-react';

interface StartModalProps {
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  workspace: string;
  workspaces: Workspace[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
}

export const StartModal: React.FC<StartModalProps> = ({
  providers,
  profiles,
  workspace,
  workspaces,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const installedProviders = (providers || []).filter((p) => p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    installedProviders.length > 0 ? installedProviders[0].id : 'agy'
  );

  const availableProfiles = (profiles || []).filter((p) => p.provider === selectedProvider);
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : 'default'
  );

  const [selectedWorkspace, setSelectedWorkspace] = useState<string>(workspace || ((workspaces && workspaces.length > 0) ? workspaces[0].path : ''));
  const [isCustomWorkspace, setIsCustomWorkspace] = useState(false);
  const [customWorkspace, setCustomWorkspace] = useState('');
  const [sessionTitle, setSessionTitle] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleProviderChange = (prov: string) => {
    setSelectedProvider(prov);
    const profs = (profiles || []).filter((p) => p.provider === prov);
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
      setError(t('startModal.emptyWorkspaceError'));
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
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
      <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4 font-mono">
        <div className="flex items-center justify-between border-b border-slate-800 pb-3">
          <div className="flex items-center space-x-2">
            <Play className="w-4 h-4 text-sky-400 fill-current" />
            <h3 className="text-sm font-bold text-slate-100 uppercase tracking-wider">
              {t('startModal.title')}
            </h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white" aria-label={t('common.closeDialog')}>
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="space-y-3 text-xs">
          {/* Target Project / Workspace */}
          <div>
            <label className="text-slate-400 flex items-center space-x-1.5">
              <FolderGit2 className="w-3.5 h-3.5 text-sky-400" />
              <span>{t('startModal.targetWorkspace')}</span>
            </label>
            <select
              value={isCustomWorkspace ? '__custom__' : selectedWorkspace}
              onChange={(e) => handleWorkspaceSelect(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 focus:outline-none focus:border-sky-500"
            >
              {(workspaces || []).map((ws) => (
                <option key={ws.path} value={ws.path}>
                  {ws.name} ({ws.path})
                </option>
              ))}
              <option value="__custom__">{t('startModal.customPath')}</option>
            </select>
            {isCustomWorkspace && (
              <input
                type="text"
                placeholder="/absolute/path/to/project"
                value={customWorkspace}
                onChange={(e) => setCustomWorkspace(e.target.value)}
                className="mt-2 w-full bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-slate-100 focus:outline-none focus:border-sky-500"
                autoFocus
              />
            )}
          </div>

          {/* Session Title / Goal */}
          <div>
            <label className="text-slate-400">{t('startModal.sessionTitle')}</label>
            <input
              type="text"
              placeholder={t('startModal.sessionTitlePlaceholder')}
              value={sessionTitle}
              onChange={(e) => setSessionTitle(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500"
            />
          </div>

          {/* Coding Provider */}
          <div>
            <label className="text-slate-400">{t('startModal.codingProvider')}</label>
            <select
              value={selectedProvider}
              onChange={(e) => handleProviderChange(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 uppercase focus:outline-none focus:border-sky-500"
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
            <label className="text-slate-400">{t('startModal.profileAccount')}</label>
            <select
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
              className="mt-1 w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-slate-100 focus:outline-none focus:border-sky-500"
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
          <div className="p-3 bg-rose-950/50 border border-rose-800 rounded text-rose-300 text-xs">
            {error}
          </div>
        )}

        <div className="flex items-center justify-end space-x-3 pt-3 border-t border-slate-800">
          <button
            onClick={onClose}
            className="px-3 py-1.5 rounded text-xs font-medium text-slate-400 hover:text-white"
          >
            {t('common.cancel', 'Cancel')}
          </button>
          <button
            disabled={loading || installedProviders.length === 0}
            onClick={handleStart}
            className="px-4 py-1.5 iapro-gradient-bg hover:opacity-95 disabled:opacity-50 text-white rounded text-xs font-semibold shadow-md shadow-purple-950/40 iapro-glow-sm transition flex items-center space-x-1.5"
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>{loading ? t('startModal.starting') : t('startModal.startRuntime')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
