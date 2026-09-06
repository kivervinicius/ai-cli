import type { WorkPackage, WorkPlan } from '../types';

type UnknownRecord = Record<string, unknown>;

function asRecord(value: unknown): UnknownRecord {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as UnknownRecord)
    : {};
}

function normalizePackage(value: unknown): WorkPackage {
  const pkg = asRecord(value);
  return {
    ...pkg,
    dependencies: Array.isArray(pkg.dependencies) ? pkg.dependencies : [],
    acceptance_criteria: Array.isArray(pkg.acceptance_criteria) ? pkg.acceptance_criteria : [],
  } as WorkPackage;
}

/** Converts untrusted API JSON into a WorkPlan safe for collection operations. */
export function normalizeWorkPlan(input: unknown): WorkPlan {
  const plan = asRecord(input);
  const phases = Array.isArray(plan.phases) ? plan.phases : [];

  return {
    ...plan,
    phases: phases.map((value) => {
      const phase = asRecord(value);
      return {
        ...phase,
        packages: Array.isArray(phase.packages) ? phase.packages.map(normalizePackage) : [],
      };
    }),
  } as WorkPlan;
}
