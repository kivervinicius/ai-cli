import React, { useEffect, useRef, useState } from 'react';
import { toast, Toaster } from 'sonner';
import type { RuntimeSession } from '../types';
import { AttentionNotificationCard } from './AttentionNotificationCard';

export interface AttentionNotificationManagerProps {
  runtimes: RuntimeSession[];
  onFocusRuntime: (runtimeId: string) => void;
}

export const AttentionNotificationManager: React.FC<AttentionNotificationManagerProps> = ({
  runtimes,
  onFocusRuntime,
}) => {
  const [dismissedIds, setDismissedIds] = useState<string[]>([]);
  const activeToastIds = useRef<Map<string, string | number>>(new Map());

  // Attention filter
  const attentionRuntimes = runtimes.filter((r) => {
    if (dismissedIds.includes(r.runtime_id)) return false;
    return (
      r.attention_reason === 'QUESTION' ||
      r.attention_reason === 'APPROVAL' ||
      r.state === 'WAITING' ||
      r.state === 'APPROVAL'
    );
  });

  const handleDismiss = (runtimeId: string) => {
    setDismissedIds((prev) => [...prev, runtimeId]);
    const toastId = activeToastIds.current.get(runtimeId);
    if (toastId) {
      toast.dismiss(toastId);
      activeToastIds.current.delete(runtimeId);
    }
  };

  useEffect(() => {
    // Current runtime IDs needing attention
    const currentNeedingAttentionIds = new Set(attentionRuntimes.map((r) => r.runtime_id));

    // Dismiss toasts for runtimes that no longer require attention (e.g. responded or stopped)
    activeToastIds.current.forEach((toastId, runtimeId) => {
      if (!currentNeedingAttentionIds.has(runtimeId)) {
        toast.dismiss(toastId);
        activeToastIds.current.delete(runtimeId);
      }
    });

    // Create or update toast for each attention runtime
    attentionRuntimes.forEach((runtime) => {
      const existingToastId = activeToastIds.current.get(runtime.runtime_id);
      if (!existingToastId) {
        const id = toast.custom(
          () => (
            <AttentionNotificationCard
              runtime={runtime}
              onFocusRuntime={onFocusRuntime}
              onDismiss={handleDismiss}
            />
          ),
          {
            id: `attention-${runtime.runtime_id}`,
            duration: Infinity, // Stay until answered or dismissed
            dismissible: false,
          }
        );
        activeToastIds.current.set(runtime.runtime_id, id);
      }
    });
  }, [attentionRuntimes, onFocusRuntime]);

  return (
    <Toaster
      position="top-right"
      expand
      richColors={false}
      gap={12}
      visibleToasts={4}
      toastOptions={{
        className: 'nx-sonner-toast-wrapper',
        unstyled: true,
      }}
    />
  );
};
