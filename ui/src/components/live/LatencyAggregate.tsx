import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatLatency } from "@/lib/format";
import type { LiveAggregates } from "@/hooks/useLiveBuffer";

export function LatencyAggregate({ a }: { a: LiveAggregates }) {
  return (
    <Card>
      <CardHeader><CardTitle className="text-sm">Latency (live window)</CardTitle></CardHeader>
      <CardContent>
        <dl className="grid grid-cols-2 gap-2 text-sm">
          <dt className="text-muted-foreground">p50</dt><dd className="text-right">{formatLatency(a.p50Ms)}</dd>
          <dt className="text-muted-foreground">p95</dt><dd className="text-right">{formatLatency(a.p95Ms)}</dd>
          <dt className="text-muted-foreground">p99</dt><dd className="text-right">{formatLatency(a.p99Ms)}</dd>
          <dt className="text-muted-foreground">max</dt><dd className="text-right">{formatLatency(a.maxMs)}</dd>
        </dl>
      </CardContent>
    </Card>
  );
}
