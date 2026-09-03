import React from 'react';
import type { RuntimeSession } from '../types';
import { AttentionNotificationManager } from './AttentionNotificationManager';

/**
 * AttentionIntermediationBanner hosts Sonner toasts for honest needs_user waits.
 */
export const AttentionIntermediationBanner: React.FC<{
  runtimes: RuntimeSession[];
  focusedProjectId?: string;
  onFocusRuntime: (runtimeId: string) => void;
}> = ({ runtimes, focusedProjectId, onFocusRuntime }) => {
  return (
    <AttentionNotificationManager
      runtimes={runtimes}
      focusedProjectId={focusedProjectId}
      onFocusRuntime={onFocusRuntime}
    />
  );
};
