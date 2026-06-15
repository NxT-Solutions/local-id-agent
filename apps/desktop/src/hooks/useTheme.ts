import { useCallback, useEffect, useState } from "react";
import {
  applyTheme,
  getStoredTheme,
  nextThemeMode,
  resolveTheme,
  setStoredTheme,
  type ThemeMode,
} from "@/lib/theme";

export function useTheme() {
  const [mode, setModeState] = useState<ThemeMode>(() => getStoredTheme());

  useEffect(() => {
    applyTheme(mode);
  }, [mode]);

  useEffect(() => {
    if (mode !== "system") {
      return;
    }

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyTheme("system");

    mediaQuery.addEventListener("change", onChange);
    return () => mediaQuery.removeEventListener("change", onChange);
  }, [mode]);

  const setMode = useCallback((next: ThemeMode) => {
    setStoredTheme(next);
    setModeState(next);
  }, []);

  const cycleMode = useCallback(() => {
    setModeState((current) => {
      const next = nextThemeMode(current);
      setStoredTheme(next);
      return next;
    });
  }, []);

  return {
    mode,
    setMode,
    cycleMode,
    resolved: resolveTheme(mode),
  };
}

export type { ThemeMode };
