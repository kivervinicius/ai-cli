import React, { useState } from 'react';
import {
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
import {
  IconButton,
  ContextMenu,
  contextMenuFromEvent,
  Tooltip,
  type ContextMenuPoint,
} from '../../design-system';
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
    kind:
      | 'projects'
      | 'overview'
      | 'agents'
      | 'resources'
      | 'maestro'
      | 'sessions'
      | 'settings'
      | 'work'
      | 'missions'
      | 'terminals',
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

  // Collapsible Accordion Sections with LocalStorage Persistence
  const [projectsExpanded, setProjectsExpanded] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('nx_rail_projects_open');
      return saved !== null ? saved === 'true' : true;
    } catch {
      return true;
    }
  });
  const [agentsExpanded, setAgentsExpanded] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('nx_rail_agents_open');
      return saved !== null ? saved === 'true' : true;
    } catch {
      return true;
    }
  });
  const [toolsExpanded, setToolsExpanded] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('nx_rail_tools_open');
      return saved !== null ? saved === 'true' : true;
    } catch {
      return true;
    }
  });

  // Inline filter query
  const [searchQuery, setSearchQuery] = useState('');

  const toggleProjects = () => {
    setProjectsExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('nx_rail_projects_open', String(next));
      } catch {}
      return next;
    });
  };

  const toggleAgents = () => {
    setAgentsExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('nx_rail_agents_open', String(next));
      } catch {}
      return next;
    });
  };

  const toggleTools = () => {
    setToolsExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('nx_rail_tools_open', String(next));
      } catch {}
      return next;
    });
  };

  const q = searchQuery.trim().toLowerCase();
  const filteredProjects = q
    ? projects.filter(
        (p) =>
          (p.name || p.id).toLowerCase().includes(q) ||
          (p.canonical_path || '').toLowerCase().includes(q),
      )
    : projects;

  const filteredAgents = q
    ? agents.filter(
        (a) => (a.name || '').toLowerCase().includes(q) || (a.role || '').toLowerCase().includes(q),
      )
    : agents;

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
          <span className="nx-brand-mark">
            <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
          </span>
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

        {/* Compact Search Filter for Rail when user has multiple items */}
        {(projects.length > 3 || agents.length > 3 || searchQuery) && (
          <div className="nx-project-rail__search-wrap">
            <input
              type="search"
              className="nx-project-rail__search-input"
              placeholder="Filtrar projetos ou agentes..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              aria-label="Filtrar projetos ou agentes"
            />
          </div>
        )}

        {/* Projects Accordion Section */}
        <div
          className="nx-rail-accordion-section"
          data-expanded={projectsExpanded ? 'true' : 'false'}
        >
          <div className="nx-project-rail__heading nx-rail-section-header">
            <button
              type="button"
              className="nx-rail-header-label"
              onClick={toggleProjects}
              aria-expanded={projectsExpanded}
            >
              <span className="nx-rail-chevron">{projectsExpanded ? '▾' : '▸'}</span>
              <span>{t('rail.projects')}</span>
              <span className="nx-rail-count">({filteredProjects.length})</span>
            </button>
            <span className="nx-rail-header-actions" onClick={(e) => e.stopPropagation()}>
              <IconButton label={t('rail.add')} onClick={() => setAddOpen(true)}>
                <Plus size={13} />
              </IconButton>
            </span>
          </div>

          {projectsExpanded && (
            <div
              className="nx-project-list nx-rail-scrollable"
              style={{ maxHeight: '160px', flex: '0 0 auto' }}
            >
              {filteredProjects.map((project) => (
                <button
                  type="button"
                  key={project.id}
                  className="nx-rail-item-btn"
                  data-active={selected?.id === project.id ? 'true' : 'false'}
                  onClick={() => {
                    onSelect(project);
                    onClose();
                  }}
                  onContextMenu={(event) =>
                    setRailMenu({ kind: 'project', project, point: contextMenuFromEvent(event) })
                  }
                  title={project.canonical_path}
                >
                  <span className="nx-project-avatar nx-rail-avatar-compact">
                    {(project.name || 'PR').slice(0, 2).toUpperCase()}
                  </span>
                  <span className="nx-rail-item-copy">
                    <strong>{project.name || project.id}</strong>
                    <small>{project.default_branch || 'main'}</small>
                  </span>
                  {selected?.id === project.id && <span className="nx-project-active-dot" />}
                </button>
              ))}
              {filteredProjects.length === 0 && (
                <p className="nx-rail-empty-msg">
                  {searchQuery ? 'Nenhum projeto encontrado' : t('rail.empty')}
                </p>
              )}
            </div>
          )}
        </div>

        {/* Agents Accordion Section (Primary Workload) */}
        <div
          className="nx-rail-accordion-section nx-rail-section-agents"
          data-expanded={agentsExpanded ? 'true' : 'false'}
        >
          <div className="nx-project-rail__heading nx-rail-section-header">
            <button
              type="button"
              className="nx-rail-header-label"
              onClick={toggleAgents}
              aria-expanded={agentsExpanded}
            >
              <span className="nx-rail-chevron">{agentsExpanded ? '▾' : '▸'}</span>
              <span>Agentes</span>
              <span className="nx-rail-count">({filteredAgents.length})</span>
            </button>
            <span className="nx-rail-header-actions" onClick={(e) => e.stopPropagation()}>
              {onNewAgent && (
                <IconButton label="Novo Agente" onClick={onNewAgent}>
                  <Plus size={13} />
                </IconButton>
              )}
            </span>
          </div>

          {agentsExpanded && (
            <div
              className="nx-project-list nx-rail-scrollable nx-agents-list-scroll"
              style={{ flex: 1, minHeight: '100px' }}
            >
              {filteredAgents.map((agent) => (
                <button
                  type="button"
                  key={agent.id}
                  className="nx-rail-item-btn"
                  onClick={() => {
                    if (onOpenAgent) onOpenAgent(agent);
                    onClose();
                  }}
                  onContextMenu={(event) =>
                    setRailMenu({ kind: 'agent', agent, point: contextMenuFromEvent(event) })
                  }
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
                  <span className="nx-rail-item-copy">
                    <strong>{agent.name}</strong>
                    <small>{agent.role || 'developer'}</small>
                  </span>
                  <TerminalSquare
                    size={12}
                    style={{ color: 'var(--nx-subtle)', marginLeft: 'auto' }}
                  />
                </button>
              ))}
              {filteredAgents.length === 0 && (
                <p className="nx-rail-empty-msg">
                  {searchQuery ? 'Nenhum agente com esse filtro.' : 'Nenhum agente ainda.'}
                </p>
              )}
            </div>
          )}
        </div>

        {/* Secondary Tools Accordion Section (Collapsible & Compact Grid) */}
        <div
          className="nx-rail-accordion-section nx-rail-section-tools"
          data-expanded={toolsExpanded ? 'true' : 'false'}
        >
          <div
            className="nx-project-rail__heading nx-rail-section-header"
            style={{ borderTop: '1px solid var(--nx-border)' }}
          >
            <button
              type="button"
              className="nx-rail-header-label"
              onClick={toggleTools}
              aria-expanded={toolsExpanded}
            >
              <span className="nx-rail-chevron">{toolsExpanded ? '▾' : '▸'}</span>
              <span>Ferramentas</span>
            </button>
          </div>

          {toolsExpanded && (
            <div className="nx-project-rail__global nx-rail-tools-grid">
              {optionalTools.map((item) => (
                <button
                  type="button"
                  key={item.id}
                  className="nx-rail-tool-btn"
                  onClick={() => {
                    onOpenGlobal(item.id as any);
                    onClose();
                  }}
                  title={item.label}
                >
                  <item.icon size={13} />
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          )}
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
