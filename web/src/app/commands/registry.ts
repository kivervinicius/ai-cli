export interface NexusCommand {
  id: string;
  label: string;
  group: string;
  description?: string;
  shortcut?: string;
  keywords?: string[];
  run: () => void;
}

export function filterCommands(commands: NexusCommand[], query: string): NexusCommand[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return commands;
  return commands
    .map((command) => {
      const label = command.label.toLowerCase();
      const haystack = [command.label, command.group, command.description, ...(command.keywords ?? [])].filter(Boolean).join(' ').toLowerCase();
      const score = label === needle ? 100 : label.startsWith(needle) ? 80 : label.includes(needle) ? 60 : haystack.includes(needle) ? 30 : 0;
      return { command, score };
    })
    .filter((entry) => entry.score > 0)
    .sort((a, b) => b.score - a.score || a.command.label.localeCompare(b.command.label))
    .map((entry) => entry.command);
}
