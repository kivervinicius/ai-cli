import { describe, expect, it } from 'vitest';
import { filterCommands, type NexusCommand } from './registry';

const commands: NexusCommand[] = [
  { id: 'open-agents', label: 'Open Agents', group: 'Project', keywords: ['team', 'workers'], run: () => undefined },
  { id: 'open-resources', label: 'Open Resources', group: 'Global', keywords: ['quota', 'provider'], run: () => undefined },
  { id: 'theme', label: 'Change Theme', group: 'Settings', keywords: ['dark', 'light'], run: () => undefined },
];

describe('command registry', () => {
  it('returns all commands for empty query', () => expect(filterCommands(commands, '')).toHaveLength(3));
  it('matches labels', () => expect(filterCommands(commands, 'resources')[0].id).toBe('open-resources'));
  it('matches keywords', () => expect(filterCommands(commands, 'quota')[0].id).toBe('open-resources'));
  it('matches group names', () => expect(filterCommands(commands, 'settings')[0].id).toBe('theme'));
});
