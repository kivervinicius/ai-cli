import { describe, expect, it } from 'vitest';
import {
  shouldMarkUnread,
  statusSuffixFromRuntime,
  surfaceTitleFromAgent,
} from './surfaceAttention';
import type { RuntimeSession } from '../types';

const runtime = (patch: Partial<RuntimeSession>): RuntimeSession =>
  ({
    runtime_id: 'rt-1',
    state: 'RUNNING',
    ...patch,
  }) as RuntimeSession;

describe('surfaceAttention', () => {
  it('keeps the agent name and appends a short status suffix', () => {
    const titled = surfaceTitleFromAgent(
      'Backend',
      runtime({ attention_reason: 'QUESTION', attention_context: 'Continue?', prompt_kind: 'yn', attention_kind: 'needs_user' })
    );
    expect(titled.title).toBe('Backend · pergunta');
    expect(titled.statusSuffix).toBe('pergunta');
    expect(titled.hasAttention).toBe(true);
  });

  it('does not replace the name with dynamic_title', () => {
    const titled = surfaceTitleFromAgent(
      'Frontend',
      runtime({
        attention_reason: 'QUESTION',
        attention_context: 'Apply changes?',
        prompt_kind: 'yn',
        attention_kind: 'needs_user',
        dynamic_title: 'ai-chat · Apply changes?',
      })
    );
    expect(titled.title.startsWith('Frontend')).toBe(true);
    expect(titled.title).not.toContain('ai-chat');
  });

  it('maps completion and error suffixes', () => {
    expect(statusSuffixFromRuntime(runtime({ attention_reason: 'TASK_COMPLETED' }))).toBe('concluído');
    expect(statusSuffixFromRuntime(runtime({ attention_reason: 'ERROR' }))).toBe('erro');
  });

  it('marks unread only on fingerprint transition when unfocused', () => {
    expect(
      shouldMarkUnread({
        previousFingerprint: 'old',
        nextFingerprint: 'new',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'needs_user',
      })
    ).toBe(true);
    expect(
      shouldMarkUnread({
        previousFingerprint: 'same',
        nextFingerprint: 'same',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'needs_user',
      })
    ).toBe(false);
    expect(
      shouldMarkUnread({
        previousFingerprint: 'old',
        nextFingerprint: 'new',
        hasAttention: true,
        surfaceFocused: true,
        attentionKind: 'needs_user',
      })
    ).toBe(false);
    expect(
      shouldMarkUnread({
        previousFingerprint: '',
        nextFingerprint: 'work',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'working',
      })
    ).toBe(false);
  });
});
