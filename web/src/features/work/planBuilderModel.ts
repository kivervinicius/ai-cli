import type { WorkPackage, WorkPlan } from '../../types';
import { normalizeWorkPlan } from '../../nexus/workPlan';

export { normalizeWorkPlan } from '../../nexus/workPlan';

function clonePlan(plan: WorkPlan): WorkPlan {
  const normalized = normalizeWorkPlan(plan);
  return {
    ...normalized,
    phases: normalized.phases.map((phase) => ({
      ...phase,
      packages: phase.packages.map((pkg) => ({
        ...pkg,
        dependencies: [...(pkg.dependencies || [])],
        acceptance_criteria: [...(pkg.acceptance_criteria || [])],
        maestro_gates: pkg.maestro_gates ? [...pkg.maestro_gates] : undefined,
        shared_artifacts: pkg.shared_artifacts ? [...pkg.shared_artifacts] : undefined,
      })),
    })),
  };
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean)));
}

export function splitPackage(plan: WorkPlan, phaseId: string, packageId: string, newPackageId: string): WorkPlan {
  const next = clonePlan(plan);
  const phase = next.phases.find((item) => item.id === phaseId);
  if (!phase) throw new Error(`phase ${phaseId} not found`);
  const index = phase.packages.findIndex((item) => item.id === packageId);
  if (index < 0) throw new Error(`package ${packageId} not found`);
  if (next.phases.some((item) => item.packages.some((pkg) => pkg.id === newPackageId))) {
    throw new Error(`package ${newPackageId} already exists`);
  }
  const source = phase.packages[index];
  const criteria = source.acceptance_criteria || [];
  if (criteria.length < 2) {
    throw new Error('split requires at least two acceptance criteria');
  }
  const midpoint = Math.ceil(criteria.length / 2);
  const firstAcceptance = criteria.slice(0, midpoint);
  const secondAcceptance = criteria.slice(midpoint);
  const first: WorkPackage = { ...source, title: `${source.title} · Parte 1`, acceptance_criteria: firstAcceptance };
  const second: WorkPackage = {
    ...source,
    id: newPackageId,
    title: `${source.title} · Parte 2`,
    dependencies: [source.id],
    acceptance_criteria: secondAcceptance,
    compiled_prompt: undefined,
    status: 'READY',
  };
  phase.packages.splice(index, 1, first, second);

  // Downstream work that depended on the original package must wait for both halves.
  for (const candidatePhase of next.phases) {
    for (const pkg of candidatePhase.packages) {
      if (pkg.id === second.id) continue;
      pkg.dependencies = unique((pkg.dependencies || []).map((dep) => dep === source.id ? second.id : dep));
    }
  }
  second.dependencies = [source.id];
  return next;
}

export function mergePackages(plan: WorkPlan, phaseId: string, survivorId: string, mergedId: string): WorkPlan {
  if (survivorId === mergedId) throw new Error('cannot merge a package into itself');
  const next = clonePlan(plan);
  const phase = next.phases.find((item) => item.id === phaseId);
  if (!phase) throw new Error(`phase ${phaseId} not found`);
  const survivorIndex = phase.packages.findIndex((item) => item.id === survivorId);
  const mergedIndex = phase.packages.findIndex((item) => item.id === mergedId);
  if (survivorIndex < 0 || mergedIndex < 0) throw new Error('merge packages must be in the selected phase');
  const survivor = phase.packages[survivorIndex];
  const merged = phase.packages[mergedIndex];

  survivor.title = survivor.title === merged.title ? survivor.title : `${survivor.title} + ${merged.title}`;
  survivor.goal = unique([survivor.goal, merged.goal]).join('\n\n');
  survivor.acceptance_criteria = unique([...(survivor.acceptance_criteria || []), ...(merged.acceptance_criteria || [])]);
  survivor.shared_artifacts = unique([...(survivor.shared_artifacts || []), ...(merged.shared_artifacts || [])]);
  survivor.dependencies = unique([...(survivor.dependencies || []), ...(merged.dependencies || [])].filter((dep) => dep !== survivorId && dep !== mergedId));
  survivor.compiled_prompt = undefined;
  survivor.status = 'READY';

  phase.packages = phase.packages.filter((item) => item.id !== mergedId);
  for (const candidatePhase of next.phases) {
    for (const pkg of candidatePhase.packages) {
      pkg.dependencies = unique((pkg.dependencies || []).map((dep) => dep === mergedId ? survivorId : dep).filter((dep) => dep !== pkg.id));
    }
  }
  return next;
}

export function setProviderLock(rawRequirements: string | undefined, provider: string, profile: string): string {
  let requirements: Record<string, unknown> = {};
  if (rawRequirements?.trim()) {
    try {
      const parsed = JSON.parse(rawRequirements);
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) requirements = { ...parsed };
    } catch {
      throw new Error('task_requirements must contain valid JSON before setting a provider lock');
    }
  }
  if (provider.trim()) requirements.provider_lock = provider.trim(); else delete requirements.provider_lock;
  if (profile.trim()) requirements.profile_lock = profile.trim(); else delete requirements.profile_lock;
  return JSON.stringify(requirements);
}

export function getProviderLock(rawRequirements: string | undefined): { provider: string; profile: string } {
  if (!rawRequirements?.trim()) return { provider: '', profile: '' };
  try {
    const parsed = JSON.parse(rawRequirements) as Record<string, unknown>;
    return {
      provider: typeof parsed.provider_lock === 'string' ? parsed.provider_lock : '',
      profile: typeof parsed.profile_lock === 'string' ? parsed.profile_lock : '',
    };
  } catch {
    return { provider: '', profile: '' };
  }
}

export function planSuggestionDiff(current: WorkPlan | null, suggestion: WorkPlan) {
  const currentByTitle = new Map<string, WorkPackage>();
  for (const phase of current?.phases || []) for (const pkg of phase.packages) currentByTitle.set(pkg.title, pkg);
  const suggestedByTitle = new Map<string, WorkPackage>();
  for (const phase of suggestion.phases) for (const pkg of phase.packages) suggestedByTitle.set(pkg.title, pkg);

  const added: string[] = [];
  const removed: string[] = [];
  const changed: string[] = [];
  for (const [title, pkg] of suggestedByTitle) {
    const old = currentByTitle.get(title);
    if (!old) { added.push(title); continue; }
    const comparable = (value: WorkPackage) => JSON.stringify({
      goal: value.goal,
      priority: value.priority,
      dependencies: value.dependencies,
      role: value.role,
      task_requirements: value.task_requirements,
      agent_allocation: value.agent_allocation,
      acceptance_criteria: value.acceptance_criteria,
      parallel_group: value.parallel_group,
    });
    if (comparable(old) !== comparable(pkg)) changed.push(title);
  }
  for (const title of currentByTitle.keys()) if (!suggestedByTitle.has(title)) removed.push(title);
  return { added: added.sort(), removed: removed.sort(), changed: changed.sort() };
}
