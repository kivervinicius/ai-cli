import React, { useEffect, useState } from 'react';
import { nexus } from './api';
import { AgentTerminal } from './AgentTerminal';
import { Project, Agent } from '../types';
import { Button, Badge, Card, Input, EmptyState, Spinner } from '../ui/primitives';

const agentTone = (status: string) =>
  status === 'WORKING'
    ? 'success'
    : status === 'WAITING' || status === 'RATE_LIMITED' || status === 'RECOVERABLE'
      ? 'warning'
      : status === 'FAILED' || status === 'STALE'
        ? 'danger'
        : status === 'RECOVERING'
          ? 'brand'
          : 'default';

const continuityLabel = (c: string) =>
  c === 'NATIVE_RESUME_VERIFIED'
    ? 'resume verified'
    : c === 'NATIVE_RESUME_UNVERIFIED'
      ? 'resume unverified'
      : c === 'CONTEXT_RECOVERED_NEW_SESSION'
        ? 'new session'
        : c === 'REATTACHED_SAME_RUNTIME'
          ? 'reattached'
          : c === 'LIVE_SAME_RUNTIME'
            ? 'same runtime'
            : c;

export const ProjectsPage: React.FC = () => {
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selected, setSelected] = useState<Project | null>(null);
  const [pathInput, setPathInput] = useState('');
  const [agentName, setAgentName] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [terminalAgent, setTerminalAgent] = useState<string | null>(null);
  const [startingAgent, setStartingAgent] = useState<string | null>(null);

  const loadProjects = async () => {
    try {
      const list = await nexus.listProjects();
      setProjects(list);
      if (!selected && list.length > 0) {
        setSelected(list[0]);
      }
    } catch (e: any) {
      setError(e.message || 'failed to load projects');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProjects();
  }, []);

  useEffect(() => {
    if (!selected) {
      setAgents([]);
      return;
    }
    nexus
      .listAgents(selected.id)
      .then(setAgents)
      .catch(() => setAgents([]));
  }, [selected]);

  const addProject = async () => {
    setError('');
    try {
      await nexus.createProject(pathInput.trim());
      setPathInput('');
      await loadProjects();
    } catch (e: any) {
      setError(e.message || 'failed to add project');
    }
  };

  const addAgent = async () => {
    if (!selected || !agentName.trim()) return;
    try {
      await nexus.createAgent(selected.id, agentName.trim());
      setAgentName('');
      const list = await nexus.listAgents(selected.id);
      setAgents(list);
    } catch (e: any) {
      setError(e.message || 'failed to create agent');
    }
  };

  const startAgent = async (agent: Agent) => {
    setStartingAgent(agent.id);
    setError('');
    try {
      await nexus.startAgent(agent.id);
      const list = await nexus.listAgents(agent.project_id);
      setAgents(list);
      setTerminalAgent(agent.id);
    } catch (e: any) {
      setError(e.message || 'failed to start agent');
    } finally {
      setStartingAgent(null);
    }
  };

  const stopAgent = async (agent: Agent) => {
    try {
      await nexus.stopAgent(agent.id);
      const list = await nexus.listAgents(agent.project_id);
      setAgents(list);
    } catch (e: any) {
      setError(e.message || 'failed to stop agent');
    }
  };

  const recoverAgent = async (agent: Agent) => {
    setError('');
    try {
      await nexus.recoverAgent(agent.id);
      const list = await nexus.listAgents(agent.project_id);
      setAgents(list);
      setTerminalAgent(agent.id);
    } catch (e: any) {
      setError(e.message || 'failed to recover agent');
    }
  };

  const deleteProject = async (p: Project) => {
    try {
      await nexus.deleteProject(p.id);
      setSelected(null);
      await loadProjects();
    } catch (e: any) {
      setError(e.message || 'failed to delete project');
    }
  };

  if (loading) return <Spinner label="Loading projects…" />;

  return (
    <div className="grid grid-cols-[300px_1fr] gap-4 h-full min-h-0">
      {/* Projects rail */}
      <div className="flex flex-col gap-3 border-r border-slate-800/70 pr-4 min-h-0">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-200">Projects</h2>
          <Badge tone="brand">Project-first</Badge>
        </div>
        <div className="flex gap-2">
          <Input value={pathInput} onChange={setPathInput} placeholder="/path/to/project" onEnter={addProject} mono />
          <Button tone="brand" onClick={addProject} disabled={!pathInput.trim()}>
            Add
          </Button>
        </div>
        {error && <p className="text-xs text-red-400">{error}</p>}

        <div className="flex flex-col gap-2 overflow-y-auto">
          {projects.length === 0 && (
            <EmptyState title="No Projects yet" hint="Add a project directory to start working with persistent agents." />
          )}
          {projects.map((p) => (
            <Card
              key={p.id}
              onClick={() => setSelected(p)}
              className={`p-3 ${selected?.id === p.id ? 'border-indigo-700/70 bg-indigo-950/30' : ''}`}
            >
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-slate-100">{p.name}</div>
                  <div className="truncate text-[11px] font-mono text-slate-500">{p.canonical_path}</div>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <Badge tone="default">{p.maestro_mode}</Badge>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      deleteProject(p);
                    }}
                    className="text-slate-600 hover:text-red-400 px-1"
                    title="Delete project"
                  >
                    ×
                  </button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      </div>

      {/* Selected project */}
      <div className="flex flex-col gap-4 min-h-0">
        {!selected ? (
          <EmptyState title="Select a Project" hint="Choose a project on the left to manage its agents and terminals." />
        ) : (
          <>
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-lg font-semibold text-slate-100">{selected.name}</h1>
                <p className="text-xs font-mono text-slate-500">
                  {selected.slug} · {selected.default_branch} · Maestro {selected.maestro_mode}
                </p>
              </div>
            </div>

            <div className="flex gap-2 items-center">
              <Input value={agentName} onChange={setAgentName} placeholder="New agent name (e.g. Backend Developer)" onEnter={addAgent} />
              <Button tone="brand" onClick={addAgent} disabled={!agentName.trim()}>
                Create Agent
              </Button>
            </div>

            <div className="flex flex-col gap-2 overflow-y-auto">
              {agents.length === 0 && (
                <EmptyState title="No Agents" hint="Create a persistent agent. An Agent survives runtime restarts, account changes and provider switches." />
              )}
              {agents.map((a) => (
                <Card key={a.id} className="flex items-center justify-between gap-3 p-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-slate-100 truncate">{a.name}</span>
                      <Badge tone={agentTone(a.status)}>{a.status}</Badge>
                      {a.continuity_status && a.continuity_status !== 'LIVE_SAME_RUNTIME' && (
                        <Badge tone="info">{continuityLabel(a.continuity_status)}</Badge>
                      )}
                    </div>
                    <div className="text-[11px] font-mono text-slate-500 truncate">
                      {a.id} · continuity {a.continuity_status}
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {a.status === 'RECOVERABLE' ? (
                      <Button size="sm" tone="warning" onClick={() => recoverAgent(a)}>
                        Recover
                      </Button>
                    ) : a.status === 'WORKING' || a.status === 'STARTING' || a.status === 'RECOVERING' ? (
                      <Button size="sm" tone="danger" onClick={() => stopAgent(a)}>
                        Stop
                      </Button>
                    ) : (
                      <Button size="sm" tone="success" onClick={() => startAgent(a)} disabled={startingAgent === a.id}>
                        {startingAgent === a.id ? 'Starting…' : 'Start'}
                      </Button>
                    )}
                    <Button size="sm" onClick={() => setTerminalAgent(a.id)} disabled={a.status !== 'WORKING' && a.status !== 'STARTING' && a.status !== 'RECOVERING'}>
                      Terminal
                    </Button>
                  </div>
                </Card>
              ))}
            </div>

            {terminalAgent && (
              <div className="flex-1 min-h-0 border-t border-slate-800/70 pt-3">
                <AgentTerminal agentId={terminalAgent} onClose={() => setTerminalAgent(null)} />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
