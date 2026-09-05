import { describe, expect, it } from 'vitest';
import type { WorkPlan } from '../../types';
import {
  mergePackages,
  normalizeWorkPlan,
  planSuggestionDiff,
  setProviderLock,
  splitPackage,
} from './planBuilderModel';

const plan = (): WorkPlan => ({
  id: 'plan-1',
  project_id: 'p',
  title: 'Plan',
  description: '',
  status: 'DRAFT',
  current_revision: 1,
  created_at: '',
  updated_at: '',
  phases: [
    {
      id: 'phase-1',
      title: 'Phase',
      order: 1,
      packages: [
        {
          id: 'a',
          title: 'Foundation',
          goal: 'base',
          priority: 'HIGH',
          status: 'READY',
          dependencies: [],
          role: 'implementer',
          acceptance_criteria: ['schema', 'tests'],
        },
        {
          id: 'b',
          title: 'API',
          goal: 'api',
          priority: 'HIGH',
          status: 'READY',
          dependencies: ['a'],
          role: 'implementer',
          acceptance_criteria: ['endpoint'],
        },
      ],
    },
  ],
});

describe('WorkPlan structural editing', () => {
  it('normalizes nullable phase/package collections from API payloads', () => {
    const malformed = {
      ...plan(),
      phases: [
        {
          ...plan().phases[0],
          packages: null as unknown as WorkPlan['phases'][number]['packages'],
        },
      ],
    };
    const normalized = normalizeWorkPlan(malformed);
    expect(normalized.phases).toHaveLength(1);
    expect(normalized.phases[0].packages).toEqual([]);
  });

  it('splits a package without allowing downstream work to bypass the second half', () => {
    const next = splitPackage(plan(), 'phase-1', 'a', 'a2');
    const packages = next.phases[0].packages;
    expect(packages.map((pkg) => pkg.id)).toEqual(['a', 'a2', 'b']);
    expect(packages[0].acceptance_criteria).toEqual(['schema']);
    expect(packages[1].acceptance_criteria).toEqual(['tests']);
    expect(packages[1].dependencies).toEqual(['a']);
    expect(packages[2].dependencies).toEqual(['a2']);
  });

  it('merges packages and rewrites downstream dependencies to the survivor', () => {
    const source = plan();
    source.phases[0].packages.push({
      id: 'c',
      title: 'Review',
      goal: 'review',
      priority: 'NORMAL',
      status: 'READY',
      dependencies: ['b'],
      role: 'reviewer',
      acceptance_criteria: ['approved'],
    });
    const next = mergePackages(source, 'phase-1', 'a', 'b');
    const packages = next.phases[0].packages;
    expect(packages.map((pkg) => pkg.id)).toEqual(['a', 'c']);
    expect(packages[0].acceptance_criteria).toEqual(['schema', 'tests', 'endpoint']);
    expect(packages[1].dependencies).toEqual(['a']);
  });
});

describe('Provider lock', () => {
  it('writes and clears provider/profile hard locks without destroying other requirements', () => {
    const initial = JSON.stringify({ required_capabilities: ['headless'], estimated_tokens: 1000 });
    const locked = setProviderLock(initial, 'claude', 'review');
    expect(JSON.parse(locked)).toMatchObject({
      required_capabilities: ['headless'],
      estimated_tokens: 1000,
      provider_lock: 'claude',
      profile_lock: 'review',
    });
    expect(JSON.parse(setProviderLock(locked, '', ''))).toEqual({
      required_capabilities: ['headless'],
      estimated_tokens: 1000,
    });
  });
});

describe('AI suggestion diff', () => {
  it('compares a generated plan without mutating the approved plan', () => {
    const current = plan();
    const suggestion = plan();
    suggestion.id = 'suggested';
    suggestion.phases[0].packages[1] = {
      ...suggestion.phases[0].packages[1],
      goal: 'new api goal',
    };
    suggestion.phases[0].packages.push({
      id: 'new',
      title: 'Security',
      goal: 'audit',
      priority: 'HIGH',
      status: 'READY',
      dependencies: [],
      role: 'reviewer',
      acceptance_criteria: ['clean'],
    });
    const diff = planSuggestionDiff(current, suggestion);
    expect(diff.added).toEqual(['Security']);
    expect(diff.changed).toEqual(['API']);
    expect(current.phases[0].packages).toHaveLength(2);
  });
});
