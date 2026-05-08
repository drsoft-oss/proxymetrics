import { HowToConnectDialog } from "./HowToConnectDialog";
import { ThemeToggle } from "./ThemeToggle";

export function Topbar() {
  return (
    <header className="col-span-3 flex h-12 items-center border-b border-border bg-card px-4">
      <div className="text-sm font-semibold">ProxyMetrics</div>
      <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
        <HowToConnectDialog />
        <span>UTC</span>
        <ThemeToggle />
      </div>
    </header>
  );
}
