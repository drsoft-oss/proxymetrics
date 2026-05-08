import { KpiTile } from "@/components/overview/KpiTile";
import { formatLatency, formatNumber } from "@/lib/format";
import type { LiveAggregates } from "@/hooks/useLiveBuffer";

export function LiveKpiStrip({ a }: { a: LiveAggregates }) {
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
      <KpiTile label="Req/s" value={a.reqPerSec.toFixed(1)} />
      <KpiTile label="Active providers" value={formatNumber(a.activeProviders)} />
      <KpiTile label="Active targets" value={formatNumber(a.activeTargets)} />
      <KpiTile label="p95 latency" value={formatLatency(a.p95Ms)} />
    </div>
  );
}
