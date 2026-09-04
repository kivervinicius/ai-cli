import React, { useState } from 'react';
import {
  FolderGit2,
  Gauge,
  History,
  Home,
  Layers,
  LayoutGrid,
  Plus,
  Settings,
  Sparkles,
  TerminalSquare,
  Workflow,
  X,
} from 'lucide-react';
import { Button, IconButton, ContextMenu, contextMenuFromEvent, Tooltip, type ContextMenuPoint } from '../../design-system';
import { AddProjectModal } from './AddProjectModal';
import type { Agent, Project } from '../../types';
import { useTranslation } from 'react-i18next';

export const ProjectRail: React.FC<{
  projects: Project[];
  selected?: Project | null;
  open: boolean;
  onClose: () => void;
  onSelect: (project: Project) => void;
  onCreated: (project: Project) => void;
  onOpenGlobal: (
    kind: 'projects' | 'overview' | 'agents' | 'resources' | 'maestro' | 'sessions' | 'settings' | 'work' | 'missions' | 'terminals'
  ) => void;
  agents?: Agent[];
  onOpenAgent?: (agent: Agent) => void;
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
}> = ({
  projects,
  selected,
  open,
  onClose,
  onSelect,
  onCreated,
  onOpenGlobal,
  agents = [],
  onOpenAgent,
  onNewAgent,
  onNewAISession,
  onProjectShell,
}) => {
  const { t } = useTranslation();
  const [addOpen, setAddOpen] = useState(false);
  const [railMenu, setRailMenu] = useState<
    | { kind: 'project'; project: Project; point: ContextMenuPoint }
    | { kind: 'agent'; agent: Agent; point: ContextMenuPoint }
    | null
  >(null);

  const optionalTools = [
    { id: 'overview', label: t('nav.overview'), icon: Home },
    { id: 'terminals', label: t('nav.terminals'), icon: TerminalSquare },
    { id: 'work', label: 'Composer', icon: Layers },
    { id: 'missions', label: 'Flow Runs', icon: Workflow },
    { id: 'resources', label: t('nav.resources'), icon: Gauge },
    { id: 'sessions', label: t('nav.sessions'), icon: History },
    { id: 'projects', label: t('projectManager.desktopsTitle'), icon: LayoutGrid },
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
            <small>Workspace OS</small>
          </span>
          <IconButton
            className="nx-project-rail__mobile-close"
            label={t('rail.close')}
            onClick={onClose}
          >
            <X size={15} />
          </IconButton>
        </div>

        {/* Primary Workspace Quick Actions — pinned so they never scroll away */}
        <div className="nx-project-rail__actions">
          <button
            type="button"
            className="nx-button"
            data-tone="brand"
            data-size="sm"
            style={{ width: '100%', justifyContent: 'flex-start', padding: '0 10px' }}
            onClick={() => {
              if (onNewAgent) onNewAgent();
              onClose();
            }}
          >
            <Plus size={13} />
            <span>Novo Agente</span>
          </button>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px' }}>
            <Tooltip content="Nova sessão de IA" side="bottom">
              <button
                type="button"
                className="nx-button"
                data-size="sm"
                style={{ fontSize: '11px', padding: '0 6px', justifyContent: 'center' }}
                onClick={() => {
                  if (onNewAISession) onNewAISession();
                  onClose();
                }}
              >
                <Sparkles size={11} />
                <span>Sessão IA</span>
              </button>
            </Tooltip>
            <Tooltip content="Terminal do projeto" side="bottom">
              <button
                type="button"
                className="nx-button"
                data-size="sm"
                style={{ fontSize: '11px', padding: '0 6px', justifyContent: 'center' }}
                onClick={() => {
                  if (onProjectShell) onProjectShell();
                  onClose();
                }}
              >
                <TerminalSquare size={11} />
                <span>Terminal</span>
              </button>
            </Tooltip>
          </div>
        </div>

        {/* Projects Section */}
        <div className="nx-project-rail__heading">
          <span>{t('rail.projects')}</span>
          <IconButton label={t('rail.add')} onClick={() => setAddOpen(true)}>
            <Plus size={14} />
          </IconButton>
        </div>

        <div className="nx-project-list" style={{ maxHeight: '180px', flex: '0 0 auto' }}>
          {projects.map((project) => (
            <button
              type="button"
              key={project.id}
              data-active={selected?.id === project.id ? 'true' : 'false'}
              onClick={() => {
                onSelect(project);
                onClose();
              }}
              onContextMenu={(event) => setRailMenu({ kind: 'project', project, point: contextMenuFromEvent(event) })}
              title={project.canonical_path}
            >
              <span className="nx-project-avatar">
                {(project.name || 'PR').slice(0, 2).toUpperCase()}
              </span>
              <span>
                <strong>{project.name || project.id}</strong>
                <small>{project.default_branch || 'main'}</small>
              </span>
              {selected?.id === project.id && <span className="nx-project-active-dot" />}
            </button>
          ))}
          {projects.length === 0 && <p>{t('rail.empty')}</p>}
        </div>

        {/* Agents Section (Primary for Current Project) */}
        <div className="nx-project-rail__heading" style={{ marginTop: '4px' }}>
          <span>Agentes do Projeto ({agents.length})</span>
          {onNewAgent && (
            <IconButton label="Novo Agente" onClick={onNewAgent}>
              <Plus size={14} />
            </IconButton>
          )}
        </div>

        <div className="nx-project-list" style={{ flex: 1, minHeight: '120px' }}>
          {agents.map((agent) => (
            <button
              type="button"
              key={agent.id}
              onClick={() => {
                if (onOpenAgent) onOpenAgent(agent);
                onClose();
              }}
              onContextMenu={(event) => setRailMenu({ kind: 'agent', agent, point: contextMenuFromEvent(event) })}
              title={`${agent.name} · ${agent.role || 'developer'} (${agent.status})`}
            >
              <span
                className="nx-status-dot"
                style={{
                  background:
                    agent.status === 'WORKING'
                      ? 'var(--nx-success, #10b981)'
                      : agent.status === 'RECOVERABLE' || agent.status === 'WAITING'
                      ? 'var(--nx-warning, #f59e0b)'
                      : agent.status === 'FAILED' || agent.status === 'STALE'
                      ? 'var(--nx-danger, #ef4444)'
                      : 'var(--nx-muted, #64748b)',
                  boxShadow:
                    agent.status === 'WORKING'
                      ? '0 0 6px var(--nx-success, #10b981)'
                      : agent.status === 'RECOVERABLE'
                      ? '0 0 6px var(--nx-warning, #f59e0b)'
                      : 'none',
                }}
              />
              <span>
                <strong>{agent.name}</strong>
                <small>{agent.role || 'developer'}</small>
              </span>
              <TerminalSquare size={12} style={{ color: 'var(--nx-subtle)', marginLeft: 'auto' }} />
            </button>
          ))}
          {agents.length === 0 && (
            <p style={{ padding: '8px 12px', fontSize: '11.5px', color: 'var(--nx-subtle)' }}>
              Nenhum agente ainda. Crie um acima.
            </p>
          )}
        </div>

        {/* Secondary Tools */}
        <div className="nx-project-rail__heading" style={{ marginTop: 'auto', borderTop: '1px solid var(--nx-border)' }}>
          <span>Ferramentas</span>
        </div>
        <div className="nx-project-rail__global" style={{ padding: '4px 8px 8px' }}>
          {optionalTools.map((item) => (
            <button
              type="button"
              key={item.id}
              onClick={() => {
                onOpenGlobal(item.id as any);
                onClose();
              }}
              style={{ height: '30px', fontSize: '11.5px' }}
            >
              <item.icon size={13} />
              <span>{item.label}</span>
            </button>
          ))}
        </div>

        <div className="nx-project-rail__footer">
          <Button size="sm" tone="ghost" onClick={() => setAddOpen(true)}>
            <FolderGit2 size={13} /> {t('rail.addLocal')}
          </Button>
        </div>
      </aside>
      <ContextMenu
        open={railMenu?.point ?? null}
        onClose={() => setRailMenu(null)}
        label={t('workspace.menu')}
        items={
          railMenu?.kind === 'project'
            ? [
                {
                  type: 'item',
                  id: 'open-project',
                  label: t('workspace.openProject'),
                  onSelect: () => {
                    onSelect(railMenu.project);
                    onClose();
                  },
                },
              ]
            : railMenu?.kind === 'agent'
              ? [
                  {
                    type: 'item',
                    id: 'open-agent',
                    label: t('workspace.openAgent'),
                    onSelect: () => {
                      if (onOpenAgent) onOpenAgent(railMenu.agent);
                      onClose();
                    },
                  },
                ]
              : []
        }
      />

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
