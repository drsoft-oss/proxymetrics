import { useMemo } from "react";
import { DataTable, type Column } from "@/components/primitives/DataTable";
import { formatMoney, formatNumber, formatPercent } from "@/lib/format";
import type { StatusCodeDistItem } from "@/types/api";

export function StatusCodesTable({
  items,
  onRowClick,
  isLoading,
  error,
  onRetry,
}: {
  items: StatusCodeDistItem[];
  onRowClick: (row: StatusCodeDistItem) => void;
  isLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
}) {
  const total = useMemo(() => items.reduce((a, it) => a + it.requests, 0), [items]);

  const columns: Column<StatusCodeDistItem>[] = [
    { key: "code", header: "Code", sortable: true, sortValue: (r) => r.code, render: (r) => String(r.code) },
    { key: "requests", header: "Requests", sortable: true, align: "right", sortValue: (r) => r.requests, render: (r) => formatNumber(r.requests) },
    {
      key: "pct", header: "% of total", sortable: true, align: "right",
      sortValue: (r) => (total > 0 ? r.requests / total : 0),
      render: (r) => total > 0 ? formatPercent(r.requests / total) : "—",
    },
    {
      key: "wasted", header: "Wasted", sortable: true, align: "right",
      sortValue: (r) => r.wasted_usd,
      render: (r) => <span className={r.wasted_usd > 0 ? "text-destructive" : undefined}>{formatMoney(r.wasted_usd)}</span>,
    },
    { key: "spend", header: "Spend", sortable: true, align: "right", sortValue: (r) => r.spend_usd, render: (r) => formatMoney(r.spend_usd) },
    { key: "top_provider", header: "Top provider", render: (r) => r.top_provider_vendor || "—" },
  ];

  return (
    <DataTable
      columns={columns}
      rows={items}
      isLoading={isLoading}
      error={error ?? null}
      onRetry={onRetry}
      onRowClick={onRowClick}
      sortBy={{ key: "requests", dir: "desc" }}
      groupBy={(r) => r.class}
      emptyMessage="No status codes in this filter window."
    />
  );
}
