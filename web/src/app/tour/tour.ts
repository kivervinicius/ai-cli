export interface TourStep {
  id: string;
  target: string;
  title: string;
  body: string;
}

export const productTourSteps: TourStep[] = [
  { id: 'workspace', target: '[data-tour="workspace"]', title: 'tour.workspaceTitle', body: 'tour.workspaceBody' },
  { id: 'command', target: '[data-tour="command"]', title: 'tour.commandTitle', body: 'tour.commandBody' },
  { id: 'status', target: '[data-tour="status"]', title: 'tour.statusTitle', body: 'tour.statusBody' },
  { id: 'taskbar', target: '[data-tour="taskbar"]', title: 'tour.taskbarTitle', body: 'tour.taskbarBody' },
  { id: 'projects', target: '[data-tour="projects"]', title: 'tour.projectsTitle', body: 'tour.projectsBody' },
];

export function availableTourSteps(steps: TourStep[], exists: (selector: string) => boolean): TourStep[] {
  return steps.filter((step) => exists(step.target));
}

export const nextTourIndex = (index: number, length: number) => Math.min(Math.max(0, length - 1), index + 1);
export const previousTourIndex = (index: number) => Math.max(0, index - 1);
