import { Link, useMatchRoute } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard, Server, Target, Hash, Activity, ClipboardCheck, Users, Settings as SettingsIcon,
} from "lucide-react";

type Item = { to: string; label: string; icon: React.ComponentType<{ className?: string }>; comingSoon?: boolean };

const items: Item[] = [
  { to: "/", label: "Overview", icon: LayoutDashboard },
  { to: "/providers", label: "Providers", icon: Server },
  { to: "/targets", label: "Targets", icon: Target },
  { to: "/status-codes", label: "Status Codes", icon: Hash },
  { to: "/live-traffic", label: "Live Traffic", icon: Activity },
  { to: "/audits", label: "Audits", icon: ClipboardCheck },
  { to: "/profiles", label: "Profiles", icon: Users },
  { to: "/settings", label: "Settings", icon: SettingsIcon, comingSoon: true },
];

export function Sidebar() {
  const matchRoute = useMatchRoute();
  return (
    <nav className="flex h-full flex-col gap-0.5 p-2">
      {items.map((item) => {
        const isActive = matchRoute({ to: item.to as never, fuzzy: false });
        const Icon = item.icon;
        return (
          <Link
            key={item.to}
            to={item.to as never}
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
            )}
          >
            <Icon className="h-4 w-4 opacity-70" />
            <span>{item.label}</span>
            {item.comingSoon && (
              <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-muted-foreground">
                soon
              </span>
            )}
          </Link>
        );
      })}
    </nav>
  );
}
