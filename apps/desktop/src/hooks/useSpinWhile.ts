import { useEffect, useRef, useState } from "react";

/**
 * Keeps `spinning` true while `active` is true, and for at least `minMs` once triggered.
 * Prevents refresh icons from flickering when requests finish quickly.
 */
export function useSpinWhile(active: boolean, minMs = 600): boolean {
  const [spinning, setSpinning] = useState(false);
  const activeSinceRef = useRef<number | null>(null);
  const offTimerRef = useRef<number | null>(null);

  useEffect(() => {
    if (active) {
      if (activeSinceRef.current === null) {
        activeSinceRef.current = Date.now();
        setSpinning(true);
      }
      if (offTimerRef.current !== null) {
        window.clearTimeout(offTimerRef.current);
        offTimerRef.current = null;
      }
      return;
    }

    if (activeSinceRef.current === null) {
      return;
    }

    const elapsed = Date.now() - activeSinceRef.current;
    const remaining = Math.max(0, minMs - elapsed);

    offTimerRef.current = window.setTimeout(() => {
      activeSinceRef.current = null;
      offTimerRef.current = null;
      setSpinning(false);
    }, remaining);

    return () => {
      if (offTimerRef.current !== null) {
        window.clearTimeout(offTimerRef.current);
        offTimerRef.current = null;
      }
    };
  }, [active, minMs]);

  return spinning;
}
