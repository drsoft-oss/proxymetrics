import { useCaptchaDetail } from "@/hooks/useCaptchaDetail";
import { useFilter } from "@/hooks/useFilter";
import { DrillSection } from "@/components/primitives/DrillSheet";
import { KpiTile } from "@/components/overview/KpiTile";
import { Skeleton } from "@/components/ui/skeleton";
import { formatNumber } from "@/lib/format";
import type { CaptchaKind } from "@/types/api";

export function CaptchaDrill({ kind }: { kind: CaptchaKind }) {
  const { filter } = useFilter();
  const detail = useCaptchaDetail(kind, filter);

  if (detail.isLoading) {
    return (
      <div className="space-y-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }
  if (detail.error) {
    return <div className="p-4 text-sm text-destructive">Couldn't load kind: {(detail.error as Error).message}</div>;
  }
  if (!detail.data) return null;
  const d = detail.data;

  return (
    <>
      <DrillSection title="KPIs">
        <div className="grid grid-cols-2 gap-2">
          <KpiTile label="Requests" value={formatNumber(d.requests)} />
          <KpiTile label="Kind" value={d.kind} />
        </div>
      </DrillSection>
      <DrillSection title="Top providers serving this kind">
        {d.top_providers.length === 0 ? (
          <div className="text-xs text-muted-foreground">No data.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {d.top_providers.map((p) => (
              <li key={`${p.vendor}-${p.type}`} className="flex items-center justify-between gap-2">
                <span className="truncate">{p.vendor} · {p.type}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{formatNumber(p.requests)} reqs</span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
      <DrillSection title="Top targets serving this kind">
        {d.top_targets.length === 0 ? (
          <div className="text-xs text-muted-foreground">No data.</div>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {d.top_targets.map((t) => (
              <li key={t.host} className="flex items-center justify-between gap-2">
                <span className="truncate">{t.host}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{formatNumber(t.requests)} reqs</span>
              </li>
            ))}
          </ul>
        )}
      </DrillSection>
    </>
  );
}
