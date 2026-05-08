import { Bar, BarChart, CartesianGrid, XAxis, YAxis, Cell } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { formatMoney } from "@/lib/format";

const config: ChartConfig = {
  wasted_usd: { label: "Wasted", color: "hsl(var(--destructive))" },
};

export type WastefulRow = { name: string; wasted_usd: number; spend_usd: number };

export function WastefulBarChart({
  title,
  rows,
  isLoading,
}: {
  title: string;
  rows: WastefulRow[];
  isLoading: boolean;
}) {
  const top5 = [...(rows ?? [])].sort((a, b) => b.wasted_usd - a.wasted_usd).slice(0, 5);
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-48 w-full" />
        ) : top5.length === 0 ? (
          <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">No data for this range.</div>
        ) : (
          <ChartContainer config={config}>
            <BarChart data={top5} layout="vertical" margin={{ left: 24 }}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
              <XAxis type="number" tickFormatter={(v) => formatMoney(Number(v))} tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <YAxis dataKey="name" type="category" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" width={120} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="wasted_usd" radius={[0, 4, 4, 0]}>
                {top5.map((_, i) => (
                  <Cell key={i} fill="hsl(var(--destructive))" />
                ))}
              </Bar>
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
