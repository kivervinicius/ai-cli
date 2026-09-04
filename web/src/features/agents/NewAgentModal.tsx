import React, { useState } from 'react';
import {
  Bot,
  Database,
  FileText,
  Layers,
  Palette,
  Play,
  Server,
  Shield,
  Terminal,
  Zap,
} from 'lucide-react';
import { Button, Card, Dialog, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, Project } from '../../types';

export interface AgentTypePreset {
  id: string;
  name: string;
  role: string;
  description: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  defaultProvider: string;
  defaultMode: 'Safe' | 'YOLO';
  recommendedSkills: string[];
}

export const CANONICAL_AGENT_TYPES: AgentTypePreset[] = [
  {
    id: 'generalist',
    name: 'Generalist Engineer',
    role: 'generalist',
    description: 'Execução abrangente, tarefas multiárea e coordenação técnica.',
    icon: Bot,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-repo-health'],
  },
  {
    id: 'backend-engineer',
    name: 'Backend Engineer',
    role: 'backend-engineer',
    description: 'APIs, microsserviços, banco de dados e arquitetura server-side.',
    icon: Server,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-database-migrations', 'skill-systematic-debugging'],
  },
  {
    id: 'frontend-engineer',
    name: 'Frontend Engineer',
    role: 'frontend-engineer',
    description: 'UI, design system, acessibilidade, estado e UX de produto.',
    icon: Palette,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-modern-ui-patterns', 'skill-frontend-ux-guardrails'],
  },
  {
    id: 'qa-engineer',
    name: 'QA & Test Engineer',
    role: 'qa-engineer',
    description: 'Testes automatizados E2E, cobertura, regressão e smoke tests.',
    icon: Zap,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-webapp-testing', 'skill-verification-before-completion'],
  },
  {
    id: 'code-reviewer',
    name: 'Code Reviewer',
    role: 'code-reviewer',
    description: 'Auditoria de diffs, segurança estática, conformidade e clean code.',
    icon: Shield,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-saas-security-scan', 'skill-quality-gate'],
  },
  {
    id: 'devops-release',
    name: 'DevOps & Release',
    role: 'devops-release',
    description: 'Pipelines CI/CD, Docker, packaging e confiabilidade operacional.',
    icon: Layers,
    defaultProvider: 'agy',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-release-engineering', 'skill-incident-response'],
  },
  {
    id: 'data-analyst',
    name: 'Data Analyst',
    role: 'data-analyst',
    description: 'Modelagem analítica, consultas SQL e insights de dados.',
    icon: Database,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-unified-analytics'],
  },
  {
    id: 'docs-writer',
    name: 'Technical Writer',
    role: 'docs-writer',
    description: 'Documentação de arquitetura, manuais, ADRs e handoff.',
    icon: FileText,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-adr', 'skill-deep-wiki'],
  },
];

export const NATIVE_PROVIDERS = [
  { id: 'claude', label: 'Claude (nexus claude)', alias: 'nexus claude' },
  { id: 'codex', label: 'Codex (nexus codex)', alias: 'nexus codex' },
  { id: 'agy', label: 'AGY / OpenCode (nexus agy)', alias: 'nexus agy' },
  { id: 'gemini', label: 'Gemini (nexus gemini)', alias: 'nexus gemini' },
  { id: 'cursor', label: 'Cursor (nexus cursor)', alias: 'nexus cursor' },
];

