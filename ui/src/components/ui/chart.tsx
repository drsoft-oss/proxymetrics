import * as React from "react";
import { ResponsiveContainer, Tooltip as RechartsTooltip } from "recharts";
import { cn } from "@/lib/utils";

export type ChartConfig = Record<string, { label: string; color: string }>;

const ChartConfigCtx = React.createContext<ChartConfig | null>(null);

export function ChartContainer({
  config,
  className,
  children,
}: {
  config: ChartConfig;
  className?: string;
  children: React.ReactElement;
}) {
  return (
    <ChartConfigCtx.Provider value={config}>
      <div className={cn("w-full", className)} style={{ height: 240 }}>
        <ResponsiveContainer width="100%" height="100%">
          {children}
        </ResponsiveContainer>
      </div>
    </ChartConfigCtx.Provider>
  );
}

export function useChartConfig(): ChartConfig {
  const v = React.useContext(ChartConfigCtx);
  if (!v) throw new Error("useChartConfig must be used inside ChartContainer");
  return v;
}

export function ChartTooltipContent({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; dataKey: string; color: string }> }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-md border border-border bg-popover p-2 text-xs shadow-md">
      {payload.map((p) => (
        <div key={p.dataKey} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ backgroundColor: p.color }} />
          <span className="text-muted-foreground">{p.name}:</span>
          <span className="font-medium">{p.value}</span>
        </div>
      ))}
    </div>
  );
}

export { RechartsTooltip as ChartTooltip };
