import { describe, expect, it } from 'vitest';
import { resolveProjectSelection } from './projectSelection';
import type { Project } from '../types';
const project = (id: string): Project => ({ id, name: id, slug: id, canonical_path: `/tmp/${id}`, repo_remote: '', repo_url: '', default_branch: 'main', maestro_mode: 'ASSIST', resource_policy: 'BALANCED', default_isolation: 'project', settings: '', created_at: '', updated_at: '' });
describe('project selection', () => {
  it('keeps current project when it still exists', () => expect(resolveProjectSelection([project('a'), project('b')], 'b')?.id).toBe('b'));
  it('falls back to first project', () => expect(resolveProjectSelection([project('a')], 'missing')?.id).toBe('a'));
  it('returns null for empty list', () => expect(resolveProjectSelection([], 'a')).toBeNull());
});
