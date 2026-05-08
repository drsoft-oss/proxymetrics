import { KpiTile } from "./KpiTile";
import { Skeleton } from "@/components/ui/skeleton";
import type { OverviewResponse } from "@/types/api";
import { formatBytes, formatMoney, formatPercent } from "@/lib/format";

export function KpiStrip({
  data,
  isLoading,
  error,
}: {
  data?: OverviewResponse;
  isLoading: boolean;
  error?: Error | null;
}) {
  if (isLoading || !data) {
    return (
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    );
  }
  if (error) {
    return <div className="text-sm text-destructive">Couldn't load KPIs: {error.message}</div>;
  }
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
      <KpiTile label="Spend this cycle" value={formatMoney(data.spend_usd_total)} />
      <KpiTile
        label="$ Wasted on errors"
        value={formatMoney(data.wasted_usd_total)}
        sub={`${formatPercent(data.wasted_pct_of_spend)} of spend on non-2xx`}
        variant="hero"
      />
      <KpiTile label="Bandwidth" value={formatBytes(data.bytes_total)} />
      <KpiTile label="Success rate" value={formatPercent(data.success_rate)} />
    </div>
  );
}
