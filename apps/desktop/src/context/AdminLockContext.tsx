import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  getAdminLockStatus,
  lockAdmin,
  setupAdminPasscode,
  unlockAdmin,
  type AdminLockStatus,
} from "@/lib/tauri";

const STATUS_POLL_MS = 30_000;

interface AdminLockContextValue {
  status: AdminLockStatus | null;
  loading: boolean;
  unlocked: boolean;
  setupRequired: boolean;
  unlockDialogOpen: boolean;
  setUnlockDialogOpen: (open: boolean) => void;
  requestUnlock: () => void;
  refreshStatus: () => Promise<void>;
  setup: (passcode: string) => Promise<void>;
  unlock: (passcode: string) => Promise<void>;
  lock: () => Promise<void>;
}

const AdminLockContext = createContext<AdminLockContextValue | null>(null);

export function AdminLockProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AdminLockStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [unlockDialogOpen, setUnlockDialogOpen] = useState(false);

  const refreshStatus = useCallback(async () => {
    try {
      const next = await getAdminLockStatus();
      setStatus(next);
    } catch {
      setStatus((current) => current ?? {
        configured: false,
        unlocked: false,
        setupRequired: true,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshStatus();
    const timer = window.setInterval(() => {
      void refreshStatus();
    }, STATUS_POLL_MS);
    return () => window.clearInterval(timer);
  }, [refreshStatus]);

  const requestUnlock = useCallback(() => {
    setUnlockDialogOpen(true);
  }, []);

  const setup = useCallback(
    async (passcode: string) => {
      await setupAdminPasscode(passcode);
      await refreshStatus();
    },
    [refreshStatus],
  );

  const unlock = useCallback(
    async (passcode: string) => {
      await unlockAdmin(passcode);
      await refreshStatus();
    },
    [refreshStatus],
  );

  const lock = useCallback(async () => {
    await lockAdmin();
    await refreshStatus();
  }, [refreshStatus]);

  const value = useMemo<AdminLockContextValue>(
    () => ({
      status,
      loading,
      unlocked: status?.unlocked ?? false,
      setupRequired: status?.setupRequired ?? false,
      unlockDialogOpen,
      setUnlockDialogOpen,
      requestUnlock,
      refreshStatus,
      setup,
      unlock,
      lock,
    }),
    [
      loading,
      lock,
      refreshStatus,
      requestUnlock,
      setup,
      status,
      unlock,
      unlockDialogOpen,
    ],
  );

  return (
    <AdminLockContext.Provider value={value}>{children}</AdminLockContext.Provider>
  );
}

export function useAdminLock() {
  const context = useContext(AdminLockContext);
  if (!context) {
    throw new Error("useAdminLock must be used within AdminLockProvider");
  }
  return context;
}
