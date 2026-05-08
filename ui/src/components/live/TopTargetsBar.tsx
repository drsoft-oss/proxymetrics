import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { LiveAggregates } from "@/hooks/useLiveBuffer";

const config: ChartConfig = { count: { label: "count", color: "hsl(var(--primary))" } };

export function TopTargetsBar({ a }: { a: LiveAggregates }) {
  return (
    <Card>
      <CardHeader><CardTitle className="text-sm">Top targets (live)</CardTitle></CardHeader>
      <CardContent>
        {a.topTargets.length === 0 ? (
          <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">Waiting for traffic…</div>
        ) : (
          <ChartContainer config={config}>
            <BarChart data={a.topTargets} layout="vertical" margin={{ left: 24 }}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
              <XAxis type="number" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <YAxis dataKey="host" type="category" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" width={140} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="count" fill="hsl(var(--primary))" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
