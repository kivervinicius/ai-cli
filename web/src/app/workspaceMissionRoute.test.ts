import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./WorkspaceSurfaceHost.tsx', import.meta.url), 'utf8');

describe('Workspace OS mission surface wiring', () => {
  it('routes Missions to Flow Runs history instead of the PlanBuilder editor', () => {
    expect(source).toContain(
      "import { FlowRunsHistorySurface } from '../features/work/FlowRunsHistorySurface';",
    );
    expect(source).toMatch(/surface\.type === 'missions'[\s\S]*?<FlowRunsHistorySurface/);
    expect(source).not.toMatch(/surface\.type === 'missions'[\s\S]*?<PlanBuilderSurface/);
    expect(source).not.toMatch(/surface\.type === 'missions'[\s\S]*?<MissionsPage projectId=/);
  });

  it('mounts PlanBuilder once via the Composer work surface', () => {
    expect(source).toMatch(/surface\.type === 'work'[\s\S]*?<WorkSurface/);
  });

  it('does not mount MaestroPage on leftover maestro tabs', () => {
    expect(source).not.toContain('MaestroPage');
    expect(source).toContain('maestroControl.legacySurface');
  });
});
