import React, { useState } from 'react';
import {
  Bot,
  BrainCircuit,
  FolderGit2,
  Gauge,
  Home,
  LayoutGrid,
  Plus,
  Settings,
  X,
} from 'lucide-react';
import { Button, IconButton } from '../../design-system';
import { AddProjectModal } from './AddProjectModal';
import type { Project } from '../../types';
import { useTranslation } from 'react-i18next';

export const ProjectRail: React.FC<{
  projects: Project[];
  selected?: Project | null;
  open: boolean;
  onClose: () => void;
  onSelect: (project: Project) => void;
  onCreated: (project: Project) => void;
  onOpenGlobal: (
    kind: 'projects' | 'overview' | 'agents' | 'resources' | 'maestro' | 'sessions' | 'settings'
  ) => void;
}> = ({ projects, selected, open, onClose, onSelect, onCreated, onOpenGlobal }) => {
  const { t } = useTranslation();
  const [addOpen, setAddOpen] = useState(false);

  const global = [
    { id: 'projects', label: t('projectManager.desktopsTitle'), icon: LayoutGrid },
    { id: 'overview', label: t('nav.overview'), icon: Home },
    { id: 'agents', label: t('nav.agents'), icon: Bot },
    { id: 'resources', label: t('nav.resources'), icon: Gauge },
    { id: 'maestro', label: 'Maestro', icon: BrainCircuit },
    { id: 'settings', label: t('nav.settings'), icon: Settings },
  ] as const;

  return (
    <>
      <aside
        className="nx-project-rail"
        data-open={open ? 'true' : 'false'}
        data-tour="projects"
        aria-label={t('rail.aria')}
      >
        <div className="nx-project-rail__brand">
          <span className="nx-brand-mark">N</span>
          <span>
            <strong>IAPro Nexus</strong>
            <small>Powered by Maestro</small>
          </span>
          <IconButton
            className="nx-project-rail__mobile-close"
            label={t('rail.close')}
            onClick={onClose}
          >
            <X size={15} />
          </IconButton>
        </div>

        <div className="nx-project-rail__global">
          {global.map((item) => (
            <button
              type="button"
              key={item.id}
              onClick={() => {
                onOpenGlobal(item.id);
                onClose();
              }}
            >
              <item.icon size={15} />
              <span>{item.label}</span>
            </button>
          ))}
        </div>

        <div className="nx-project-rail__heading">
          <span>{t('rail.projects')}</span>
          <IconButton label={t('rail.add')} onClick={() => setAddOpen(true)}>
            <Plus size={14} />
          </IconButton>
        </div>

        <div className="nx-project-list">
          {projects.map((project) => (
            <button
              type="button"
              key={project.id}
              data-active={selected?.id === project.id ? 'true' : 'false'}
              onClick={() => {
                onSelect(project);
                onClose();
              }}
              title={project.canonical_path}
            >
              <span className="nx-project-avatar">
                {(project.name || '').slice(0, 2).toUpperCase()}
              </span>
              <span>
                <strong>{project.name}</strong>
                <small>{project.default_branch || 'main'}</small>
              </span>
              {selected?.id === project.id && <span className="nx-project-active-dot" />}
            </button>
          ))}
          {projects.length === 0 && <p>{t('rail.empty')}</p>}
        </div>

        <div className="nx-project-rail__footer">
          <Button size="sm" tone="ghost" onClick={() => setAddOpen(true)}>
            <FolderGit2 size={13} /> {t('rail.addLocal')}
          </Button>
        </div>
      </aside>

      <div
        className="nx-project-rail-overlay"
        data-open={open ? 'true' : 'false'}
        onClick={onClose}
      />

      <AddProjectModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onCreated={(project) => {
          onCreated(project);
          setAddOpen(false);
        }}
      />
    </>
  );
};
