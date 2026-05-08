import { useEffect, useState } from "react";

export type SSEStatus = "connecting" | "open" | "closed";

export function useSSE<T>(
  url: string,
  parser: (data: string) => T,
  ringSize = 50
): { events: T[]; status: SSEStatus } {
  const [events, setEvents] = useState<T[]>([]);
  const [status, setStatus] = useState<SSEStatus>("connecting");

  useEffect(() => {
    const es = new EventSource(url);
    es.onopen = () => setStatus("open");
    es.onerror = () => setStatus("closed");
    es.addEventListener("event", (ev) => {
      let parsed: T;
      try {
        parsed = parser((ev as MessageEvent).data as string);
      } catch {
        return;
      }
      setEvents((prev) => {
        const next = [parsed, ...prev];
        if (next.length > ringSize) next.length = ringSize;
        return next;
      });
    });
    return () => es.close();
  }, [url, parser, ringSize]);

  return { events, status };
}
