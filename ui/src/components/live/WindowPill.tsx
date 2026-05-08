import { cn } from "@/lib/utils";

export type LiveWindow = 30 | 60 | 300;

const OPTIONS: { value: LiveWindow; label: string }[] = [
  { value: 30, label: "30s" },
  { value: 60, label: "1m" },
  { value: 300, label: "5m" },
];

export function WindowPill({
  value,
  onChange,
  paused,
  onTogglePause,
}: {
  value: LiveWindow;
  onChange: (v: LiveWindow) => void;
  paused: boolean;
  onTogglePause: () => void;
}) {
  return (
    <div className="inline-flex items-center gap-1 rounded-md border border-border bg-background p-0.5 text-xs">
      {OPTIONS.map((o) => (
        <button
          key={o.value}
          onClick={() => onChange(o.value)}
          className={cn(
            "rounded px-2 py-1",
            value === o.value ? "bg-accent text-accent-foreground" : "hover:bg-accent/40 text-muted-foreground"
          )}
        >
          {o.label}
        </button>
      ))}
      <button
        onClick={onTogglePause}
        className={cn(
          "ml-1 rounded px-2 py-1",
          paused ? "bg-destructive/20 text-destructive" : "text-muted-foreground hover:bg-accent/40"
        )}
        aria-label={paused ? "Resume" : "Pause"}
      >
        {paused ? "▶" : "⏸"}
      </button>
    </div>
  );
}
