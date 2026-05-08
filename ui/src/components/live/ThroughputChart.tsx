import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import type { LiveAggregates } from "@/hooks/useLiveBuffer";

const config: ChartConfig = { reqPerSec: { label: "req/s", color: "hsl(var(--primary))" } };

export function ThroughputChart({ a }: { a: LiveAggregates }) {
  const data = a.throughputSeries.map((s) => ({ t: new Date(s.t * 1000).toLocaleTimeString("en-US", { hour12: false }), reqPerSec: s.reqPerSec }));
  return (
    <Card>
      <CardHeader><CardTitle className="text-sm">Throughput (req/s)</CardTitle></CardHeader>
      <CardContent>
        <ChartContainer config={config}>
          <AreaChart data={data}>
            <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
            <XAxis dataKey="t" tick={{ fontSize: 10 }} stroke="hsl(var(--muted-foreground))" />
            <YAxis tick={{ fontSize: 10 }} stroke="hsl(var(--muted-foreground))" />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area type="monotone" dataKey="reqPerSec" stroke="hsl(var(--primary))" fill="hsl(var(--primary))" fillOpacity={0.4} />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
