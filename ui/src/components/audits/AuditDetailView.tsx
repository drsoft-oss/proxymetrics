import { useEffect, useState, type ReactNode } from "react";
import { useAuditDetail } from "@/hooks/useAudits";
import { AuditResultsSheet } from "./AuditResultsSheet";
import { AuditMap } from "./AuditMap";
import type { RunSummary } from "@/types/audit";

export function AuditDetailView({ runId, onReRun }: { runId: string; onReRun: (runId: string) => void }) {
  const detail = useAuditDetail(runId);
  if (!detail.data) return <div className="p-4 text-sm">Loading…</div>;
  const run = detail.data.run;
  const rows = detail.data.requests;
  const zoom = run.check_level === "city" ? 9 : run.check_level === "state" ? 5 : 4;
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 border-b p-3 text-sm">
        <span className="font-semibold">Run</span>
        <span className="font-mono text-xs">{run.id}</span>
        <span>·</span>
        <span>{run.status}</span>
        <button onClick={() => onReRun(run.id)} className="ml-auto rounded border px-2 py-0.5 text-xs">Re-run</button>
      </div>
      <HeaderRow2 run={run} />
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
            expected={{ country: run.expected_country, state: run.expected_state, city: run.expected_city, type: run.expected_type }}
          />
        </div>
      </div>
    </div>
  );
}

function HeaderRow2({ run }: { run: RunSummary }) {
  const isRunning = run.status === "running";

  // Tick once per second while running so Duration updates live.
  const [, force] = useState(0);
  useEffect(() => {
    if (!isRunning) return;
    const id = setInterval(() => force((n) => n + 1), 1000);
    return () => clearInterval(id);
  }, [isRunning]);

  const items: Array<[string, ReactNode]> = [];
  items.push(["Proxy", <span className="font-mono text-xs">{run.proxy_url}</span>]);
  const loc = [run.expected_country, run.expected_state, run.expected_city].filter(Boolean).join(" / ");
  if (loc) items.push(["Location", loc]);
  items.push(["Type", run.expected_type]);
  if (run.provider) items.push(["Provider", run.provider]);
  items.push(["Session", run.session_key ? `rotating (${run.session_key})` : "static"]);

  const attempted = run.completed_count + run.error_count;
  items.push(["Checks", isRunning ? `${attempted} / ${run.request_count}` : `${run.request_count}`]);

  if (run.error_count > 0) {
    items.push([
      "Errors",
      <span data-testid="audit-header-errors" className="text-red-500">{run.error_count}</span>,
    ]);
  }

  items.push(["Started", new Date(run.started_at).toLocaleString()]);
  if (!isRunning && run.finished_at) {
    items.push(["Finished", new Date(run.finished_at).toLocaleString()]);
  }
  items.push(["Duration", formatDuration(run.started_at, run.finished_at, isRunning)]);

  if (run.completed_count > 0) {
    const acc = Math.round((run.location_match_count / run.completed_count) * 100);
    const honest = Math.round((run.type_match_count / run.completed_count) * 100);
    items.push(["Accuracy", `${acc}%`]);
    items.push(["Type-honest", `${honest}%`]);
    if (run.unique_ip_count !== undefined && run.completed_count > 1) {
      const pool = Math.round((run.unique_ip_count / run.completed_count) * 100);
      items.push(["Pool", `${pool}%`]);
    }
  }

  return (
    <div
      data-testid="audit-header-row2"
      className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b px-3 py-2 text-xs"
    >
      {items.map(([k, v], i) => (
        <span key={k} className="flex items-center gap-1">
          {i > 0 && <span className="text-muted-foreground">·</span>}
          <span className="text-muted-foreground">{k}:</span>
          <span>{v}</span>
        </span>
      ))}
    </div>
  );
}

function formatDuration(startedAt: string, finishedAt: string | undefined, isRunning: boolean): string {
  const start = new Date(startedAt).getTime();
  const end = isRunning ? Date.now() : finishedAt ? new Date(finishedAt).getTime() : start;
  const totalSec = Math.max(0, Math.floor((end - start) / 1000));
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  if (h > 0) return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  return `${m}:${String(s).padStart(2, "0")}`;
}
