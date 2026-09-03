// @ts-expect-error Vitest executes in Node; browser tsconfig intentionally omits @types/node.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./modals/MaestroControlModal.tsx', import.meta.url), 'utf8');

describe('Maestro UI honesty', () => {
  it('never fabricates a version or skills when Maestro is unavailable', () => {
    expect(source).not.toContain("|| '0.1.25'");
    expect(source).not.toContain('skill-saas-factory');
    expect(source).not.toContain('skill-security-hooks');
    expect(source).toContain('capabilities?.skills || []');
  });

  it('does not expose ASSIST / autonomous / off as workspace modes', () => {
    expect(source).not.toContain('modeAssist');
    expect(source).not.toContain('modeAutonomous');
    expect(source).not.toContain('modeOff');
    expect(source).not.toContain('ASSIST');
    expect(source).not.toContain('AUTONOMOUS');
    expect(source).not.toContain('ORCHESTRATE');
    expect(source).not.toContain('onOpenMaestroSurface');
  });
});
