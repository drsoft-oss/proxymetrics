import { Pie, PieChart, Cell } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, type ChartConfig } from "@/components/ui/chart";
import { statusFill } from "@/lib/colors";
import type { LiveAggregates } from "@/hooks/useLiveBuffer";
import type { StatusClass } from "@/types/api";

const config: ChartConfig = { count: { label: "count", color: "hsl(var(--primary))" } };

export function StatusMixDonut({ a }: { a: LiveAggregates }) {
  const slices = (["2xx", "3xx", "4xx", "5xx"] as StatusClass[])
    .map((sc) => ({
      name: sc,
      value: sc === "2xx" ? a.statusMix.class2xx : sc === "3xx" ? a.statusMix.class3xx : sc === "4xx" ? a.statusMix.class4xx : a.statusMix.class5xx,
      fill: statusFill(sc),
    }))
    .filter((s) => s.value > 0);
  return (
    <Card>
      <CardHeader><CardTitle className="text-sm">Status mix (live)</CardTitle></CardHeader>
      <CardContent>
        {slices.length === 0 ? (
          <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">Waiting for traffic…</div>
        ) : (
          <ChartContainer config={config}>
            <PieChart>
              <Pie data={slices} dataKey="value" nameKey="name" innerRadius={48} outerRadius={80}>
                {slices.map((s) => <Cell key={s.name} fill={s.fill} />)}
              </Pie>
            </PieChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
