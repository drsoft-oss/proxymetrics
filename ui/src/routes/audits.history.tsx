import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useAudits } from "@/hooks/useAudits";
import { AuditHistoryTable } from "@/components/audits/AuditHistoryTable";

export const Route = createFileRoute("/audits/history")({
  component: HistoryTab,
});

function HistoryTab() {
  const audits = useAudits();
  const nav = useNavigate({ from: "/audits/history" });
  return (
    <AuditHistoryTable
      rows={audits.data ?? []}
      onRowClick={(id) => nav({ to: "/audits/$id", params: { id } })}
    />
  );
}
