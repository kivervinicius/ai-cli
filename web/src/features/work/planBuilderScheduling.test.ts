// @ts-expect-error Vitest executes in Node; browser tsconfig intentionally omits @types/node.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./PlanBuilderSurface.tsx', import.meta.url), 'utf8');
const autonomySource = readFileSync(new URL('./MissionAutonomyCard.tsx', import.meta.url), 'utf8');

describe('Mission scheduling wiring', () => {
  it('allows AFTER_RUN scheduling while the dependency Mission is still running', () => {
    expect(source).toContain('onClick={handleAfterRun}');
    expect(source).toContain("t('planBuilder.afterMission')");
    expect(source).not.toContain("disabled={activeRun.state !== 'COMPLETED_VERIFIED'}");
  });
});

describe('Mission autonomy approval wiring', () => {
  it('passes the approved autonomy contract to immediate and scheduled runs without silently forcing the first Agent', () => {
    expect(source).toContain('contract: autonomyContract');
    expect(source).not.toContain('agentId: agents[0]?.id');
    expect(source).toContain('Approve Contract & Run Mission');
    expect(autonomySource).toContain('allow_external_network');
    expect(autonomySource).toContain('allow_secret_access');
    expect(autonomySource).toContain('allow_paid_services');
  });
});
