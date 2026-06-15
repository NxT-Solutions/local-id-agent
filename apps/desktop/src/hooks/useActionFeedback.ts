import { useCallback, useEffect, useRef, useState } from "react";

export type ActionFeedbackVariant = "success" | "error";

export interface ActionFeedbackState {
  message: string;
  variant: ActionFeedbackVariant;
}

const DEFAULT_DURATION_MS = 2000;

export function useActionFeedback(durationMs = DEFAULT_DURATION_MS) {
  const [feedback, setFeedback] = useState<ActionFeedbackState | null>(null);
  const hideTimerRef = useRef<number | null>(null);

  const clearTimers = useCallback(() => {
    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
  }, []);

  useEffect(() => clearTimers, [clearTimers]);

  const dismiss = useCallback(() => {
    clearTimers();
    setFeedback(null);
  }, [clearTimers]);

  const show = useCallback(
    (message: string, variant: ActionFeedbackVariant = "success") => {
      clearTimers();
      setFeedback({ message, variant });

      hideTimerRef.current = window.setTimeout(() => {
        setFeedback(null);
      }, durationMs);
    },
    [clearTimers, durationMs],
  );

  const showSuccess = useCallback(
    (message: string) => show(message, "success"),
    [show],
  );

  const showError = useCallback(
    (message: string) => show(message, "error"),
    [show],
  );

  const onOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        dismiss();
      }
    },
    [dismiss],
  );

  return { feedback, show, showSuccess, showError, onOpenChange };
}
