export type ThemeMode = "light" | "dark" | "system";

export const THEME_STORAGE_KEY = "localid-desktop-theme";

function isThemeMode(value: string | null): value is ThemeMode {
  return value === "light" || value === "dark" || value === "system";
}

export function getSystemTheme(): "light" | "dark" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

export function getStoredTheme(): ThemeMode {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (isThemeMode(stored)) {
      return stored;
    }
  } catch {
    // localStorage may be unavailable in some embedded contexts
  }
  return "system";
}

export function resolveTheme(mode: ThemeMode): "light" | "dark" {
  return mode === "system" ? getSystemTheme() : mode;
}

export function applyTheme(mode: ThemeMode): void {
  const resolved = resolveTheme(mode);
  document.documentElement.classList.toggle("dark", resolved === "dark");
}

export function setStoredTheme(mode: ThemeMode): void {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, mode);
  } catch {
    // ignore persistence failures
  }
  applyTheme(mode);
}

export function initTheme(): ThemeMode {
  const mode = getStoredTheme();
  applyTheme(mode);
  return mode;
}

export const THEME_CYCLE_ORDER: ThemeMode[] = ["light", "dark", "system"];

export function nextThemeMode(mode: ThemeMode): ThemeMode {
  const index = THEME_CYCLE_ORDER.indexOf(mode);
  return THEME_CYCLE_ORDER[(index + 1) % THEME_CYCLE_ORDER.length];
}
