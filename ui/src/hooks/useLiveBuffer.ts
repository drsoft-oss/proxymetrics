import { useEffect, useMemo, useRef, useState } from "react";
import type { EventRow } from "@/types/api";

export type LiveAggregates = {
  reqPerSec: number;
  activeProviders: number;
  activeTargets: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  maxMs: number;
  statusMix: { class2xx: number; class3xx: number; class4xx: number; class5xx: number };
  throughputSeries: { t: number; reqPerSec: number }[]; // 1-second buckets, ~windowSec entries
  topProviders: { vendor: string; count: number }[];
  topTargets: { host: string; count: number }[];
};

const EMPTY_AGG: LiveAggregates = {
  reqPerSec: 0,
  activeProviders: 0,
  activeTargets: 0,
  p50Ms: 0,
  p95Ms: 0,
  p99Ms: 0,
  maxMs: 0,
  statusMix: { class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0 },
  throughputSeries: [],
  topProviders: [],
  topTargets: [],
};

export function useLiveBuffer(opts: { windowSec: number; paused: boolean }): {
  events: EventRow[];
  aggregates: LiveAggregates;
} {
  const { windowSec, paused } = opts;
  const [events, setEvents] = useState<EventRow[]>([]);
  const pausedRef = useRef(paused);
  pausedRef.current = paused;

  // Subscribe to SSE.
  useEffect(() => {
    const es = new EventSource("/api/v1/events/stream");
    const onEvent = (ev: MessageEvent) => {
      if (pausedRef.current) return;
      let parsed: EventRow;
      try {
        parsed = JSON.parse(ev.data as string) as EventRow;
      } catch {
        return;
      }
      setEvents((prev) => [parsed, ...prev]);
    };
    es.addEventListener("event", onEvent);
    return () => {
      es.removeEventListener("event", onEvent);
      es.close();
    };
  }, []);

  // Window expiry tick.
  useEffect(() => {
    const id = setInterval(() => {
      const cutoff = Date.now() - windowSec * 1000;
      setEvents((prev) => prev.filter((e) => new Date(e.ts).getTime() >= cutoff));
    }, 1000);
    return () => clearInterval(id);
  }, [windowSec]);

  const aggregates = useMemo<LiveAggregates>(() => {
    if (events.length === 0) return EMPTY_AGG;
    const cutoff = Date.now() - windowSec * 1000;
    const inWindow = events.filter((e) => new Date(e.ts).getTime() >= cutoff);
    if (inWindow.length === 0) return EMPTY_AGG;

    const providers = new Map<string, number>();
    const targets = new Map<string, number>();
    const mix = { class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0 };
    const lat: number[] = [];
    let maxMs = 0;

    for (const e of inWindow) {
      providers.set(e.vendor, (providers.get(e.vendor) ?? 0) + 1);
      if (e.target_host) targets.set(e.target_host, (targets.get(e.target_host) ?? 0) + 1);
      if (e.status_class === "2xx") mix.class2xx++;
      else if (e.status_class === "3xx") mix.class3xx++;
      else if (e.status_class === "4xx") mix.class4xx++;
      else if (e.status_class === "5xx") mix.class5xx++;
      lat.push(e.latency_ms);
      if (e.latency_ms > maxMs) maxMs = e.latency_ms;
    }
    lat.sort((a, b) => a - b);
    const pct = (p: number) => (lat.length === 0 ? 0 : lat[Math.min(lat.length - 1, Math.floor(p * lat.length))]);

    // 1-second buckets for throughput. Length is windowSec or windowSec+1 depending
    // on whether cutoff falls on a whole-second boundary. Including the partial
    // first bucket keeps the bar sum consistent with reqPerSec.
    const series: { t: number; reqPerSec: number }[] = [];
    const startSec = Math.floor(cutoff / 1000);
    const endSec = Math.floor(Date.now() / 1000);
    const bucket = new Map<number, number>();
    for (const e of inWindow) {
      const sec = Math.floor(new Date(e.ts).getTime() / 1000);
      if (sec > endSec) continue; // guard against future-timestamp events (clock skew)
      bucket.set(sec, (bucket.get(sec) ?? 0) + 1);
    }
    for (let s = startSec; s <= endSec; s++) {
      series.push({ t: s, reqPerSec: bucket.get(s) ?? 0 });
    }

    const topProviders = [...providers.entries()]
      .map(([vendor, count]) => ({ vendor, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5);
    const topTargets = [...targets.entries()]
      .map(([host, count]) => ({ host, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5);

    return {
      reqPerSec: inWindow.length / windowSec,
      activeProviders: providers.size,
      activeTargets: targets.size,
      p50Ms: pct(0.5),
      p95Ms: pct(0.95),
      p99Ms: pct(0.99),
      maxMs,
      statusMix: mix,
      throughputSeries: series,
      topProviders,
      topTargets,
    };
  }, [events, windowSec]);

  return { events, aggregates };
}
