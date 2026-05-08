import type { RunSummary } from "@/types/audit";

type Props = { rows: RunSummary[]; onRowClick: (id: string) => void };

function poolCell(r: RunSummary): { text: string; cls: string } {
  if (r.unique_ip_count === undefined || r.completed_count <= 1) {
    return { text: "—", cls: "text-muted-foreground" };
  }
  const pct = Math.round((r.unique_ip_count / r.completed_count) * 100);
  const cls =
    pct >= 90 ? "text-green-500" :
    pct >= 50 ? "text-amber-500" :
                "text-red-500";
  return { text: `${pct}%`, cls };
}

export function AuditHistoryTable({ rows, onRowClick }: Props) {
  if (rows.length === 0) {
    return <div className="p-4 text-sm text-muted-foreground">No audits yet — start one from the Geo audit tab.</div>;
  }
  return (
    <table className="w-full text-sm">
      <thead className="text-xs uppercase tracking-wide text-muted-foreground">
        <tr>
          <th className="text-left px-3 py-2">ID</th>
          <th className="text-left px-3 py-2">Started</th>
          <th className="text-left px-3 py-2">Provider</th>
          <th className="text-left px-3 py-2">Place</th>
          <th className="text-left px-3 py-2">Type</th>
          <th className="text-right px-3 py-2">Count</th>
          <th className="text-right px-3 py-2">Accuracy</th>
          <th className="text-right px-3 py-2">Type-honest</th>
          <th className="text-right px-3 py-2">Pool</th>
          <th className="text-left px-3 py-2">Status</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => {
          const acc = r.completed_count > 0 ? Math.round((r.location_match_count / r.completed_count) * 100) : 0;
          const honest = r.completed_count > 0 ? Math.round((r.type_match_count / r.completed_count) * 100) : 0;
          return (
            <tr
              key={r.id}
              role="row"
              aria-label={r.id}
              tabIndex={0}
              onClick={() => onRowClick(r.id)}
              className="cursor-pointer hover:bg-accent/40"
            >
              <td className="px-3 py-1.5 font-mono text-xs">{r.id}</td>
              <td className="px-3 py-1.5">{new Date(r.started_at).toLocaleString()}</td>
              <td className="px-3 py-1.5">{r.provider ?? "—"}</td>
              <td className="px-3 py-1.5">{[r.expected_country, r.expected_state, r.expected_city].filter(Boolean).join(" / ")}</td>
              <td className="px-3 py-1.5">{r.expected_type}</td>
              <td className="px-3 py-1.5 text-right">{r.request_count}</td>
              <td className="px-3 py-1.5 text-right">{acc}%</td>
              <td className="px-3 py-1.5 text-right">{honest}%</td>
              {(() => {
                const { text, cls } = poolCell(r);
                return (
                  <td
                    data-testid={`pool-${r.id}`}
                    className={"px-3 py-1.5 text-right " + cls}
                  >
                    {text}
                  </td>
                );
              })()}
              <td className="px-3 py-1.5">
                <StatusPill status={r.status} />
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function StatusPill({ status }: { status: RunSummary["status"] }) {
  const cls =
    status === "completed" ? "bg-green-500/20 text-green-700 dark:text-green-400" :
    status === "cancelled" ? "bg-muted text-muted-foreground" :
    status === "failed"    ? "bg-red-500/20 text-red-700 dark:text-red-400" :
                             "bg-blue-500/20 text-blue-700 dark:text-blue-400";
  return <span className={"rounded px-1.5 py-0.5 text-xs " + cls}>{status}</span>;
}
