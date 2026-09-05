export type ShortcutScope = 'global' | 'workspace' | 'terminal' | 'chat' | 'flow' | 'dialog';

export interface ShortcutDefinition {
  id: string;
  key: string; // e.g. 'k', 'p', 'Enter', 'Escape'
  ctrlOrMeta?: boolean;
  shift?: boolean;
  alt?: boolean;
  scope: ShortcutScope;
  description: string;
  action: (event: KeyboardEvent) => void;
  /** If true, default browser action is prevented. */
  preventDefault?: boolean;
}

export class KeyboardShortcutRegistry {
  private static instance: KeyboardShortcutRegistry;
  private shortcuts: Map<string, ShortcutDefinition> = new Map();
  private activeScope: ShortcutScope = 'global';
  private listenerAttached = false;

  private constructor() {
    this.handleKeyDown = this.handleKeyDown.bind(this);
  }

  public static getInstance(): KeyboardShortcutRegistry {
    if (!KeyboardShortcutRegistry.instance) {
      KeyboardShortcutRegistry.instance = new KeyboardShortcutRegistry();
    }
    return KeyboardShortcutRegistry.instance;
  }

  public setScope(scope: ShortcutScope): void {
    this.activeScope = scope;
  }

  public getScope(): ShortcutScope {
    return this.activeScope;
  }

  public register(shortcut: ShortcutDefinition): () => void {
    this.shortcuts.set(shortcut.id, shortcut);
    this.ensureListener();
    return () => {
      this.shortcuts.delete(shortcut.id);
    };
  }

  public clear(): void {
    this.shortcuts.clear();
  }

  private ensureListener(): void {
    if (this.listenerAttached || typeof window === 'undefined') return;
    window.addEventListener('keydown', this.handleKeyDown, { capture: true });
    this.listenerAttached = true;
  }

  public handleKeyDown(event: KeyboardEvent): void {
    // Check if user is typing IME composition
    if (event.isComposing || event.keyCode === 229) {
      return;
    }

    const target = event.target as HTMLElement | null;
    const isInputOrTextarea = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable;
    const isTerminalTarget = Boolean(target?.closest('.xterm') || target?.classList.contains('xterm-helper-textarea'));

    // If focus is inside terminal, NEVER hijack standard shell / terminal keys
    if (isTerminalTarget && this.activeScope !== 'dialog') {
      // Allow only Global palette triggers like Ctrl+Shift+P or Ctrl+K if desired,
      // but DO NOT hijack Ctrl+C, Ctrl+V, Ctrl+Shift+C, Ctrl+Shift+V, Enter, Arrows, etc.
      const isGlobalCmd = (event.ctrlKey || event.metaKey) && (event.key.toLowerCase() === 'k' || (event.shiftKey && event.key.toLowerCase() === 'p'));
      if (!isGlobalCmd) {
        return;
      }
    }

    // Match candidate shortcuts
    for (const shortcut of this.shortcuts.values()) {
      // Scope validation: 'global' applies everywhere unless in dialog where dialog has priority
      if (shortcut.scope !== 'global' && shortcut.scope !== this.activeScope) {
        continue;
      }

      const matchKey = event.key.toLowerCase() === shortcut.key.toLowerCase();
      const matchCtrl = Boolean(shortcut.ctrlOrMeta) === (event.ctrlKey || event.metaKey);
      const matchShift = Boolean(shortcut.shift) === event.shiftKey;
      const matchAlt = Boolean(shortcut.alt) === event.altKey;

      if (matchKey && matchCtrl && matchShift && matchAlt) {
        // If typing in input, don't intercept plain navigation keys without modifier
        if (isInputOrTextarea && !shortcut.ctrlOrMeta && !shortcut.alt && shortcut.key !== 'Escape') {
          continue;
        }

        if (shortcut.preventDefault) {
          event.preventDefault();
        }
        shortcut.action(event);
        break;
      }
    }
  }
}
