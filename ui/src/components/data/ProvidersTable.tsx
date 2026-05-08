import { DataTable, type Column } from "@/components/primitives/DataTable";
import { StatusMixBar } from "./StatusMixBar";
import { Sparkline } from "./Sparkline";
import { formatMoney, formatNumber } from "@/lib/format";
import type { ProviderItem } from "@/types/api";

export function ProvidersTable({
  rows,
  isLoading,
  error,
  onRetry,
  onRowClick,
}: {
  rows: ProviderItem[];
  isLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  onRowClick: (row: ProviderItem) => void;
}) {
  const columns: Column<ProviderItem>[] = [
    {
      key: "vendor",
      header: "Provider",
      sortable: true,
      sortValue: (r) => r.vendor,
      render: (r) => <span>{r.vendor}</span>,
    },
    {
      key: "request_count",
      header: "Reqs",
      sortable: true,
      align: "right",
      sortValue: (r) => r.request_count,
      render: (r) => formatNumber(r.request_count),
    },
    {
      key: "wasted_usd",
      header: "Wasted",
      sortable: true,
      align: "right",
      sortValue: (r) => r.wasted_usd,
      render: (r) => (
        <span className={r.wasted_usd > 0 ? "text-destructive" : undefined}>{formatMoney(r.wasted_usd)}</span>
      ),
    },
    {
      key: "per_req",
      header: "$/req",
      sortable: true,
      align: "right",
      sortValue: (r) => (r.request_count > 0 ? r.spend_usd / r.request_count : 0),
      render: (r) =>
        r.request_count > 0 ? `$${(r.spend_usd / r.request_count).toFixed(4)}` : "—",
    },
    {
      key: "status_mix",
      header: "Status mix",
      render: (r) => <StatusMixBar mix={r.status_mix} />,
    },
    {
      key: "trend",
      header: "24h trend",
      render: (r) => <Sparkline data={r.requests_by_hour} />,
    },
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
      emptyMessage="No providers in this filter window. Try expanding the time range."
    />
  );
}
