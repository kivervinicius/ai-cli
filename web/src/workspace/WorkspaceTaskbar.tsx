import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FolderGit2, GitBranch, Wifi, BrainCircuit, AlertTriangle } from 'lucide-react';
import { nexus } from '../nexus/api';
import { BranchSwitcherModal } from '../features/projects/BranchSwitcherModal';
import type { Agent, Project } from '../types';

export const WorkspaceTaskbar: React.FC<{
  project?: Project;
  agents?: Agent[];
}> = ({ project, agents = [] }) => {
  const { t } = useTranslation();
  const [branchModalOpen, setBranchModalOpen] = useState(false);
  const [currentBranch, setCurrentBranch] = useState<string>('');
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
  } | null>(null);

  useEffect(() => {
    nexus.getSystemUpdates().then(setSysInfo).catch(() => undefined);
  }, []);

  useEffect(() => {
    if (project?.default_branch) {
      setCurrentBranch(project.default_branch);
    }
  }, [project?.default_branch]);

  const working = agents.filter((a) => a.status === 'WORKING').length;
  const attention = agents.filter((a) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(a.status)
  ).length;

  return (
    <footer className="nx-workspace-statusbar" data-tour="taskbar" role="status" aria-label="Status Bar">
      {/* Left: Project & Branch */}
      <div className="nx-statusbar-left">
        {project && (
          <>
            <span className="nx-statusbar-item" title={project.canonical_path}>
              <FolderGit2 size={12} />
              <span>{project.name}</span>
            </span>
            <button
              type="button"
              className="nx-statusbar-item nx-statusbar-branch nx-statusbar-btn"
              onClick={() => setBranchModalOpen(true)}
              title={t('git.switchBranchTooltip', 'Clique para alternar ou criar branches Git')}
            >
              <GitBranch size={11} />
              <span>{currentBranch || project.default_branch || 'main'}</span>
            </button>

            <BranchSwitcherModal
              open={branchModalOpen}
              onClose={() => setBranchModalOpen(false)}
              project={project}
              onBranchChanged={(b) => setCurrentBranch(b)}
            />
          </>
        )}
      </div>

      {/* Center: Agents status */}
      <div className="nx-statusbar-center">
        {attention > 0 ? (
          <span className="nx-statusbar-item" style={{ color: 'var(--nx-warning, #f59e0b)' }}>
            <AlertTriangle size={12} />
            <span>{t('statusBar.attention', { count: attention, defaultValue: `${attention} precisam de atenção` })}</span>
          </span>
        ) : working > 0 ? (
          <span className="nx-statusbar-item nx-statusbar-working">
            <span className="nx-status-dot nx-status-dot--working" />
            <span>{t('statusBar.working', { count: working })}</span>
          </span>
        ) : (
          <span className="nx-statusbar-item">
            <span className="nx-status-dot nx-status-dot--idle" />
            <span>{t('statusBar.ready')}</span>
          </span>
        )}
      </div>

      {/* Right: Versions & Connection */}
      <div className="nx-statusbar-right">
        <span className="nx-statusbar-item nx-statusbar-version" title="Nexus Core">
          <strong>Nexus</strong> v{sysInfo?.nexus_version || '0.4.1'}
        </span>
        <span className="nx-statusbar-item nx-statusbar-version" title="Orquestrador Maestro">
          <BrainCircuit size={11} />
          <strong>Maestro</strong> v{sysInfo?.maestro_version || '0.1.25'}
        </span>
        <span className="nx-statusbar-item nx-statusbar-local">
          <Wifi size={11} />
          <span>{t('statusBar.local')}</span>
        </span>
      </div>
    </footer>
  );
};
