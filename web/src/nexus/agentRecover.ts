import { nexus } from './api';
import {
  isRecoverAlreadyAlive,
  isRequiredResourceSelection,
  shouldFallbackRecoverToStart,
} from './agentTerminalModel';
import type { RuntimeSession } from '../types';

export class RequiredResourceError extends Error {
  readonly code = 'REQUIRED_RESOURCE_SELECTION' as const;

  constructor(message: string) {
    super(message);
    this.name = 'RequiredResourceError';
  }
}

export function isRequiredResourceError(err: unknown): err is RequiredResourceError {
  return err instanceof RequiredResourceError ||
    (err instanceof Error && isRequiredResourceSelection(err.message));
}

/**
 * Recover the agent runtime, falling back to Start when Recover refuses.
 * Already-alive (legacy 409) is soft-success; preferred path is 200 + runtime.
 */
export async function recoverOrStartAgent(agentId: string): Promise<{ runtime?: RuntimeSession }> {
  try {
    return await nexus.recoverAgent(agentId);
  } catch (recoverErr) {
    const recoverMsg = recoverErr instanceof Error ? recoverErr.message : String(recoverErr);
    if (isRequiredResourceSelection(recoverMsg)) {
      throw new RequiredResourceError(recoverMsg);
    }
    // Legacy servers may still 409 already-alive without a body.
    if (isRecoverAlreadyAlive(recoverMsg)) {
      return { runtime: undefined };
    }
    if (!shouldFallbackRecoverToStart(recoverMsg)) {
      throw recoverErr instanceof Error ? recoverErr : new Error(recoverMsg);
    }
    try {
      return await nexus.startAgent(agentId);
    } catch (startErr) {
      const startMsg = startErr instanceof Error ? startErr.message : String(startErr);
      if (isRequiredResourceSelection(startMsg)) {
        throw new RequiredResourceError(startMsg);
      }
      throw new Error(`${recoverMsg} · Start: ${startMsg}`);
    }
  }
}