export const NewAgentModal: React.FC<{
  open: boolean;
  onClose: () => void;
  project: Project;
  onCreated: (agent: Agent) => void;
}> = ({ open, onClose, project, onCreated }) => {
  const [selectedType, setSelectedType] = useState<AgentTypePreset>(CANONICAL_AGENT_TYPES[1]);
  const [name, setName] = useState(CANONICAL_AGENT_TYPES[1].name);
  const [origin, setOrigin] = useState<'native' | 'custom'>('native');
  const [provider, setProvider] = useState(CANONICAL_AGENT_TYPES[1].defaultProvider);
  const [mode, setMode] = useState<'Safe' | 'YOLO'>('Safe');
  const [commandTemplate, setCommandTemplate] = useState(
    'docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}'
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const handleSelectType = (preset: AgentTypePreset) => {
    setSelectedType(preset);
    setName(preset.name);
    setProvider(preset.defaultProvider);
    setMode(preset.defaultMode);
  };

  const resolvedPreview =
    origin === 'native'
      ? `nexus ${provider}${mode === 'YOLO' ? ' --yolo' : ''}`
      : commandTemplate
          .replace('{cwd}', project.canonical_path || '/workspace')
          .replace('{args}', mode === 'YOLO' ? '--yolo' : '');

  const handleCreate = async () => {
    if (!name.trim()) return;
    setBusy(true);
    setError('');
    try {
      // 1. Create persistent agent in database
      const created = await nexus.createAgent(project.id, name.trim(), selectedType.role);

      // 2. Configure revision with provider & adapter options
      const extraArgs: string[] = [];
      if (mode === 'YOLO') extraArgs.push('--yolo');

      const configOptions: Record<string, any> = {
        role_type: selectedType.role,
        origin,
        mode,
      };
      if (extraArgs.length > 0) {
        configOptions.extra_args = extraArgs;
      }
      if (origin === 'custom') {
        configOptions.command_template = commandTemplate;
        configOptions.execution_adapter = 'command_template';
      } else {
        configOptions.execution_adapter = 'native_alias';
      }

      await nexus.applyAgentConfig(
        created.id,
        {
          provider: origin === 'custom' ? 'opencode' : provider,
          profile: 'default',
          isolation: 'project',
          workspace: project.canonical_path || undefined,
          options: configOptions,
        }
      );

      onCreated(created);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Novo Agente Especialista" wide>
      <div style={{ display: 'grid', gap: '16px', maxHeight: '72vh', overflowY: 'auto', paddingRight: '4px' }}>
        {error && <Card className="nx-inline-error">{error}</Card>}

        {/* 1. Escolha da Especialidade */}
        <div>
          <label style={{ display: 'block', fontSize: '12px', fontWeight: 700, marginBottom: '8px', color: 'var(--nx-text-soft)' }}>
            1. Selecione a especialidade do agente
          </label>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: '8px' }}>
            {CANONICAL_AGENT_TYPES.map((preset) => {
              const Icon = preset.icon;
              const isSelected = selectedType.id === preset.id;
              return (
                <button
                  type="button"
                  key={preset.id}
                  onClick={() => handleSelectType(preset)}
                  style={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: '10px',
                    padding: '10px',
                    border: '1px solid',
                    borderColor: isSelected ? 'var(--nx-accent)' : 'var(--nx-border)',
                    borderRadius: '8px',
                    background: isSelected ? 'var(--nx-accent-soft)' : 'var(--nx-surface-2)',
                    color: isSelected ? 'var(--nx-text)' : 'var(--nx-text-soft)',
                    cursor: 'pointer',
                    textAlign: 'left',
                    transition: 'all 0.12s',
                  }}
                >
                  <span
                    style={{
                      width: '28px',
                      height: '28px',
                      borderRadius: '6px',
                      display: 'grid',
                      placeItems: 'center',
                      background: isSelected ? 'var(--nx-accent)' : 'var(--nx-surface-3)',
                      color: isSelected ? 'white' : 'var(--nx-accent-text)',
                      flexShrink: 0,
                    }}
                  >
                    <Icon size={15} />
                  </span>
                  <div style={{ minWidth: 0 }}>
                    <strong style={{ fontSize: '12.5px', display: 'block' }}>{preset.name}</strong>
                    <small style={{ fontSize: '11px', color: 'var(--nx-muted)', lineHeight: '1.3', display: 'block' }}>
                      {preset.description}
                    </small>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        {/* 2. Nome do Agente */}
        <div style={{ display: 'grid', gap: '6px' }}>
          <label style={{ fontSize: '12px', fontWeight: 700, color: 'var(--nx-text-soft)' }}>
            2. Nome de exibição
          </label>
          <Input value={name} onChange={setName} placeholder="Nome do agente..." />
        </div>

        {/* 3. Origem e Adaptador de Execução */}
        <div style={{ display: 'grid', gap: '8px' }}>
          <label style={{ fontSize: '12px', fontWeight: 700, color: 'var(--nx-text-soft)' }}>
            3. Origem de execução
          </label>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              type="button"
              className="nx-button"
              data-tone={origin === 'native' ? 'brand' : 'default'}
              data-size="sm"
              onClick={() => setOrigin('native')}
            >
              Nativo (Alias Nexus supervisionado)
            </button>
            <button
              type="button"
              className="nx-button"
              data-tone={origin === 'custom' ? 'brand' : 'default'}
              data-size="sm"
              onClick={() => setOrigin('custom')}
            >
              Customizado (Adaptador / Docker Template)
            </button>
          </div>

          {origin === 'native' ? (
            <div style={{ display: 'grid', gap: '6px', marginTop: '4px' }}>
              <label style={{ fontSize: '11.5px', color: 'var(--nx-muted)' }}>Provedor e Alias Nexus</label>
              <select
                className="nx-select"
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
              >
                {NATIVE_PROVIDERS.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.label}
                  </option>
                ))}
              </select>
            </div>
          ) : (
            <div style={{ display: 'grid', gap: '6px', marginTop: '4px' }}>
              <label style={{ fontSize: '11.5px', color: 'var(--nx-muted)' }}>
                Template de comando com placeholders (<code>{'{cwd}'}</code>, <code>{'{args}'}</code>)
              </label>
              <Input
                value={commandTemplate}
                onChange={setCommandTemplate}
                data-mono="true"
                placeholder='docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}'
              />
              <small style={{ fontSize: '11px', color: 'var(--nx-subtle)' }}>
                Use <code>{'{args}'}</code> como argumento separado. O Nexus não executa shell: operadores como <code>|</code> e <code>;</code> são bloqueados. Para OpenCode no Docker, a autenticação deve existir dentro do mesmo container (<code>opencode auth login &lt;provider&gt;</code>).
              </small>
            </div>
          )}
        </div>

        {/* 4. Modo Inicial */}
        <div style={{ display: 'grid', gap: '6px' }}>
          <label style={{ fontSize: '12px', fontWeight: 700, color: 'var(--nx-text-soft)' }}>
            4. Modo operacional
          </label>
          <div style={{ display: 'flex', gap: '8px' }}>
            {(['Safe', 'YOLO'] as const).map((m) => (
              <button
                key={m}
                type="button"
                className="nx-button"
                data-size="sm"
                data-tone={mode === m ? (m === 'YOLO' ? 'warning' : 'brand') : 'default'}
                onClick={() => setMode(m)}
              >
                {m === 'YOLO' ? '⚡ YOLO (Autonomia contínua)' : '🛡️ Safe (Padrão interativo)'}
              </button>
            ))}
          </div>
        </div>

        {/* 5. Preview do Comando Resolvido */}
        <div style={{ padding: '10px', background: 'var(--nx-surface-3)', borderRadius: '8px', border: '1px solid var(--nx-border)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px', fontSize: '11.5px', color: 'var(--nx-subtle)' }}>
            <Terminal size={12} />
            <span>Comando supervisionado final:</span>
          </div>
          <code style={{ fontSize: '12px', color: 'var(--nx-accent-text)', wordBreak: 'break-all' }}>
            {resolvedPreview}
          </code>
        </div>

        {/* Ações */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', paddingTop: '8px', borderTop: '1px solid var(--nx-border)' }}>
          <Button onClick={onClose} disabled={busy}>Cancelar</Button>
          <Button tone="brand" onClick={handleCreate} disabled={busy || !name.trim()}>
            <Play size={13} /> {busy ? 'Criando...' : 'Criar e Abrir Terminal'}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
