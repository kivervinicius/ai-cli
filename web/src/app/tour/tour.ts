export interface TourStep { id: string; target: string; title: string; body: string; }
export const productTourSteps: TourStep[] = [
  { id: 'projects', target: '[data-tour="projects"]', title: 'Projects are your desktops', body: 'Each Project owns its Agents, layout, Maestro context, Missions and working surfaces.' },
  { id: 'project-nav', target: '[data-tour="project-nav"]', title: 'Project navigation', body: 'Overview, Work, Plan, Agents and operational areas open inside the same Project context.' },
  { id: 'workspace', target: '[data-tour="workspace"]', title: 'Workspace OS', body: 'Tabs and terminals can be split, moved, maximized or popped out. Closing a surface does not stop its Agent.' },
  { id: 'taskbar', target: '[data-tour="taskbar"]', title: 'Workspace taskbar', body: 'Jump instantly between open surfaces, similar to switching applications in an operating system.' },
  { id: 'command', target: '[data-tour="command"]', title: 'Command Palette', body: 'Press Ctrl/Cmd+K to open Agents, Projects, Resources, Maestro, settings and workspace actions.' },
  { id: 'status', target: '[data-tour="status"]', title: 'Resource status', body: 'The top bar keeps Agent, provider and Maestro health visible while you work.' },
];
export function availableTourSteps(steps: TourStep[], exists: (selector: string) => boolean): TourStep[] { return steps.filter((step) => exists(step.target)); }
export const nextTourIndex = (index: number, length: number) => Math.min(Math.max(0, length - 1), index + 1);
export const previousTourIndex = (index: number) => Math.max(0, index - 1);
