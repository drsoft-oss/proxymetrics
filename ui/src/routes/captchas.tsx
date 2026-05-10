import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useFilter } from "@/hooks/useFilter";
import { useCaptchas } from "@/hooks/useCaptchas";
import { useCaptchaDistribution } from "@/hooks/useCaptchaDistribution";
import { useOverview } from "@/hooks/useOverview";
import { CaptchaHero } from "@/components/data/CaptchaHero";
import { CaptchaByVendorChart } from "@/components/data/CaptchaByVendorChart";
import { CaptchaTable, type CaptchaTableRow } from "@/components/data/CaptchaTable";
import { CaptchaDrill } from "@/components/data/CaptchaDrill";
import { DrillSheet } from "@/components/primitives/DrillSheet";
import type { CaptchaKind } from "@/types/api";

export const Route = createFileRoute("/captchas")({
  component: CaptchasPage,
});

function CaptchasPage() {
  const { filter } = useFilter();
  const captchas = useCaptchas(filter);
  const dist = useCaptchaDistribution(filter);
  const overview = useOverview(filter);
  const [selected, setSelected] = useState<CaptchaKind | null>(null);

  const rows = captchas.data?.rows ?? [];
  const items = dist.data?.items ?? [];
  const totalRequests = overview.data?.request_count ?? 0;

  return (
    <div className="flex flex-col gap-4 p-4">
      <h1 className="text-lg font-semibold">Captchas</h1>
      <CaptchaHero distribution={items} vendorRows={rows} totalRequests={totalRequests} />
      <CaptchaByVendorChart rows={rows} isLoading={captchas.isLoading} />
      <CaptchaTable
        rows={rows}
        isLoading={captchas.isLoading}
        error={captchas.error as Error | null}
        onRetry={() => captchas.refetch()}
        onRowClick={(r: CaptchaTableRow) => setSelected(r.kind)}
      />
      <DrillSheet
        open={selected != null}
        onClose={() => setSelected(null)}
        title={selected != null ? `Captcha kind · ${selected}` : "Captcha kind"}
      >
        {selected != null && <CaptchaDrill kind={selected} />}
      </DrillSheet>
    </div>
  );
}
