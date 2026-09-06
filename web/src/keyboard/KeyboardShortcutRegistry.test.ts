import { describe, expect, it, beforeEach } from 'vitest';
import { KeyboardShortcutRegistry } from './KeyboardShortcutRegistry';

class MockKeyboardEvent {
  type: string;
  key: string;
  ctrlKey: boolean;
  metaKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  isComposing: boolean;
  keyCode: number;
  target: any;
  defaultPrevented = false;

  constructor(type: string, init: any = {}) {
    this.type = type;
    this.key = init.key || '';
    this.ctrlKey = Boolean(init.ctrlKey);
    this.metaKey = Boolean(init.metaKey);
    this.shiftKey = Boolean(init.shiftKey);
    this.altKey = Boolean(init.altKey);
    this.isComposing = Boolean(init.isComposing);
    this.keyCode = init.keyCode || 0;
    this.target = init.target || null;
  }

  preventDefault() {
    this.defaultPrevented = true;
  }
}

(globalThis as any).KeyboardEvent = MockKeyboardEvent;

describe('KeyboardShortcutRegistry', () => {
  let registry: KeyboardShortcutRegistry;

  beforeEach(() => {
    registry = KeyboardShortcutRegistry.getInstance();
    registry.clear();
    registry.setScope('global');
  });

  it('triggers registered global shortcut', () => {
    let triggered = false;
    registry.register({
      id: 'test-k',
      key: 'k',
      ctrlOrMeta: true,
      scope: 'global',
      description: 'Command Palette',
      action: () => {
        triggered = true;
      },
    });

    const event = new (globalThis as any).KeyboardEvent('keydown', { key: 'k', ctrlKey: true });
    registry.handleKeyDown(event);
    expect(triggered).toBe(true);
  });

  it('ignores shortcuts during IME composition', () => {
    let triggered = false;
    registry.register({
      id: 'test-enter',
      key: 'Enter',
      scope: 'chat',
      description: 'Send',
      action: () => {
        triggered = true;
      },
    });

    const event = new (globalThis as any).KeyboardEvent('keydown', {
      key: 'Enter',
      isComposing: true,
    });
    registry.handleKeyDown(event);
    expect(triggered).toBe(false);
  });

  it('respects scopes (chat vs global)', () => {
    let chatTriggered = false;
    registry.register({
      id: 'chat-enter',
      key: 'Enter',
      scope: 'chat',
      description: 'Send Chat',
      action: () => {
        chatTriggered = true;
      },
    });

    registry.setScope('global');
    registry.handleKeyDown(new (globalThis as any).KeyboardEvent('keydown', { key: 'Enter' }));
    expect(chatTriggered).toBe(false);

    registry.setScope('chat');
    registry.handleKeyDown(new (globalThis as any).KeyboardEvent('keydown', { key: 'Enter' }));
    expect(chatTriggered).toBe(true);
  });
});
