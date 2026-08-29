import type { Agent, RuntimeSession } from '../types';

export function agentForRuntime(runtime: RuntimeSession, agents: Agent[]): Agent | undefined {
  if (!runtime.agent_id) return undefined;
  return agents.find((agent) => agent.id === runtime.agent_id);
}

export function terminalSurfaceIDForRuntime(runtime: RuntimeSession): string | undefined {
  return runtime.agent_id ? `agent:${runtime.agent_id}:terminal` : undefined;
}
