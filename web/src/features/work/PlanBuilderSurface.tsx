import React, { useCallback, useEffect, useState } from 'react';
import {
  ArrowRight,
  Bot,
  ChevronDown,
  ChevronRight,
  Copy,
  FolderGit2,
  History,
  Layers,
  Play,
  Plus,
  Route,
  Sparkles,
} from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Input } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { Agent, ClarificationCheckpoint, MissionRun, PlanPhase, Project, WorkPackage, WorkPlan } from '../../types';
import { clarificationFromError, unresolvedBlocking } from './clarificationModel';

export const PlanBuilderSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onClose?: () => void;
}> = ({ project, agents }) => {
  const [plans, setPlans] = useState<WorkPlan[]>([]);
  const [selectedPlan, setSelectedPlan] = useState<WorkPlan | null>(null);
  const [_loading, setLoading] = useState(false);
  const [autoGoal, setAutoGoal] = useState('');
  const [generating, setGenerating] = useState(false);
  const [compiledPrompt, setCompiledPrompt] = useState<any>(null);
  const [activeRun, setActiveRun] = useState<MissionRun | null>(null);
  const [stepping, setStepping] = useState(false);
  const [expandedPhases, setExpandedPhases] = useState<Record<string, boolean>>({});
  const [clarification, setClarification] = useState<ClarificationCheckpoint | null>(null);
  const [clarificationAnswers, setClarificationAnswers] = useState<Record<string, string>>({});
  const [planError, setPlanError] = useState('');

  const loadPlans = useCallback(async () => {
    try {
      setLoading(true);
      const list = await nexusApi.getPlans(project.id);
      setPlans(list || []);
      if (list && list.length > 0 && !selectedPlan) {
        setSelectedPlan(list[0]);
        // Expand first phase by default
        if (list[0].phases && list[0].phases.length > 0) {
          setExpandedPhases({ [list[0].phases[0].id]: true });
        }
      }
    } catch (e) {
      console.error('Failed to load work plans:', e);
    } finally {
      setLoading(false);
    }
  }, [project.id, selectedPlan]);

  useEffect(() => {
    loadPlans();
  }, [loadPlans]);

  const selectGeneratedPlan = (plan: WorkPlan) => {
    setPlans((prev) => [plan, ...prev.filter((item) => item.id !== plan.id)]);
    setSelectedPlan(plan);
    setAutoGoal('');
    setClarification(null);
    setClarificationAnswers({});
    if (plan.phases && plan.phases.length > 0) {
      setExpandedPhases({ [plan.phases[0].id]: true });
    }
  };

  const handleGenerateAIPlan = async () => {
    if (!autoGoal.trim()) return;
    try {
      setGenerating(true);
      setPlanError('');
      setClarification(null);
      const plan = await nexusApi.createPlan(project.id, {
        goal: autoGoal,
        auto_plan: true,
      });
      selectGeneratedPlan(plan);
    } catch (e) {
      const checkpoint = clarificationFromError(e);
      if (checkpoint) {
        setClarification(checkpoint);
        setClarificationAnswers(Object.fromEntries(checkpoint.unknowns.filter((item) => item.answer).map((item) => [item.key, item.answer || ''])));
      } else {
        setPlanError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setGenerating(false);
    }
  };

  const handleResolveClarification = async () => {
    if (!clarification) return;
    try {
      setGenerating(true);
      setPlanError('');
      const result = await nexusApi.resolveClarification(clarification.id, clarificationAnswers);
      selectGeneratedPlan(result.plan);
    } catch (e) {
      const checkpoint = clarificationFromError(e);
      if (checkpoint) {
        setClarification(checkpoint);
      } else {
        setPlanError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setGenerating(false);
    }
  };

  const handleAddPhase = async () => {
    if (!selectedPlan) return;
    const newPhase: PlanPhase = {
      id: `phase_${Date.now()}`,
      title: `Nova Fase ${selectedPlan.phases.length + 1}`,
      order: selectedPlan.phases.length + 1,
      packages: [
        {
          id: `pkg_${Date.now()}`,
          title: 'Novo Pacote de Trabalho',
          goal: 'Descrever o objetivo técnico deste pacote',
          priority: 'NORMAL',
          status: 'READY',
          dependencies: [],
          role: 'implementer',
          acceptance_criteria: ['Critério de aceitação 1'],
        },
      ],
    };
    const updatedPlan: WorkPlan = {
      ...selectedPlan,
      phases: [...selectedPlan.phases, newPhase],
    };
    try {
      const res = await nexusApi.updatePlan(selectedPlan.id, updatedPlan, 'Fase adicionada');
      setSelectedPlan(res.plan);
      setPlans((prev) => prev.map((p) => (p.id === res.plan.id ? res.plan : p)));
      setExpandedPhases((prev) => ({ ...prev, [newPhase.id]: true }));
    } catch (e) {
      console.error('Failed to add phase:', e);
    }
  };

  const handleAddPackage = async (phaseId: string) => {
    if (!selectedPlan) return;
    const newPkg: WorkPackage = {
      id: `pkg_${Date.now()}`,
      title: 'Novo Pacote de Trabalho',
      goal: 'Descrever objetivo do pacote',
      priority: 'NORMAL',
      status: 'READY',
      dependencies: [],
      role: 'implementer',
      acceptance_criteria: ['Testes passam com sucesso'],
    };
    const updatedPhases = selectedPlan.phases.map((ph) =>
      ph.id === phaseId ? { ...ph, packages: [...ph.packages, newPkg] } : ph
    );
    const updatedPlan: WorkPlan = {
      ...selectedPlan,
      phases: updatedPhases,
    };
    try {
      const res = await nexusApi.updatePlan(selectedPlan.id, updatedPlan, 'Pacote adicionado');
      setSelectedPlan(res.plan);
      setPlans((prev) => prev.map((p) => (p.id === res.plan.id ? res.plan : p)));
    } catch (e) {
      console.error('Failed to add package:', e);
    }
  };

  const handleCompilePrompt = async (packageId: string, phaseId: string) => {
    if (!selectedPlan) return;
    try {
      const res = await nexusApi.compilePackagePrompt(selectedPlan.id, packageId, phaseId);
      setCompiledPrompt(res);
    } catch (e) {
      console.error('Failed to compile prompt:', e);
    }
  };

  const handleLaunchRun = async () => {
    if (!selectedPlan) return;
    try {
      const run = await nexusApi.runPlan(selectedPlan.id, agents[0]?.id);
      setActiveRun(run);
    } catch (e) {
      console.error('Failed to launch run:', e);
    }
  };

  const handleStepRun = async () => {
    if (!activeRun) return;
    try {
      setStepping(true);
      const res = await nexusApi.stepRun(activeRun.id);
      setActiveRun(res.run);
    } catch (e) {
      console.error('Failed to step run:', e);
    } finally {
      setStepping(false);
    }
  };

  return (
    <div className="nx-surface-scroll nx-plan-builder" style={{ padding: '24px', maxWidth: '1400px', margin: '0 auto' }}>
      <div className="nx-page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <span className="nx-eyebrow" style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--color-brand)' }}>
            <Route size={14} /> NEXUS INTELLIGENCE & WORKPLANS
          </span>
          <h1 style={{ fontSize: '24px', fontWeight: 600, margin: '4px 0' }}>Visual Plan Builder & Autonomy Runner</h1>
          <p style={{ color: 'var(--color-text-muted)' }}>
            Decomposição estruturada de objetivos em WorkPackages auditáveis com compilação determinística e portões de verificação.
          </p>
        </div>
        <Badge tone="brand">
          <FolderGit2 size={12} style={{ marginRight: '4px' }} /> {project.name}
        </Badge>
      </div>

      {/* AI Intent Decomposer Bar */}
      <Card style={{ marginBottom: '24px', padding: '16px', border: '1px solid var(--color-brand-border, #3b82f640)', background: 'var(--color-surface-elevated, #18181b)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px', fontWeight: 600 }}>
          <Sparkles size={16} style={{ color: 'var(--color-brand)' }} />
          <span>Nexus Intent Decomposer (AI Planning)</span>
        </div>
        <div style={{ display: 'flex', gap: '12px' }}>
          <Input
            value={autoGoal}
            onChange={setAutoGoal}
            placeholder="Ex: Implementar autenticação JWT RS256 com middleware, migrations SQLite e testes com race detector..."
            style={{ flex: 1 }}
          />
          <Button tone="brand" disabled={!autoGoal.trim() || generating} onClick={handleGenerateAIPlan}>
            <Sparkles size={14} /> {generating ? 'Decompondo...' : 'Gerar Plano Estruturado'}
          </Button>
        </div>
      </Card>

      {planError && (
        <InlineAlert tone="danger" title="Nexus Intelligence could not generate the plan">
          {planError}. Direct AI sessions remain available; configure a real Intelligence provider in Settings to use AI Planning.
        </InlineAlert>
      )}

      {clarification && (
        <Card style={{ margin: '16px 0 24px', padding: '16px' }}>
          <div style={{ marginBottom: 12 }}>
            <strong>Clarification required before autonomous planning</strong>
            <p style={{ margin: '6px 0 0', color: 'var(--color-text-muted)' }}>
              Nexus stopped instead of guessing. These answers become durable facts in the WorkPlan.
            </p>
          </div>
          <div style={{ display: 'grid', gap: 14 }}>
            {unresolvedBlocking(clarification).map((item) => (
              <label key={item.key} style={{ display: 'grid', gap: 6 }}>
                <span><strong>{item.question}</strong></span>
                {item.rationale && <small style={{ color: 'var(--color-text-muted)' }}>{item.rationale}</small>}
                {item.suggested_options && item.suggested_options.length > 0 ? (
                  <select
                    value={clarificationAnswers[item.key] || ''}
                    onChange={(event) => setClarificationAnswers((prev) => ({ ...prev, [item.key]: event.target.value }))}
                  >
                    <option value="">Choose…</option>
                    {item.suggested_options.map((option) => <option key={option} value={option}>{option}</option>)}
                  </select>
                ) : (
                  <Input
                    value={clarificationAnswers[item.key] || ''}
                    onChange={(value) => setClarificationAnswers((prev) => ({ ...prev, [item.key]: value }))}
                    placeholder="Answer required"
                  />
                )}
              </label>
            ))}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 14 }}>
            <Button
              tone="brand"
              disabled={generating || unresolvedBlocking(clarification).some((item) => !(clarificationAnswers[item.key] || '').trim())}
              onClick={() => void handleResolveClarification()}
            >
              <ArrowRight size={14} /> Continue planning
            </Button>
          </div>
        </Card>
      )}

      {/* Main Two-Column Layout: Plan Hierarchy & Inspector/Runner */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 420px', gap: '24px' }}>
        {/* Left Column: Plan Hierarchy */}
        <div>
          {/* Plan Selector & Actions */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <select
                value={selectedPlan?.id || ''}
                onChange={(e) => {
                  const p = plans.find((x) => x.id === e.target.value);
                  if (p) setSelectedPlan(p);
                }}
                style={{
                  padding: '8px 12px',
                  borderRadius: '6px',
                  background: 'var(--color-surface)',
                  color: 'inherit',
                  border: '1px solid var(--color-border)',
                }}
              >
                {plans.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title} (Rev {p.current_revision})
                  </option>
                ))}
              </select>
              {selectedPlan && (
                <Badge tone="default">
                  <History size={12} style={{ marginRight: '4px' }} /> Rev {selectedPlan.current_revision}
                </Badge>
              )}
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
              <Button onClick={handleAddPhase}>
                <Plus size={14} /> Adicionar Fase
              </Button>
              <Button tone="brand" disabled={!selectedPlan} onClick={handleLaunchRun}>
                <Play size={14} /> Executar Plano
              </Button>
            </div>
          </div>

          {/* Phases & Packages Tree */}
          {selectedPlan?.phases?.map((phase, phaseIdx) => {
            const isExpanded = expandedPhases[phase.id] ?? true;
            return (
              <Card key={phase.id} style={{ marginBottom: '16px', padding: '16px' }}>
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}
                  onClick={() =>
                    setExpandedPhases((prev) => ({
                      ...prev,
                      [phase.id]: !isExpanded,
                    }))
                  }
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: 600 }}>
                    {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                    <Layers size={16} style={{ color: 'var(--color-brand)' }} />
                    <span>Fase {phaseIdx + 1}: {phase.title}</span>
                    <Badge tone="default">{phase.packages.length} pacotes</Badge>
                  </div>
                  <Button
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleAddPackage(phase.id);
                    }}
                  >
                    <Plus size={12} /> Pacote
                  </Button>
                </div>

                {isExpanded && (
                  <div style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                    {phase.packages.map((pkg) => (
                      <div
                        key={pkg.id}
                        style={{
                          padding: '14px',
                          borderRadius: '8px',
                          border: '1px solid var(--color-border)',
                          background: 'var(--color-surface)',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                          <div>
                            <strong style={{ fontSize: '15px' }}>{pkg.title}</strong>
                            <div style={{ display: 'flex', gap: '6px', marginTop: '4px' }}>
                              <Badge tone={pkg.priority === 'CRITICAL' ? 'danger' : pkg.priority === 'HIGH' ? 'warning' : 'default'}>
                                {pkg.priority}
                              </Badge>
                              <Badge tone="default">
                                <Bot size={12} style={{ marginRight: '3px' }} /> {pkg.role}
                              </Badge>
                            </div>
                          </div>
                          <Button size="sm" onClick={() => handleCompilePrompt(pkg.id, phase.id)}>
                            <Copy size={12} /> Compilar Prompt
                          </Button>
                        </div>

                        <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', margin: '6px 0' }}>{pkg.goal}</p>

                        {pkg.acceptance_criteria && pkg.acceptance_criteria.length > 0 && (
                          <div style={{ marginTop: '8px', fontSize: '12px' }}>
                            <span style={{ fontWeight: 600, color: 'var(--color-text-muted)' }}>Critérios de Aceitação:</span>
                            <ul style={{ margin: '4px 0 0 16px', padding: 0 }}>
                              {pkg.acceptance_criteria.map((c, i) => (
                                <li key={i}>{c}</li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            );
          })}

          {(!selectedPlan || selectedPlan.phases.length === 0) && (
            <Card style={{ padding: '32px', textAlign: 'center', color: 'var(--color-text-muted)' }}>
              <Route size={32} style={{ margin: '0 auto 12px', opacity: 0.5 }} />
              <p>Nenhum plano ativo. Use o Decompositor AI acima ou crie um plano manualmente.</p>
            </Card>
          )}
        </div>

        {/* Right Column: Prompt Compiler Preview & Live Mission Runner */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {/* Active Mission Runner Panel */}
          {activeRun && (
            <Card style={{ padding: '16px', border: '1px solid var(--color-brand)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <span style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <Play size={14} style={{ color: 'var(--color-brand)' }} /> Autonomous Runner
                </span>
                <Badge tone={activeRun.state === 'COMPLETED' ? 'success' : activeRun.state === 'ESCALATED' ? 'danger' : 'brand'}>
                  {activeRun.state}
                </Badge>
              </div>

              <div style={{ fontSize: '13px', marginBottom: '12px' }}>
                <div>Progresso: Pacote {activeRun.current_pkg_index + 1} de {activeRun.package_runs.length}</div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>
                  Tentativas permitidas: {activeRun.contract.max_retries} | Verificação: {activeRun.contract.require_verification ? 'Ativa' : 'Desativada'}
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '16px' }}>
                {activeRun.package_runs.map((pr, idx) => (
                  <div
                    key={pr.id}
                    style={{
                      padding: '8px 12px',
                      borderRadius: '6px',
                      background: idx === activeRun.current_pkg_index ? 'var(--color-surface-elevated)' : 'transparent',
                      border: idx === activeRun.current_pkg_index ? '1px solid var(--color-brand-border)' : '1px solid var(--color-border)',
                      fontSize: '12px',
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <span>{pr.title}</span>
                    <Badge tone={pr.state === 'VERIFIED' ? 'success' : pr.state === 'FAILED' ? 'danger' : 'default'}>
                      {pr.state} (T{pr.attempt})
                    </Badge>
                  </div>
                ))}
              </div>

              <Button
                tone="brand"
                disabled={activeRun.state === 'COMPLETED' || stepping}
                onClick={handleStepRun}
                style={{ width: '100%' }}
              >
                <ArrowRight size={14} /> {stepping ? 'Executando passo...' : 'Avançar Passo de Autonomia'}
              </Button>
            </Card>
          )}

          {/* Compiled Prompt Inspector */}
          <Card style={{ padding: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
              <span style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Copy size={14} /> Prompt Compilado
              </span>
              {compiledPrompt && (
                <Badge tone="default">{compiledPrompt.estimated_tokens} tokens est.</Badge>
              )}
            </div>

            {compiledPrompt ? (
              <div style={{ fontSize: '12px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <div>
                  <strong style={{ color: 'var(--color-text-muted)' }}>System Prompt Scoped:</strong>
                  <pre
                    style={{
                      padding: '8px',
                      borderRadius: '4px',
                      background: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      maxHeight: '160px',
                      overflowY: 'auto',
                      whiteSpace: 'pre-wrap',
                      marginTop: '4px',
                    }}
                  >
                    {compiledPrompt.system_prompt}
                  </pre>
                </div>

                <div>
                  <strong style={{ color: 'var(--color-text-muted)' }}>User Prompt (Task & Acceptance):</strong>
                  <pre
                    style={{
                      padding: '8px',
                      borderRadius: '4px',
                      background: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      maxHeight: '140px',
                      overflowY: 'auto',
                      whiteSpace: 'pre-wrap',
                      marginTop: '4px',
                    }}
                  >
                    {compiledPrompt.user_prompt}
                  </pre>
                </div>
              </div>
            ) : (
              <p style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>
                Selecione &quot;Compilar Prompt&quot; em qualquer pacote de trabalho para inspecionar o escopo determinístico compilado.
              </p>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
};
