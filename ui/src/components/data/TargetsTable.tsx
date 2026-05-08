import { DataTable, type Column } from "@/components/primitives/DataTable";
import { StatusMixBar } from "./StatusMixBar";
import { Sparkline } from "./Sparkline";
import { formatMoney, formatNumber } from "@/lib/format";
import type { TargetItem } from "@/types/api";

export function TargetsTable({
  rows,
  isLoading,
  error,
  onRetry,
  onRowClick,
}: {
  rows: TargetItem[];
  isLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  onRowClick: (row: TargetItem) => void;
}) {
  const columns: Column<TargetItem>[] = [
    { key: "target_host", header: "Target", sortable: true, sortValue: (r) => r.target_host, render: (r) => r.target_host },
    { key: "request_count", header: "Reqs", sortable: true, align: "right", sortValue: (r) => r.request_count, render: (r) => formatNumber(r.request_count) },
    {
      key: "wasted_usd", header: "Wasted", sortable: true, align: "right",
      sortValue: (r) => r.wasted_usd,
      render: (r) => <span className={r.wasted_usd > 0 ? "text-destructive" : undefined}>{formatMoney(r.wasted_usd)}</span>,
    },
    {
      key: "per_req", header: "$/req", sortable: true, align: "right",
      sortValue: (r) => (r.request_count > 0 ? r.spend_usd / r.request_count : 0),
      render: (r) => r.request_count > 0 ? `$${(r.spend_usd / r.request_count).toFixed(4)}` : "—",
    },
    { key: "status_mix", header: "Status mix", render: (r) => <StatusMixBar mix={r.status_mix} /> },
    { key: "trend", header: "24h trend", render: (r) => <Sparkline data={r.requests_by_hour} /> },
  ];

  return (
    <DataTable
      columns={columns}
      rows={rows}
      isLoading={isLoading}
      error={error ?? null}
      onRetry={onRetry}
      onRowClick={onRowClick}
      sortBy={{ key: "wasted_usd", dir: "desc" }}
      pageSize={25}
      emptyMessage="No targets in this filter window. Try expanding the time range."
    />
  );
}
