import { useState } from "react";
import { Monitor, Moon, Sun, type LucideIcon } from "lucide-react";
import { HoverTooltip } from "@/components/ui/hover-tooltip";
import { useTheme, type ThemeMode } from "@/hooks/useTheme";
import { cn } from "@/lib/utils";

const themeByMode: Record<
  ThemeMode,
  { label: string; icon: LucideIcon; iconClass: string }
> = {
  light: {
    label: "Light",
    icon: Sun,
    iconClass: "text-amber-500",
  },
  dark: {
    label: "Dark",
    icon: Moon,
    iconClass: "text-indigo-400",
  },
  system: {
    label: "System",
    icon: Monitor,
    iconClass: "text-foreground",
  },
};

export function ThemeSwitcher() {
  const { mode, cycleMode } = useTheme();
  const [spinning, setSpinning] = useState(false);
  const { label, icon: Icon, iconClass } = themeByMode[mode];

  function handleClick() {
    setSpinning(true);
    cycleMode();
    window.setTimeout(() => setSpinning(false), 450);
  }

  return (
    <div className="flex justify-center">
      <HoverTooltip label={`${label} — click for next theme`} side="right">
        <button
          type="button"
          aria-label={`Theme: ${label}. Click to switch.`}
          onClick={handleClick}
          className={cn(
            "flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg border bg-muted/40 text-muted-foreground transition-colors duration-200",
            "hover:bg-muted hover:text-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          )}
        >
        <Icon
          className={cn(
            "h-4 w-4 shrink-0",
            iconClass,
            spinning && "animate-spin-once",
          )}
        />
      </button>
    </HoverTooltip>
    </div>
  );
}
