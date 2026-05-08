import { useMemo } from "react";
import { KpiTile } from "@/components/overview/KpiTile";
import { formatMoney, formatNumber, formatPercent } from "@/lib/format";
import type { StatusCodeDistItem } from "@/types/api";

type Cls = "2xx" | "3xx" | "4xx" | "5xx";
const CLASSES: Cls[] = ["2xx", "3xx", "4xx", "5xx"];

export function StatusCodesHero({ items }: { items: StatusCodeDistItem[] }) {
  const summary = useMemo(() => {
    const total = items.reduce((acc, it) => acc + it.requests, 0);
    const byClass: Record<Cls, { requests: number; wasted: number; spend: number }> = {
      "2xx": { requests: 0, wasted: 0, spend: 0 },
      "3xx": { requests: 0, wasted: 0, spend: 0 },
      "4xx": { requests: 0, wasted: 0, spend: 0 },
      "5xx": { requests: 0, wasted: 0, spend: 0 },
    };
    for (const it of items) {
      byClass[it.class].requests += it.requests;
      byClass[it.class].wasted += it.wasted_usd;
      byClass[it.class].spend += it.spend_usd;
    }
    const ifEliminated = byClass["4xx"].wasted + byClass["5xx"].wasted;
    return { total, byClass, ifEliminated };
  }, [items]);

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
      <KpiTile
        label="If every non-2xx had succeeded"
        value={formatMoney(summary.ifEliminated)}
        sub={`4xx: ${formatMoney(summary.byClass["4xx"].wasted)} · 5xx: ${formatMoney(summary.byClass["5xx"].wasted)}`}
        variant="hero"
      />
      {CLASSES.map((cls) => {
        const pct = summary.total > 0 ? summary.byClass[cls].requests / summary.total : 0;
        const wasted = summary.byClass[cls].wasted;
        return (
          <KpiTile
            key={cls}
            label={cls}
            value={formatPercent(pct)}
            sub={`${formatNumber(summary.byClass[cls].requests)} reqs${wasted > 0 ? ` · ${formatMoney(wasted)} wasted` : ""}`}
          />
        );
      })}
    </div>
  );
}
