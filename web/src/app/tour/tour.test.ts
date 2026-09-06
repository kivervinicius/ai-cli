import { describe, expect, it } from 'vitest';
import { availableTourSteps, nextTourIndex, previousTourIndex, type TourStep } from './tour';

const steps: TourStep[] = [
  { id: 'projects', target: '[data-tour="projects"]', title: 'Projects', body: 'x' },
  { id: 'workspace', target: '[data-tour="workspace"]', title: 'Workspace', body: 'x' },
  { id: 'missing', target: '[data-tour="missing"]', title: 'Missing', body: 'x' },
];

describe('tour model', () => {
  it('keeps only visible targets', () =>
    expect(
      availableTourSteps(steps, (selector) => selector !== '[data-tour="missing"]'),
    ).toHaveLength(2));
  it('advances within bounds', () => expect(nextTourIndex(0, 3)).toBe(1));
  it('stays at final step', () => expect(nextTourIndex(2, 3)).toBe(2));
  it('moves backward within bounds', () => expect(previousTourIndex(2)).toBe(1));
  it('stays at zero', () => expect(previousTourIndex(0)).toBe(0));
});
