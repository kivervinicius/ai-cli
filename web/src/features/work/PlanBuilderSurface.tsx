import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ArrowRight,
  Bot,
  ChevronDown,
  ChevronRight,
  Copy,
  FolderGit2,
  GripVertical,
  History,
  Layers,
  Play,
  Pause,
  Plus,
  RotateCcw,
  Route,
  XCircle,
  Sparkles,
  Trash2,
} from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Input } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { Agent, ClarificationCheckpoint, MissionRun, MissionSchedule, PlanPhase, PlanRevision, PlanRevisionDiff, Project, WorkPackage, WorkPlan } from '../../types';
import { clarificationFromError, unresolvedBlocking } from './clarificationModel';
import { composerGateForReadiness } from './composerModel';
import { FlowCanvas } from './FlowCanvas';
import { FlowStepInspector } from './FlowStepInspector';
import { expandStepLocalized, flowFromWorkPlan, removeStepLocalized, splitStepLocalized, updateStepLocalized, validateFlow, workPlanFromFlow, type FlowDraftModel, type FlowStepModel } from './flowModel';

export const PlanBuilderSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onOpenAgent?: (agent: Agent) => void;
  onRunCreated?: (run: MissionRun) => void;
  onClose?: () => void;
  initialGoal?: string;
}> = ({ project, agents, onOpenAgent, onRunCreated, initialGoal = '' }) => {
  const [plans, setPlans] = useState<WorkPlan[]>([]);
  const [selectedPlan, setSelectedPlan] = useState<WorkPlan | null>(null);
  const [_loading, setLoading] = useState(false);
  const [autoGoal, setAutoGoal] = useState('');
  const [generating, setGenerating] = useState(false);
  const [compiledPrompt, setCompiledPrompt] = useState<any>(null);
  const [activeRun, setActiveRun] = useState<MissionRun | null>(null);
  const [schedules, setSchedules] = useState<MissionSchedule[]>([]);
  const [scheduleAt, setScheduleAt] = useState('');
  const [revisions, setRevisions] = useState<PlanRevision[]>([]);
  const [revisionDiff, setRevisionDiff] = useState<PlanRevisionDiff | null>(null);
  const [dragPackage, setDragPackage] = useState<{ phaseId: string; packageId: string } | null>(null);
  const [runBusy, setRunBusy] = useState(false);
  const [expandedPhases, setExpandedPhases] = useState<Record<string, boolean>>({});
  const [clarification, setClarification] = useState<ClarificationCheckpoint | null>(null);
  const [clarificationAnswers, setClarificationAnswers] = useState<Record<string, string>>({});
  const [planError, setPlanError] = useState('');
  const [flowDraft, setFlowDraft] = useState<FlowDraftModel | null>(null);
  const [selectedStepId, setSelectedStepId] = useState('');
  const [flowNotice, setFlowNotice] = useState('');
  const [stepComparison, setStepComparison] = useState('');

  useEffect(() => { if (initialGoal.trim()) setAutoGoal((current) => current || initialGoal.trim()); }, [initialGoal]);

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

  useEffect(() => {
    nexusApi.getSchedules(project.id).then(setSchedules).catch((error) => console.error('Failed to load mission schedules:', error));
  }, [project.id]);

  useEffect(() => {
    if (!selectedPlan) { setRevisions([]); return; }
    nexusApi.getPlan(selectedPlan.id).then((detail) => {
      setRevisions(detail.revisions || []);
      if (detail.plan.current_revision !== selectedPlan.current_revision) setSelectedPlan(detail.plan);
    }).catch((error) => console.error('Failed to load plan revisions:', error));
  }, [selectedPlan?.id, selectedPlan?.current_revision]);

  useEffect(() => {
    if (!selectedPlan) { setFlowDraft(null); setSelectedStepId(''); return; }
    const next = flowFromWorkPlan(selectedPlan);
    setFlowDraft(next);
    setSelectedStepId((current) => current && next.steps.some((step) => step.id === current) ? current : (next.steps[0]?.id || ''));
  }, [selectedPlan]);

  useEffect(() => {
    if (!activeRun) return;
    const terminal = new Set(['COMPLETED_VERIFIED', 'FAILED', 'FAILED_NO_PROGRESS', 'FAILED_BUDGET_EXCEEDED', 'FAILED_VERIFICATION', 'CANCELED_BY_USER']);
    if (terminal.has(activeRun.state) || activeRun.state === 'PAUSED' || activeRun.state === 'BLOCKED_NEEDS_USER') return;
    let disposed = false;
    const refresh = async () => {
      try {
        const run = await nexusApi.getRun(activeRun.id);
        if (!disposed) setActiveRun(run);
      } catch (error) {
        if (!disposed) console.error('Failed to refresh mission run:', error);
      }
    };
    const timer = window.setInterval(() => void refresh(), 1500);
    return () => { disposed = true; window.clearInterval(timer); };
  }, [activeRun?.id, activeRun?.state]);

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

  const selectedStep = useMemo(() => flowDraft?.steps.find((step) => step.id === selectedStepId) || null, [flowDraft, selectedStepId]);
  const baselineFlow = useMemo(() => selectedPlan ? flowFromWorkPlan(selectedPlan) : null, [selectedPlan]);
  const flowDirty = Boolean(flowDraft && baselineFlow && JSON.stringify(flowDraft) !== JSON.stringify(baselineFlow));
  const flowErrors = useMemo(() => flowDraft ? validateFlow(flowDraft) : [], [flowDraft]);

  const handleGenerateAIPlan = async () => {
    if (!autoGoal.trim()) return;
    try {
      setGenerating(true);
      setPlanError('');
      setClarification(null);
      const readiness = await nexusApi.getContextReadiness(project.id);
      const gate = composerGateForReadiness(readiness.state);
      if (!gate.canCompose) {
        setPlanError(gate.reason);
        return;
      }
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

  const persistPlan = async (nextPlan: WorkPlan, summary: string) => {
    const res = await nexusApi.updatePlan(nextPlan.id, nextPlan, summary);
    setSelectedPlan(res.plan);
    setPlans((prev) => prev.map((p) => (p.id === res.plan.id ? res.plan : p)));
    setRevisions((prev) => [res.revision, ...prev]);
    return res.plan;
  };

  const patchPackage = async (phaseId: string, packageId: string, patch: Partial<WorkPackage>, summary: string) => {
    if (!selectedPlan) return;
    const next = {
      ...selectedPlan,
      phases: selectedPlan.phases.map((phase) => phase.id === phaseId ? {
        ...phase,
        packages: phase.packages.map((pkg) => pkg.id === packageId ? { ...pkg, ...patch } : pkg),
      } : phase),
    };
    try { await persistPlan(next, summary); } catch (error) { console.error('Failed to update work package:', error); }
  };

  const handleDeletePackage = async (phaseId: string, packageId: string) => {
    if (!selectedPlan) return;
    const next = {
      ...selectedPlan,
      phases: selectedPlan.phases.map((phase) => ({
        ...phase,
        packages: phase.packages
          .filter((pkg) => !(phase.id === phaseId && pkg.id === packageId))
          .map((pkg) => ({ ...pkg, dependencies: (pkg.dependencies || []).filter((dep) => dep !== packageId) })),
      })),
    };
    try { await persistPlan(next, 'Pacote removido e dependências normalizadas'); } catch (error) { console.error(error); }
  };

  const handleDropPackage = async (targetPhaseId: string, targetPackageId: string) => {
    if (!selectedPlan || !dragPackage || (dragPackage.phaseId === targetPhaseId && dragPackage.packageId === targetPackageId)) return;
    let moving: WorkPackage | undefined;
    let phases = selectedPlan.phases.map((phase) => {
      const found = phase.packages.find((pkg) => phase.id === dragPackage.phaseId && pkg.id === dragPackage.packageId);
      if (found) moving = found;
      return { ...phase, packages: phase.packages.filter((pkg) => !(phase.id === dragPackage.phaseId && pkg.id === dragPackage.packageId)) };
    });
    if (!moving) return;
    phases = phases.map((phase) => {
      if (phase.id !== targetPhaseId) return phase;
      const index = phase.packages.findIndex((pkg) => pkg.id === targetPackageId);
      const packages = [...phase.packages];
      packages.splice(index < 0 ? packages.length : index, 0, moving!);
      return { ...phase, packages };
    });
    setDragPackage(null);
    try { await persistPlan({ ...selectedPlan, phases }, 'Pacotes reordenados por drag/drop'); } catch (error) { console.error(error); }
  };

  const handleRestoreRevision = async (revision: number) => {
    if (!selectedPlan || revision === selectedPlan.current_revision) return;
    try {
      const result = await nexusApi.restorePlanRevision(selectedPlan.id, revision);
      setSelectedPlan(result.plan);
      setPlans((prev) => prev.map((plan) => plan.id === result.plan.id ? result.plan : plan));
      setRevisions((prev) => [result.revision, ...prev]);
      setRevisionDiff(null);
    } catch (error) { console.error('Failed to restore plan revision:', error); }
  };

  const handleCompareRevision = async (revision: number) => {
    if (!selectedPlan || revision === selectedPlan.current_revision) return;
    try { setRevisionDiff(await nexusApi.diffPlanRevisions(selectedPlan.id, revision, selectedPlan.current_revision)); }
    catch (error) { console.error('Failed to compare revisions:', error); }
  };

  const allPackages = selectedPlan?.phases.flatMap((phase) => phase.packages) || [];

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

  const saveFlowDraft = async (summary = 'Flow Draft updated') => {
    if (!selectedPlan || !flowDraft) return null;
    const errors = validateFlow(flowDraft);
    if (errors.length) { setPlanError(errors.join(' ')); return null; }
    try {
      const persisted = await persistPlan(workPlanFromFlow(flowDraft, selectedPlan), summary);
      setFlowNotice(`Flow Draft saved as revision ${persisted.current_revision}.`);
      return persisted;
    } catch (error) {
      setPlanError(error instanceof Error ? error.message : String(error));
      return null;
    }
  };

  const changeFlowStep = (patch: Partial<FlowStepModel>) => {
    if (!flowDraft || !selectedStep) return;
    setFlowDraft(updateStepLocalized(flowDraft, selectedStep.id, patch));
    setStepComparison('');
  };

  const handleFlowStepAction = (action: 'REFINE' | 'EXPAND' | 'COMPARE' | 'SPLIT' | 'REMOVE') => {
    if (!flowDraft || !selectedStep) return;
    if (action === 'REFINE') {
      const unique = (items: string[]) => [...new Set(items.map((item) => item.trim()).filter(Boolean))];
      setFlowDraft(updateStepLocalized(flowDraft, selectedStep.id, {
        title: selectedStep.title.trim(), goal: selectedStep.goal.trim(),
        acceptanceCriteria: unique(selectedStep.acceptanceCriteria), relevantPaths: unique(selectedStep.relevantPaths),
        maestroSkills: unique(selectedStep.maestroSkills), verificationRequirements: unique(selectedStep.verificationRequirements),
      }));
      setFlowNotice(`Refined only “${selectedStep.title}”; unrelated Steps were preserved.`);
      return;
    }
    if (action === 'EXPAND') {
      setFlowDraft(expandStepLocalized(flowDraft, selectedStep.id));
      setFlowNotice(`Expanded only “${selectedStep.title}”; unrelated Steps were preserved.`);
      return;
    }
    if (action === 'COMPARE') {
      const previous = revisions.find((revision) => revision.revision < (selectedPlan?.current_revision || 0));
      if (!previous) { setStepComparison('No previous revision is available for this Step.'); return; }
      try {
        const previousPlan = JSON.parse(previous.snapshot_json) as WorkPlan;
        const prior = flowFromWorkPlan(previousPlan).steps.find((step) => step.id === selectedStep.id);
        setStepComparison(prior ? `Revision ${previous.revision}\n${JSON.stringify(prior, null, 2)}\n\nCurrent draft\n${JSON.stringify(selectedStep, null, 2)}` : 'This Step did not exist in the previous revision.');
      } catch { setStepComparison('The previous revision could not be decoded.'); }
      return;
    }
    if (action === 'SPLIT') {
      const stamp = Date.now().toString(36);
      const next = splitStepLocalized(flowDraft, selectedStep.id, [`${selectedStep.id}-a-${stamp}`, `${selectedStep.id}-b-${stamp}`]);
      setFlowDraft(next); setSelectedStepId(next.steps.find((step) => step.id.startsWith(`${selectedStep.id}-a-`))?.id || '');
      setFlowNotice(`Split “${selectedStep.title}” locally and redirected direct dependents.`);
      return;
    }
    const next = removeStepLocalized(flowDraft, selectedStep.id);
    setFlowDraft(next); setSelectedStepId(next.steps[0]?.id || '');
    setFlowNotice(`Removed “${selectedStep.title}” and normalized direct dependency references.`);
  };

  const handleLaunchRun = async () => {
    if (!selectedPlan || !flowDraft) return;
    const errors = validateFlow(flowDraft);
    if (errors.length) { setPlanError(errors.join(' ')); return; }
    try {
      setPlanError('');
      let approved = selectedPlan;
      if (flowDirty || selectedPlan.status === 'DRAFT') {
        const nextDraft = { ...flowDraft, policyStored: true, status: 'READY' as const };
        setFlowDraft(nextDraft);
        const saved = await persistPlan(workPlanFromFlow(nextDraft, selectedPlan), 'Flow approved for execution');
        approved = saved;
      }
      const legacyDefault = approved.phases.flatMap((phase) => phase.packages).some((pkg) => !pkg.assignment_strategy) ? agents[0]?.id : undefined;
      const run = await nexusApi.runPlan(approved.id, approved.current_revision, legacyDefault);
      setActiveRun(run);
      onRunCreated?.(run);
      setFlowNotice(`Flow Run ${run.id} started from revision ${run.plan_revision}.`);
    } catch (e) {
      setPlanError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleTakeControl = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      const run = await nexusApi.takeControlRun(activeRun.id, 'manual takeover from Workspace OS');
      setActiveRun(run);
      const current = run.package_runs[run.current_pkg_index];
      if (current?.assigned_agent && onOpenAgent) {
        const detail = await nexusApi.getAgent(current.assigned_agent);
        onOpenAgent(detail.agent);
      }
    } catch (e) {
      console.error('Failed to take control of mission:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleResumeRun = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      setActiveRun(await nexusApi.returnToMission(activeRun.id));
    } catch (e) {
      console.error('Failed to return mission to autopilot:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleCancelRun = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      setActiveRun(await nexusApi.cancelRun(activeRun.id, 'canceled from Workspace OS'));
    } catch (e) {
      console.error('Failed to cancel mission:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleScheduleAt = async () => {
    if (!selectedPlan || !scheduleAt) return;
    try {
      const item = await nexusApi.schedulePlan(selectedPlan.id, 'AT', { scheduledFor: new Date(scheduleAt).toISOString(), agentId: agents[0]?.id });
      setSchedules((prev) => [item, ...prev]);
      setScheduleAt('');
    } catch (e) {
      console.error('Failed to schedule mission:', e);
    }
  };

  const handleWhenResources = async () => {
    if (!selectedPlan) return;
    try {
      const item = await nexusApi.schedulePlan(selectedPlan.id, 'WHEN_RESOURCES', { agentId: agents[0]?.id });
      setSchedules((prev) => [item, ...prev]);
    } catch (e) {
      console.error('Failed to schedule mission for resource availability:', e);
    }
  };


  return (
    <div className="nx-surface-scroll nx-plan-builder" style={{ padding: '24px', maxWidth: '1400px', margin: '0 auto' }}>
      <div className="nx-page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <span className="nx-eyebrow" style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--color-brand)' }}>
            <Route size={14} /> COMPOSER · FLOW DRAFT
          </span>
          <h1 style={{ fontSize: '24px', fontWeight: 600, margin: '4px 0' }}>Flow Canvas & Step Inspector</h1>
          <p style={{ color: 'var(--color-text-muted)' }}>
            Static approved DAG over the existing WorkPlan/Mission engine. Draft edits never execute work.
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
          <span>Create Flow Draft</span>
        </div>
        <div style={{ display: 'flex', gap: '12px' }}>
          <Input
            value={autoGoal}
            onChange={setAutoGoal}
            placeholder="Ex: Implementar autenticação JWT RS256 com middleware, migrations SQLite e testes com race detector..."
            style={{ flex: 1 }}
          />
          <Button tone="brand" disabled={!autoGoal.trim() || generating} onClick={handleGenerateAIPlan}>
            <Sparkles size={14} /> {generating ? 'Creating…' : 'Generate Flow Draft'}
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
                <Play size={14} /> Approve & Run
              </Button>
            </div>
          </div>

          {flowNotice && <InlineAlert tone="success" title="Flow Draft">{flowNotice}</InlineAlert>}
          {flowErrors.length > 0 && <InlineAlert tone="warning" title="Flow needs attention">{flowErrors.join(' ')}</InlineAlert>}
          {flowDraft && (
            <>
              <div className="nx-flow-editor-toolbar">
                <label>Autonomy policy
                  <select value={flowDraft.policy} onChange={(event) => setFlowDraft({ ...flowDraft, policy: event.target.value as FlowDraftModel['policy'], policyStored: true })}>
                    <option value="GUIDED">Guided</option><option value="AUTONOMOUS">Autonomous</option>
                  </select>
                </label>
                <Badge tone={flowDirty ? 'warning' : 'success'}>{flowDirty ? 'Unsaved Draft' : `Revision ${selectedPlan?.current_revision || 0}`}</Badge>
                <Button size="sm" disabled={!flowDirty || flowErrors.length > 0} onClick={() => void saveFlowDraft()}>{flowDirty ? 'Save Draft' : 'Saved'}</Button>
              </div>
              <FlowCanvas flow={flowDraft} selectedId={selectedStepId} onSelect={setSelectedStepId} />
              {stepComparison && <Card className="nx-flow-step-compare"><pre>{stepComparison}</pre></Card>}
              <details className="nx-flow-compat-details"><summary>Advanced WorkPlan compatibility view</summary>
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
                        draggable
                        onDragStart={() => setDragPackage({ phaseId: phase.id, packageId: pkg.id })}
                        onDragOver={(event) => event.preventDefault()}
                        onDrop={() => void handleDropPackage(phase.id, pkg.id)}
                        style={{
                          padding: '14px',
                          borderRadius: '8px',
                          border: '1px solid var(--color-border)',
                          background: 'var(--color-surface)',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                          <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
                            <GripVertical size={15} style={{ color: 'var(--color-text-muted)', cursor: 'grab', marginTop: 2 }} />
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
                          </div>
                          <div style={{ display: 'flex', gap: 6 }}>
                            <Button size="sm" onClick={() => handleCompilePrompt(pkg.id, phase.id)}>
                            <Copy size={12} /> Compilar Prompt
                          </Button>
                            <Button size="sm" onClick={() => void handleDeletePackage(phase.id, pkg.id)}><Trash2 size={12} /></Button>
                          </div>
                        </div>

                        <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', margin: '6px 0' }}>{pkg.goal}</p>

                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 8, marginTop: 10, fontSize: 11 }}>
                          <label style={{ display: 'grid', gap: 4 }}>Priority
                            <select value={pkg.priority} onChange={(event) => void patchPackage(phase.id, pkg.id, { priority: event.target.value as WorkPackage['priority'] }, 'Prioridade alterada')}>
                              {['CRITICAL','HIGH','NORMAL','LOW'].map((value) => <option key={value} value={value}>{value}</option>)}
                            </select>
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Agent allocation
                            <select value={pkg.agent_allocation || ''} onChange={(event) => void patchPackage(phase.id, pkg.id, { agent_allocation: event.target.value }, 'Agent allocation alterado')}>
                              <option value="">Auto scheduler</option>
                              {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
                            </select>
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Parallel group
                            <input value={pkg.parallel_group || ''} onChange={(event) => void patchPackage(phase.id, pkg.id, { parallel_group: event.target.value }, 'Parallel group alterado')} />
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Dependencies
                            <select multiple value={pkg.dependencies || []} onChange={(event) => void patchPackage(phase.id, pkg.id, { dependencies: Array.from(event.target.selectedOptions).map((option) => option.value) }, 'Dependências alteradas')}>
                              {allPackages.filter((candidate) => candidate.id !== pkg.id).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.title}</option>)}
                            </select>
                          </label>
                        </div>

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
              <p>No Flow Draft yet. Simple work can stay with an Agent.</p>
            </Card>
          )}
              </details>
            </>
          )}
        </div>

        {/* Right Column: Step Inspector + compatibility tools */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {flowDraft && <FlowStepInspector flow={flowDraft} step={selectedStep} agents={agents} onChange={changeFlowStep} onAction={handleFlowStepAction} />}
          <Card style={{ padding: '16px' }}>
            <div style={{ fontWeight: 600, marginBottom: 10 }}><History size={14} /> Revision History</div>
            <div style={{ display: 'grid', gap: 6 }}>
              {(revisions || []).slice(0, 6).map((revision) => (
                <div key={revision.id} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', gap: 6, alignItems: 'center', fontSize: 11 }}>
                  <span>Rev {revision.revision} · {revision.change_summary}</span>
                  <Button size="sm" disabled={revision.revision === selectedPlan?.current_revision} onClick={() => void handleCompareRevision(revision.revision)}>Diff</Button>
                  <Button size="sm" disabled={revision.revision === selectedPlan?.current_revision} onClick={() => void handleRestoreRevision(revision.revision)}>Restore</Button>
                </div>
              ))}
              {revisionDiff && (
                <div style={{ padding: 8, border: '1px solid var(--color-border)', borderRadius: 6, fontSize: 11 }}>
                  Rev {revisionDiff.from_revision} → {revisionDiff.to_revision}: +{revisionDiff.added_packages.length} / -{revisionDiff.removed_packages.length} / ~{revisionDiff.changed_packages.length} packages
                </div>
              )}
            </div>
          </Card>

          <Card style={{ padding: '16px' }}>
            <div style={{ fontWeight: 600, marginBottom: 10 }}>Mission Scheduling</div>
            <div style={{ display: 'grid', gap: 8 }}>
              <input
                type="datetime-local"
                value={scheduleAt}
                onChange={(event) => setScheduleAt(event.target.value)}
                style={{ padding: '8px 10px', borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit' }}
              />
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <Button disabled={!selectedPlan || !scheduleAt} onClick={handleScheduleAt}>Schedule date/time</Button>
                <Button disabled={!selectedPlan} onClick={handleWhenResources}>When resources free</Button>
              </div>
              {activeRun && selectedPlan && (
                <Button onClick={async () => {
                  const item = await nexusApi.schedulePlan(selectedPlan.id, 'AFTER_RUN', { afterRunId: activeRun.id, agentId: agents[0]?.id });
                  setSchedules((prev) => [item, ...prev]);
                }}>Run after current Mission</Button>
              )}
              {(schedules || []).slice(0, 3).map((item) => (
                <div key={item.id} style={{ fontSize: 11, color: 'var(--color-text-muted)', display: 'flex', justifyContent: 'space-between' }}>
                  <span>{item.mode}{item.scheduled_for ? ` · ${new Date(item.scheduled_for).toLocaleString()}` : ''}</span>
                  <Badge tone={item.status === 'FAILED' ? 'danger' : item.status === 'COMPLETED' ? 'success' : 'default'}>{item.status}</Badge>
                </div>
              ))}
            </div>
          </Card>

          {/* Legacy Mission Runner controls remain available without competing with the canonical Flow Run surface. */}
          {activeRun && (
            <details className="nx-flow-compat-details nx-flow-run-compat">
              <summary>Advanced Mission Runner compatibility view · {activeRun.state}</summary>
              <Card style={{ padding: '16px', border: '1px solid var(--color-brand)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <span style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <Play size={14} style={{ color: 'var(--color-brand)' }} /> Autonomous Runner
                </span>
                <Badge tone={activeRun.state === 'COMPLETED_VERIFIED' ? 'success' : activeRun.state.startsWith('FAILED') || activeRun.state === 'BLOCKED_NEEDS_USER' ? 'danger' : activeRun.state === 'PAUSED' ? 'warning' : 'brand'}>
                  {activeRun.state}
                </Badge>
              </div>

              <div style={{ fontSize: '13px', marginBottom: '12px' }}>
                <div>Progresso: {activeRun.package_runs.filter((pkg) => pkg.state === 'VERIFIED').length} de {activeRun.package_runs.length} pacotes verificados</div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>Snapshot: {activeRun.execution_snapshot_id || '—'} · Rev {activeRun.plan_revision} · Iteração {activeRun.total_iterations}/{activeRun.contract.max_total_iterations}</div>
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

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                {activeRun.state === 'PAUSED' || activeRun.state === 'BLOCKED_NEEDS_USER' ? (
                  <Button tone="brand" disabled={runBusy} onClick={handleResumeRun}>
                    <RotateCcw size={14} /> Return to Mission
                  </Button>
                ) : (
                  <Button disabled={runBusy || activeRun.state === 'COMPLETED_VERIFIED' || activeRun.state.startsWith('FAILED') || activeRun.state === 'CANCELED_BY_USER'} onClick={handleTakeControl}>
                    <Pause size={14} /> Take Control
                  </Button>
                )}
                <Button disabled={runBusy || activeRun.state === 'COMPLETED_VERIFIED' || activeRun.state === 'CANCELED_BY_USER'} onClick={handleCancelRun}>
                  <XCircle size={14} /> Cancel
                </Button>
              </div>
              <p style={{ fontSize: 11, color: 'var(--color-text-muted)', margin: '10px 0 0' }}>
                The runner advances automatically. Take Control pauses autonomy and opens the current persistent Agent; Return to Mission resumes from the persisted checkpoint.
              </p>
            </Card>
            </details>
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
