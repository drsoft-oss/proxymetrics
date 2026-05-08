import { useSSE } from "@/hooks/useSSE";
import { Badge } from "@/components/ui/badge";
import type { EventRow } from "@/types/api";
import { formatTimeShort } from "@/lib/format";

const STATUS_VARIANT: Record<string, "success" | "destructive" | "warning" | "secondary"> = {
  "2xx": "success",
  "3xx": "secondary",
  "4xx": "destructive",
  "5xx": "warning",
  timeout: "warning",
  connection_error: "destructive",
};

export function LiveActivityRail() {
  const { events, status } = useSSE<EventRow>("/api/v1/events/stream", (s) => JSON.parse(s) as EventRow);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-border p-3 text-sm font-medium">
        <span
          className={`h-2 w-2 rounded-full ${
            status === "open" ? "bg-success shadow-[0_0_6px_hsl(var(--success))]" : "bg-muted"
          }`}
        />
        Live activity
        <span className="ml-auto text-[10px] uppercase text-muted-foreground">{status}</span>
      </div>
      <div className="flex-1 overflow-y-auto">
        {events.length === 0 ? (
          <div className="p-3 text-xs text-muted-foreground">
            Waiting for events…
          </div>
        ) : (
          events.map((e) => (
            <div key={e.request_id} className="grid grid-cols-[60px_1fr_auto] items-center gap-2 border-b border-border p-2 text-[11px]">
              <span className="text-muted-foreground">{formatTimeShort(e.ts)}</span>
              <span className="truncate" title={e.target_host}>{e.target_host ?? "—"}</span>
              <Badge variant={STATUS_VARIANT[e.status_class] ?? "secondary"} className="text-[10px]">
                {e.status_code || e.status_class}
              </Badge>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
