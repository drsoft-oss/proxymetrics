import { useMemo } from "react";
import { KpiTile } from "@/components/overview/KpiTile";
import { formatNumber, formatPercent } from "@/lib/format";
import type { CaptchaDistributionItem, CaptchasResponse } from "@/types/api";

export function CaptchaHero({
  distribution,
  vendorRows,
  totalRequests,
}: {
  distribution: CaptchaDistributionItem[];
  vendorRows: CaptchasResponse["rows"];
  totalRequests: number;
}) {
  const summary = useMemo(() => {
    const captchaTotal = distribution.reduce((acc, it) => acc + it.requests, 0);
    const top = distribution.reduce<CaptchaDistributionItem | null>(
      (best, it) => (best == null || it.requests > best.requests ? it : best),
      null,
    );
    const pct = totalRequests > 0 ? captchaTotal / totalRequests : 0;
    void vendorRows;
    return { captchaTotal, top, pct };
  }, [distribution, vendorRows, totalRequests]);

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
      <KpiTile label="Captcha hits in window" value={formatNumber(summary.captchaTotal)} variant="hero" />
      <KpiTile
        label="% of requests that hit a captcha"
        value={totalRequests > 0 ? formatPercent(summary.pct) : "—"}
        sub={totalRequests > 0 ? `${formatNumber(totalRequests)} total reqs` : "no requests in window"}
      />
      <KpiTile
        label="Top kind"
        value={summary.top ? summary.top.kind : "—"}
        sub={summary.top ? `${formatNumber(summary.top.requests)} hits` : undefined}
      />
    </div>
  );
}
