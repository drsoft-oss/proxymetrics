import { useMemo } from "react";
import { DataTable, type Column } from "@/components/primitives/DataTable";
import { formatNumber, formatPercent } from "@/lib/format";
import type { CaptchaKind, CaptchaRow } from "@/types/api";

type Row = {
  vendor: string;
  type: string;
  kind: CaptchaKind;
  requests: number;
};

function flatten(rows: CaptchaRow[]): Row[] {
  const out: Row[] = [];
  for (const r of rows) {
    for (const [k, n] of Object.entries(r.by_kind)) {
      if (!n) continue;
      out.push({ vendor: r.vendor, type: r.type, kind: k as CaptchaKind, requests: n });
    }
  }
  return out;
}

export function CaptchaTable({
  rows,
  onRowClick,
  isLoading,
  error,
  onRetry,
}: {
  rows: CaptchaRow[];
  onRowClick: (row: Row) => void;
  isLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
}) {
  const flat = useMemo(() => flatten(rows), [rows]);
  const total = useMemo(() => flat.reduce((a, r) => a + r.requests, 0), [flat]);

  const columns: Column<Row>[] = [
    { key: "vendor",   header: "Vendor",   sortable: true, sortValue: (r) => r.vendor,   render: (r) => r.vendor },
    { key: "type",     header: "Type",     sortable: true, sortValue: (r) => r.type,     render: (r) => r.type },
    { key: "kind",     header: "Kind",     sortable: true, sortValue: (r) => r.kind,     render: (r) => r.kind },
    {
      key: "requests", header: "Requests", sortable: true, align: "right",
      sortValue: (r) => r.requests, render: (r) => formatNumber(r.requests),
    },
    {
      key: "pct", header: "% of total", sortable: true, align: "right",
      sortValue: (r) => (total > 0 ? r.requests / total : 0),
      render: (r) => total > 0 ? formatPercent(r.requests / total) : "—",
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={flat}
      isLoading={isLoading}
      error={error ?? null}
      onRetry={onRetry}
      onRowClick={onRowClick}
      sortBy={{ key: "requests", dir: "desc" }}
      groupBy={(r) => r.kind}
      emptyMessage="No captcha hits in this filter window."
    />
  );
}

export type CaptchaTableRow = Row;
