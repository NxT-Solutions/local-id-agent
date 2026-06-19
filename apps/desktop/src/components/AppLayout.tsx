import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  Activity,
  BookOpen,
  LayoutDashboard,
  Lock,
  LockKeyhole,
  PlayCircle,
  Settings,
  Shield,
} from "lucide-react";
import { UnlockDialog } from "@/components/admin/UnlockDialog";
import { SystemStatusBar } from "@/components/layout/SystemStatusBar";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAdminLock } from "@/context/AdminLockContext";
import { cn } from "@/lib/utils";

function navLinkClass(isActive: boolean) {
  return cn(
    "flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
    "justify-center md:justify-start",
    isActive
      ? "bg-primary text-primary-foreground shadow-sm"
      : "text-foreground/80 hover:bg-muted hover:text-foreground",
  );
}

const userNavItems = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/about", label: "Diagnostics", icon: Activity },
] as const;

const adminNavItems = [
  { to: "/setup", label: "Setup", icon: BookOpen },
  { to: "/settings", label: "Settings", icon: Settings },
  { to: "/demo", label: "Demo", icon: PlayCircle },
] as const;

export function AppLayout() {
  const {
    unlocked,
    lock,
    unlockDialogOpen,
    setUnlockDialogOpen,
    requestUnlock,
  } = useAdminLock();
  const location = useLocation();
  const navigate = useNavigate();
  const navItems = unlocked
    ? [...userNavItems, ...adminNavItems]
    : [...userNavItems];

  return (
    <TooltipProvider delayDuration={400}>
      <div className="flex h-screen overflow-hidden bg-background">
        <aside className="flex h-full w-14 shrink-0 flex-col border-r bg-card md:w-48 lg:w-56">
          <div className="sticky top-0 z-10 shrink-0 border-b bg-card px-2 py-4 md:px-4 md:py-5">
            <div className="flex items-center justify-center gap-2.5 md:justify-start">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Shield className="h-4 w-4" />
              </div>
              <div className="hidden min-w-0 md:block">
                <p className="truncate text-sm font-semibold tracking-tight">
                  LocalID Agent
                </p>
                <p className="truncate text-[0.6875rem] text-muted-foreground">
                  Identity bridge
                </p>
              </div>
            </div>
          </div>

          <nav className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto p-2 md:p-3">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={"end" in item ? item.end : undefined}
                aria-label={item.label}
                title={item.label}
                className={({ isActive }) => navLinkClass(isActive)}
              >
                <item.icon className="h-4 w-4 shrink-0" />
                <span className="hidden truncate md:inline">{item.label}</span>
              </NavLink>
            ))}
          </nav>

          <div className="shrink-0 space-y-2 border-t bg-card p-2 md:px-4 md:py-3">
            <div className="hidden flex-col gap-2 md:flex">
              {unlocked ? (
                <>
                  <Badge variant="secondary" className="w-fit gap-1">
                    <Shield className="h-3 w-3" />
                    Admin
                  </Badge>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => void lock()}
                  >
                    <Lock className="h-4 w-4" />
                    Lock admin
                  </Button>
                </>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full justify-start"
                  onClick={requestUnlock}
                >
                  <LockKeyhole className="h-4 w-4" />
                  Unlock admin
                </Button>
              )}
            </div>
            <ThemeSwitcher />
            <p className="hidden text-[0.6875rem] leading-relaxed text-muted-foreground md:block">
              Signs challenges locally. Does not issue sessions or tokens.
            </p>
          </div>
        </aside>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden">
          <div className="sticky top-0 z-10 shrink-0">
            <SystemStatusBar />
          </div>
          <main className="surface-muted min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
            <div className="mx-auto max-w-5xl min-w-0 px-4 py-6 sm:px-6 sm:py-8">
              <Outlet />
            </div>
          </main>
        </div>
      </div>

      <UnlockDialog
        open={unlockDialogOpen}
        onOpenChange={setUnlockDialogOpen}
        onUnlocked={() => {
          const from =
            typeof location.state === "object" &&
            location.state !== null &&
            "from" in location.state &&
            typeof location.state.from === "string"
              ? location.state.from
              : null;
          if (from) {
            navigate(from, { replace: true, state: null });
          }
        }}
      />
    </TooltipProvider>
  );
}
