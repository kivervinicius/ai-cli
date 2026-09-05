import { describe, expect, it } from 'vitest';
import { composerGateForReadiness } from './composerModel';
import type { ContextReadinessState } from '../../types';

describe('Composer Context Readiness gate', () => {
  const states: ContextReadinessState[] = ['MISSING', 'HYDRATING', 'READY', 'STALE', 'FAILED'];
  it('allows planning/refinement only when READY', () => {
    for (const state of states)
      expect(composerGateForReadiness(state).canCompose).toBe(state === 'READY');
  });
  it('maps non-ready states to explicit actions', () => {
    expect(composerGateForReadiness('MISSING').action).toBe('PREPARE');
    expect(composerGateForReadiness('HYDRATING').action).toBe('WAIT');
    expect(composerGateForReadiness('STALE').action).toBe('REFRESH');
    expect(composerGateForReadiness('FAILED').action).toBe('RETRY');
  });
});
