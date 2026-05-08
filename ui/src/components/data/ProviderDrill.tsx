import { useMemo } from "react";
import { useProviderDetail } from "@/hooks/useProviderDetail";
import { useFilter } from "@/hooks/useFilter";
import { DrillSection } from "@/components/primitives/DrillSheet";
import { KpiTile } from "@/components/overview/KpiTile";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusMixBar } from "./StatusMixBar";
import { formatMoney, formatNumber, formatPercent } from "@/lib/format";
import type { StatusClass, StatusMix } from "@/types/api";

export function ProviderDrill({ profileId }: { profileId: string }) {
  const { filter } = useFilter();
  const detail = useProviderDetail(profileId, filter);

  const totals = useMemo(() => {
    const empty: StatusMix & { requests: number; spend: number; wasted: number } = {
      class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0, requests: 0, spend: 0, wasted: 0,
    };
    if (!detail.data) return empty;
    let requests = 0, spend = 0, wasted = 0;
    const mix = { class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0 };
    for (const row of detail.data.by_status_class) {
      requests += row.request_count;
      spend += row.spend_usd;
      const sc = row.status_class as StatusClass;
      if (sc === "2xx") mix.class2xx += row.request_count;
      else if (sc === "3xx") mix.class3xx += row.request_count;
      else if (sc === "4xx") mix.class4xx += row.request_count;
      else if (sc === "5xx") mix.class5xx += row.request_count;
      if (sc !== "2xx") wasted += row.spend_usd;
    }
    return { ...mix, requests, spend, wasted };
  }, [detail.data]);

  if (detail.isLoading) {
    return (
      <div className="space-y-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }
  if (detail.error) {
    return <div className="p-4 text-sm text-destructive">Couldn't load provider: {(detail.error as Error).message}</div>;
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
      <DrillSection title="Top targets">
        {detail.data.top_targets.length === 0 ? (
          <div className="text-xs text-muted-foreground">No targets in window.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {detail.data.top_targets.slice(0, 5).map((t) => (
              <li key={t.target_host} className="flex items-center justify-between gap-2">
                <span className="truncate">{t.target_host}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatNumber(t.request_count)} reqs · {formatMoney(t.spend_usd)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
    </>
  );
}
