import { useStatusCodeDetail } from "@/hooks/useStatusCodeDetail";
import { useFilter } from "@/hooks/useFilter";
import { DrillSection } from "@/components/primitives/DrillSheet";
import { KpiTile } from "@/components/overview/KpiTile";
import { Skeleton } from "@/components/ui/skeleton";
import { formatMoney, formatNumber } from "@/lib/format";

export function CodeDrill({ code }: { code: number }) {
  const { filter } = useFilter();
  const detail = useStatusCodeDetail(code, filter);

  if (detail.isLoading) {
    return (
      <div className="space-y-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }
  if (detail.error) {
    return <div className="p-4 text-sm text-destructive">Couldn't load code: {(detail.error as Error).message}</div>;
  }
  if (!detail.data) return null;

  const d = detail.data;

  return (
    <>
      <DrillSection title="KPIs">
        <div className="grid grid-cols-2 gap-2">
          <KpiTile label="Requests" value={formatNumber(d.requests)} />
          <KpiTile label="Wasted" value={formatMoney(d.wasted_usd)} variant={d.wasted_usd > 0 ? "hero" : "default"} />
          <KpiTile label="Spend" value={formatMoney(d.spend_usd)} />
          <KpiTile label="Class" value={d.class} />
        </div>
      </DrillSection>
      <DrillSection title="Top providers emitting this code">
        {d.top_providers.length === 0 ? (
          <div className="text-xs text-muted-foreground">No data.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {d.top_providers.map((p) => (
              <li key={p.vendor} className="flex items-center justify-between gap-2">
                <span className="truncate">{p.vendor}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatNumber(p.requests)} reqs · {formatMoney(p.wasted_usd)} wasted
                </span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
      <DrillSection title="Top targets emitting this code">
        {d.top_targets.length === 0 ? (
          <div className="text-xs text-muted-foreground">No data.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {d.top_targets.map((t) => (
              <li key={t.host} className="flex items-center justify-between gap-2">
                <span className="truncate">{t.host}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatNumber(t.requests)} reqs · {formatMoney(t.wasted_usd)} wasted
                </span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
    </>
  );
}
