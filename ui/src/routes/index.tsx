import { createFileRoute } from "@tanstack/react-router";
import { useFilter } from "@/hooks/useFilter";
import { useOverview } from "@/hooks/useOverview";
import { useRollups } from "@/hooks/useRollups";
import { useProviders } from "@/hooks/useProviders";
import { useTargets } from "@/hooks/useTargets";
import { useStatusCodes } from "@/hooks/useStatusCodes";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { OnboardingBanner } from "@/components/overview/OnboardingBanner";
import { KpiStrip } from "@/components/overview/KpiStrip";
import { SpendTrendChart } from "@/components/overview/SpendTrendChart";
import { WastefulBarChart, type WastefulRow } from "@/components/overview/WastefulBarChart";
import { StatusCodeDonut } from "@/components/overview/StatusCodeDonut";
import type { FingerprintResponse } from "@/types/api";

export const Route = createFileRoute("/")({
  component: OverviewPage,
});

function pickLevel(from: string, to: string): "1hour" | "1day" {
  const span = new Date(to).getTime() - new Date(from).getTime();
  return span > 30 * 24 * 60 * 60 * 1000 ? "1day" : "1hour";
}

function OverviewPage() {
  const { filter } = useFilter();
  const overview = useOverview(filter);
  const rollups = useRollups(pickLevel(filter.from, filter.to), filter);
  const providers = useProviders(filter);
  const targets = useTargets(filter);
  const statusCodes = useStatusCodes(filter);
  const fingerprint = useQuery({
    queryKey: queryKeys.fingerprint(),
    queryFn: () => apiGet<FingerprintResponse>("/cacert/fingerprint"),
    staleTime: Infinity,
  });

  const providerRows: WastefulRow[] = (providers.data?.items ?? []).map((p) => ({
    name: p.label || p.profile_id,
    wasted_usd: p.wasted_usd,
    spend_usd: p.spend_usd,
  }));
  const targetRows: WastefulRow[] = (targets.data?.items ?? []).map((t) => ({
    name: t.target_host,
    wasted_usd: t.wasted_usd,
    spend_usd: t.spend_usd,
  }));

  return (
    <div className="flex flex-col gap-4 p-4">
      <OnboardingBanner
        requestCount={overview.data?.request_count ?? 0}
        fingerprint={fingerprint.data?.sha256}
      />
      <KpiStrip data={overview.data} isLoading={overview.isLoading} error={overview.error as Error | null} />
      <SpendTrendChart rollups={rollups.data?.items} isLoading={rollups.isLoading} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <WastefulBarChart title="Top 5 wasteful providers" rows={providerRows} isLoading={providers.isLoading} />
        <WastefulBarChart title="Top 5 wasteful targets" rows={targetRows} isLoading={targets.isLoading} />
      </div>
      <StatusCodeDonut data={statusCodes.data} isLoading={statusCodes.isLoading} />
    </div>
  );
}
