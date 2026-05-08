import { useMemo } from "react";
import { Cell, Pie, PieChart } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ChartContainer, ChartTooltip, type ChartConfig } from "@/components/ui/chart";
import { statusFill } from "@/lib/colors";
import type { StatusCodesResponse, StatusClass } from "@/types/api";
import { formatMoney } from "@/lib/format";

const STATUS_ORDER: StatusClass[] = ["2xx", "3xx", "4xx", "5xx", "timeout", "connection_error"];

const config: ChartConfig = {
  spend_usd: { label: "Spend", color: "hsl(var(--primary))" },
};

export function StatusCodeDonut({ data, isLoading }: { data?: StatusCodesResponse; isLoading: boolean }) {
  const slices = useMemo(() => {
    if (!data) return [];
    const totals: Record<StatusClass, number> = {
      "2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0, timeout: 0, connection_error: 0,
    };
    for (const row of data.rows) {
      for (const sc of STATUS_ORDER) {
        const cell = row.by_status[sc];
        if (cell) totals[sc] += cell.spend_usd;
      }
    }
    return STATUS_ORDER.filter((sc) => totals[sc] > 0).map((sc) => ({
      name: sc,
      value: totals[sc],
      fill: statusFill(sc),
    }));
  }, [data]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Status code distribution</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-60 w-full" />
        ) : slices.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">No data for this range.</div>
        ) : (
          <ChartContainer config={config}>
            <PieChart>
              <ChartTooltip
                content={({ active, payload }) => {
                  if (!active || !payload?.length) return null;
                  const p = payload[0];
                  return (
                    <div className="rounded-md border border-border bg-popover p-2 text-xs shadow-md">
                      <div className="font-medium">{p.name}</div>
                      <div className="text-muted-foreground">{formatMoney(Number(p.value))}</div>
                    </div>
                  );
                }}
              />
              <Pie data={slices} dataKey="value" nameKey="name" innerRadius={60} outerRadius={90} paddingAngle={2}>
                {slices.map((s, i) => (
                  <Cell key={i} fill={s.fill} />
                ))}
              </Pie>
            </PieChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
