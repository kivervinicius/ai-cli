export type FlowRunState =
  'QUEUED' | 'READY' | 'RUNNING' | 'VERIFYING' | 'COMPLETED' | 'BLOCKED' | 'FAILED' | 'CANCELED';

const failedStates = new Set([
  'FAILED',
  'FAILED_NO_PROGRESS',
  'FAILED_BUDGET_EXCEEDED',
  'FAILED_VERIFICATION',
]);

export function flowRunStateFromMission(state: string): FlowRunState {
  switch (String(state || '').toUpperCase()) {
    case 'PENDING':
      return 'QUEUED';
    case 'READY':
      return 'READY';
    case 'ALLOCATING':
    case 'COMPILING':
    case 'EXECUTING':
      return 'RUNNING';
    case 'TESTING':
    case 'REVIEWING':
    case 'REMEDIATING':
      return 'VERIFYING';
    case 'VERIFIED':
    case 'COMPLETED_VERIFIED':
      return 'COMPLETED';
    case 'BLOCKED_NEEDS_USER':
    case 'PAUSED':
    case 'ESCALATED':
      return 'BLOCKED';
    case 'CANCELED_BY_USER':
    case 'CANCELED':
    case 'CANCELLED':
      return 'CANCELED';
    default:
      return failedStates.has(String(state || '').toUpperCase()) ? 'FAILED' : 'BLOCKED';
  }
}

export function packageRunState(state: string): FlowRunState {
  return flowRunStateFromMission(state);
}
