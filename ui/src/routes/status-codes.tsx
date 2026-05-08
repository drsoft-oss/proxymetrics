import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useFilter } from "@/hooks/useFilter";
import { useStatusCodeDistribution } from "@/hooks/useStatusCodeDistribution";
import { StatusCodesHero } from "@/components/data/StatusCodesHero";
import { StatusCodesDistChart } from "@/components/data/StatusCodesDistChart";
import { StatusCodesTable } from "@/components/data/StatusCodesTable";
import { CodeDrill } from "@/components/data/CodeDrill";
import { DrillSheet } from "@/components/primitives/DrillSheet";

export const Route = createFileRoute("/status-codes")({
  component: StatusCodesPage,
});

function StatusCodesPage() {
  const { filter } = useFilter();
  const dist = useStatusCodeDistribution(filter);
  const [selected, setSelected] = useState<number | null>(null);

  const items = dist.data?.items ?? [];

  return (
    <div className="flex flex-col gap-4 p-4">
      <h1 className="text-lg font-semibold">Status Codes</h1>
      <StatusCodesHero items={items} />
      <StatusCodesDistChart items={items} isLoading={dist.isLoading} />
      <StatusCodesTable
        items={items}
        isLoading={dist.isLoading}
        error={dist.error as Error | null}
        onRetry={() => dist.refetch()}
        onRowClick={(r) => setSelected(r.code)}
      />
      <DrillSheet
        open={selected != null}
        onClose={() => setSelected(null)}
        title={selected != null ? `Status code ${selected}` : "Status code"}
      >
        {selected != null && <CodeDrill code={selected} />}
      </DrillSheet>
    </div>
  );
}
