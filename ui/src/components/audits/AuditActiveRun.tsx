import { useAuditDetail } from "@/hooks/useAudits";
import { useAuditEvents } from "@/hooks/useAuditEvents";
import { AuditResultsSheet } from "./AuditResultsSheet";
import { AuditMap } from "./AuditMap";
import type { RequestRow } from "@/types/audit";

type Props = { runId: string; onCancelled: () => void };

export function AuditActiveRun({ runId, onCancelled }: Props) {
  const detail = useAuditDetail(runId);
  const events = useAuditEvents(detail.data?.run.status === "running" ? runId : null);

  if (!detail.data) return <div className="p-4 text-sm">Loading…</div>;
  const run = detail.data.run;
  const rows: RequestRow[] = events.requests.length > 0 ? events.requests : detail.data.requests;
  const live = run.status === "running" && !events.finished;

  const cancel = async () => {
    await fetch(`/api/v1/audits/${runId}/cancel`, { method: "POST" });
    onCancelled();
  };

  const zoom = run.check_level === "city" ? 9 : run.check_level === "state" ? 5 : 4;

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 border-b p-3 text-sm">
        <span className="font-semibold">Run</span>
        <span>{run.provider ?? "—"}</span>
        <span>·</span>
        <span>{[run.expected_country, run.expected_state, run.expected_city].filter(Boolean).join(" / ")}</span>
        <span>·</span>
        <span>{rows.filter((r) => !r.error).length} / {run.request_count}</span>
        {live && (
          <button onClick={cancel} className="ml-auto rounded border px-2 py-0.5 text-xs">Stop</button>
        )}
      </div>
      <div className="flex flex-1 min-h-0">
        <div className="w-[38%] overflow-y-auto border-r p-3">
          <AuditResultsSheet
            rows={rows}
            total={run.request_count}
            expectedCity={run.expected_city}
            sessionKey={run.session_key}
            uniqueIPCount={run.unique_ip_count}
            completedCount={run.completed_count}
          />
        </div>
        <div className="flex-1">
          <AuditMap
            rows={rows}
            center={[run.expected_lat, run.expected_lon]}
            zoom={zoom}
            expected={{ country: run.expected_country, state: run.expected_state, city: run.expected_city }}
          />
        </div>
      </div>
    </div>
  );
}
