import { Bar, BarChart, CartesianGrid, XAxis, YAxis, Cell } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { statusFill } from "@/lib/colors";
import type { StatusCodeDistItem, StatusClass } from "@/types/api";

const config: ChartConfig = {
  requests: { label: "Requests", color: "hsl(var(--primary))" },
};

export function StatusCodesDistChart({
  items,
  isLoading,
}: {
  items: StatusCodeDistItem[];
  isLoading?: boolean;
}) {
  // Sort by class then numeric code for a clean visual.
  const ordered = [...items].sort((a, b) =>
    a.class === b.class ? a.code - b.code : a.class.localeCompare(b.class)
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Status code distribution</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-60 w-full" />
        ) : ordered.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">No data for this range.</div>
        ) : (
          <ChartContainer config={config}>
            <BarChart data={ordered}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
              <XAxis dataKey="code" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="requests">
                {ordered.map((it) => (
                  <Cell key={it.code} fill={statusFill(it.class as StatusClass)} />
                ))}
              </Bar>
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
