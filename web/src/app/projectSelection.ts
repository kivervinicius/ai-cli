import type { Project } from '../types';
export function resolveProjectSelection(
  projects: Project[],
  currentId?: string | null,
): Project | null {
  return projects.find((project) => project.id === currentId) ?? projects[0] ?? null;
}
