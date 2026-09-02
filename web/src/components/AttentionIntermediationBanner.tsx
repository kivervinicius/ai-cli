import React from 'react';
import type { RuntimeSession } from '../types';
import { AttentionNotificationManager } from './AttentionNotificationManager';

/**
 * AttentionIntermediationBanner is refactored to use Sonner with Untitled UI styling.
 * Preserves exact props signature for complete backwards compatibility with all callers.
 */
export const AttentionIntermediationBanner: React.FC<{
  runtimes: RuntimeSession[];
  onFocusRuntime: (runtimeId: string) => void;
}> = ({ runtimes, onFocusRuntime }) => {
  return (
    <AttentionNotificationManager
      runtimes={runtimes}
      onFocusRuntime={onFocusRuntime}
    />
  );
};
