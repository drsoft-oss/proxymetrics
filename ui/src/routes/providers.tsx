import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useFilter } from "@/hooks/useFilter";
import { useProviders } from "@/hooks/useProviders";
import { ProvidersTable } from "@/components/data/ProvidersTable";
import { ProviderDrill } from "@/components/data/ProviderDrill";
import { DrillSheet } from "@/components/primitives/DrillSheet";

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
});

function ProvidersPage() {
  const { filter } = useFilter();
  const providers = useProviders(filter);
  const [selected, setSelected] = useState<string | null>(null);

  const selectedRow = providers.data?.items.find((r) => r.profile_id === selected);

  return (
    <div className="flex flex-col gap-3 p-4">
      <h1 className="text-lg font-semibold">Providers</h1>
      <ProvidersTable
        rows={providers.data?.items ?? []}
        isLoading={providers.isLoading}
        error={providers.error as Error | null}
        onRetry={() => providers.refetch()}
        onRowClick={(r) => setSelected(r.profile_id)}
      />
      <DrillSheet
        open={selected != null}
        onClose={() => setSelected(null)}
        title={selectedRow?.label || selectedRow?.vendor || "Provider"}
        subtitle={selectedRow ? `${selectedRow.vendor} · ${selectedRow.type}` : undefined}
      >
        {selected && <ProviderDrill profileId={selected} />}
      </DrillSheet>
    </div>
  );
}
