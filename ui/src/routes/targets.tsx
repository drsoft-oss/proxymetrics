import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useFilter } from "@/hooks/useFilter";
import { useTargets } from "@/hooks/useTargets";
import { TargetsTable } from "@/components/data/TargetsTable";
import { TargetDrill } from "@/components/data/TargetDrill";
import { DrillSheet } from "@/components/primitives/DrillSheet";

export const Route = createFileRoute("/targets")({
  component: TargetsPage,
});

function TargetsPage() {
  const { filter } = useFilter();
  const targets = useTargets(filter);
  const [selected, setSelected] = useState<string | null>(null);

  return (
    <div className="flex flex-col gap-3 p-4">
      <h1 className="text-lg font-semibold">Targets</h1>
      <TargetsTable
        rows={targets.data?.items ?? []}
        isLoading={targets.isLoading}
        error={targets.error as Error | null}
        onRetry={() => targets.refetch()}
        onRowClick={(r) => setSelected(r.target_host)}
      />
      <DrillSheet
        open={selected != null}
        onClose={() => setSelected(null)}
        title={selected ?? "Target"}
      >
        {selected && <TargetDrill host={selected} />}
      </DrillSheet>
    </div>
  );
}
