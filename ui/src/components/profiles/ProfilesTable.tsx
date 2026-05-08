import { DataTable, type Column } from "@/components/primitives/DataTable";
import type { ProfileWithUsage } from "@/types/profile";

export type ProfilesTableProps = {
  rows: ProfileWithUsage[];
  loading: boolean;
  onRowClick: (id: string) => void;
};

const columns: Column<ProfileWithUsage>[] = [
  {
    key: "id",
    header: "ID",
    sortable: true,
    sortValue: (r) => r.id,
    render: (r) => <code className="text-xs">{r.id}</code>,
  },
  {
    key: "label",
    header: "Label",
    sortable: true,
    sortValue: (r) => r.label,
    render: (r) => <span>{r.label}</span>,
  },
  {
    key: "vendor",
    header: "Vendor",
    sortable: true,
    sortValue: (r) => r.vendor,
    render: (r) => (
      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wider">
        {r.vendor}
      </span>
    ),
  },
  {
    key: "type",
    header: "Type",
    sortable: true,
    sortValue: (r) => r.type,
    render: (r) => (
      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wider">
        {r.type}
      </span>
    ),
  },
  {
    key: "price_per_gb",
    header: "$/GB",
    sortable: true,
    align: "right",
    sortValue: (r) => r.price_per_gb ?? -1,
    render: (r) =>
      r.price_per_gb == null ? (
        <span className="text-muted-foreground">—</span>
      ) : (
        <span>{r.price_per_gb.toFixed(2)}</span>
      ),
  },
  {
    key: "requests",
    header: "Reqs 24h",
    sortable: true,
    align: "right",
    sortValue: (r) => r.requests,
    render: (r) => <span>{formatCompactInt(r.requests)}</span>,
  },
  {
    key: "spend_usd",
    header: "Spend 24h",
    sortable: true,
    align: "right",
    sortValue: (r) => r.spend_usd,
    render: (r) => <span>${r.spend_usd.toFixed(2)}</span>,
  },
];

function formatCompactInt(n: number): string {
  if (n < 1000) return String(n);
  if (n < 1_000_000) return (n / 1000).toFixed(n < 10_000 ? 1 : 0) + "k";
  return (n / 1_000_000).toFixed(1) + "M";
}

export function ProfilesTable({ rows, loading, onRowClick }: ProfilesTableProps) {
  if (!loading && rows.length === 0) {
    return (
      <div className="rounded-md border border-border bg-background p-8 text-center text-sm text-muted-foreground">
        No profiles yet — they'll appear here once traffic flows through ProxyMetrics.
      </div>
    );
  }
  return (
    <DataTable<ProfileWithUsage>
      rows={rows}
      columns={columns}
      isLoading={loading}
      sortBy={{ key: "requests", dir: "desc" }}
      onRowClick={(r) => onRowClick(r.id)}
    />
  );
}
