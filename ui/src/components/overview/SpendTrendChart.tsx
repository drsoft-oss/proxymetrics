import { useMemo } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { RollupRow, StatusClass } from "@/types/api";
import { statusFill } from "@/lib/colors";
import { formatMoney } from "@/lib/format";

const STATUS_ORDER: StatusClass[] = ["2xx", "3xx", "4xx", "5xx", "timeout", "connection_error"];

const chartConfig: ChartConfig = {
  "2xx": { label: "2xx", color: statusFill("2xx") },
  "3xx": { label: "3xx", color: statusFill("3xx") },
  "4xx": { label: "4xx", color: statusFill("4xx") },
  "5xx": { label: "5xx", color: statusFill("5xx") },
  timeout: { label: "timeout", color: statusFill("timeout") },
  connection_error: { label: "conn err", color: statusFill("connection_error") },
};

type Bucket = { ts: string } & Record<StatusClass, number>;

export function SpendTrendChart({
  rollups,
  isLoading,
}: {
  rollups?: RollupRow[];
  isLoading: boolean;
}) {
  const buckets = useMemo<Bucket[]>(() => {
    if (!rollups) return [];
    const m = new Map<string, Bucket>();
    for (const r of rollups) {
      let b = m.get(r.ts_bucket);
      if (!b) {
        b = { ts: r.ts_bucket, "2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0, timeout: 0, connection_error: 0 };
        m.set(r.ts_bucket, b);
      }
      b[r.status_class] += r.cost_usd_total;
    }
    return [...m.values()].sort((a, b) => a.ts.localeCompare(b.ts));
  }, [rollups]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Spend trend (stacked by status_class)</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-60 w-full" />
        ) : buckets.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">No data for this range.</div>
        ) : (
          <ChartContainer config={chartConfig}>
            <AreaChart data={buckets}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
              <XAxis dataKey="ts" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <YAxis tickFormatter={(v) => formatMoney(Number(v))} tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" width={80} />
              <ChartTooltip content={<ChartTooltipContent />} />
              {STATUS_ORDER.map((sc) => (
                <Area
                  key={sc}
                  type="monotone"
                  dataKey={sc}
                  stackId="1"
                  stroke={statusFill(sc)}
                  fill={statusFill(sc)}
                  fillOpacity={0.6}
                />
              ))}
            </AreaChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
