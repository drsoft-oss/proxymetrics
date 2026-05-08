import { useMemo } from "react";
import { useTargetDetail } from "@/hooks/useTargetDetail";
import { useFilter } from "@/hooks/useFilter";
import { DrillSection } from "@/components/primitives/DrillSheet";
import { KpiTile } from "@/components/overview/KpiTile";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusMixBar } from "./StatusMixBar";
import { formatMoney, formatNumber, formatPercent } from "@/lib/format";
import type { StatusClass } from "@/types/api";

export function TargetDrill({ host }: { host: string }) {
  const { filter } = useFilter();
  const detail = useTargetDetail(host, filter);

  const totals = useMemo(() => {
    const t = { class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0, requests: 0, spend: 0, wasted: 0 };
    if (!detail.data) return t;
    for (const row of detail.data.by_status_class) {
      t.requests += row.request_count;
      t.spend += row.spend_usd;
      const sc = row.status_class as StatusClass;
      if (sc === "2xx") t.class2xx += row.request_count;
      else if (sc === "3xx") t.class3xx += row.request_count;
      else if (sc === "4xx") t.class4xx += row.request_count;
      else if (sc === "5xx") t.class5xx += row.request_count;
      if (sc !== "2xx") t.wasted += row.spend_usd;
    }
    return t;
  }, [detail.data]);

  if (detail.isLoading) {
    return (
      <div className="space-y-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }
  if (detail.error) {
    return <div className="p-4 text-sm text-destructive">Couldn't load target: {(detail.error as Error).message}</div>;
  }
  if (!detail.data) return null;

  const errRate = totals.requests > 0 ? (totals.requests - totals.class2xx) / totals.requests : 0;

  return (
    <>
      <DrillSection title="KPIs">
        <div className="grid grid-cols-2 gap-2">
          <KpiTile label="Requests" value={formatNumber(totals.requests)} />
          <KpiTile label="Wasted" value={formatMoney(totals.wasted)} variant={totals.wasted > 0 ? "hero" : "default"} />
          <KpiTile label="Spend" value={formatMoney(totals.spend)} />
          <KpiTile label="Error rate" value={formatPercent(errRate)} />
        </div>
      </DrillSection>
      <DrillSection title="Status mix">
        <StatusMixBar mix={totals} width={400} />
        <div className="mt-2 grid grid-cols-4 gap-1 text-[11px] text-muted-foreground">
          <span>2xx {totals.class2xx}</span>
          <span>3xx {totals.class3xx}</span>
          <span>4xx {totals.class4xx}</span>
          <span>5xx {totals.class5xx}</span>
        </div>
      </DrillSection>
      <DrillSection title="Top providers">
        {detail.data.by_provider.length === 0 ? (
          <div className="text-xs text-muted-foreground">No providers in window.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {[...detail.data.by_provider].sort((a, b) => b.request_count - a.request_count).slice(0, 5).map((p) => (
              <li key={p.profile_id} className="flex items-center justify-between gap-2">
                <span className="truncate">{p.vendor}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatNumber(p.request_count)} reqs · {formatMoney(p.wasted_usd)} wasted
                </span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
    </>
  );
}
