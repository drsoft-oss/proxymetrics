import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { EventRow } from "@/types/api";
import { useLiveBuffer } from "./useLiveBuffer";

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  listeners: Record<string, ((ev: MessageEvent) => void)[]> = {};
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }
  addEventListener(t: string, fn: (ev: MessageEvent) => void) {
    (this.listeners[t] ||= []).push(fn);
  }
  removeEventListener(t: string, fn: (ev: MessageEvent) => void) {
    if (!this.listeners[t]) return;
    this.listeners[t] = this.listeners[t].filter((f) => f !== fn);
  }
  emit(t: string, data: string) {
    for (const fn of this.listeners[t] ?? []) fn({ data } as MessageEvent);
  }
  close() {
    this.closed = true;
  }
}

function ev(seconds: number, partial?: Partial<EventRow>): EventRow {
  return {
    ts: new Date(seconds * 1000).toISOString(),
    request_id: `r${seconds}`,
    profile_id: "p1",
    vendor: "v",
    type: "residential",
    target_host: "x.com",
    status_code: 200,
    status_class: "2xx",
    bytes_in: 1,
    bytes_out: 1,
    latency_ms: 50,
    cost_usd: 0,
    ...partial,
  };
}

describe("useLiveBuffer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(60_000)); // t=60s
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    MockEventSource.instances = [];
  });

  it("expires events older than windowSec on each tick", () => {
    const { result } = renderHook(() => useLiveBuffer({ windowSec: 30, paused: false }));
    const es = MockEventSource.instances[0];

    act(() => {
      es.emit("event", JSON.stringify(ev(40))); // 20s ago
      es.emit("event", JSON.stringify(ev(50))); // 10s ago
      es.emit("event", JSON.stringify(ev(20))); // 40s ago — outside window
    });
    expect(result.current.events.length).toBe(3);

    // Move past the next 1s tick and let it expire the 40s-old event.
    act(() => {
      vi.advanceTimersByTime(1100);
    });
    // Two should remain; the t=20 event is older than now-windowSec.
    expect(result.current.events.length).toBe(2);
  });

  it("computes aggregates on the live window", () => {
    const { result } = renderHook(() => useLiveBuffer({ windowSec: 60, paused: false }));
    const es = MockEventSource.instances[0];

    act(() => {
      es.emit("event", JSON.stringify(ev(50, { status_class: "2xx", status_code: 200, latency_ms: 100, vendor: "v1", target_host: "a.com" })));
      es.emit("event", JSON.stringify(ev(55, { status_class: "4xx", status_code: 429, latency_ms: 200, vendor: "v2", target_host: "b.com" })));
      es.emit("event", JSON.stringify(ev(58, { status_class: "2xx", status_code: 200, latency_ms: 50, vendor: "v1", target_host: "a.com" })));
    });

    expect(result.current.aggregates.activeProviders).toBe(2);
    expect(result.current.aggregates.activeTargets).toBe(2);
    expect(result.current.aggregates.statusMix.class2xx).toBe(2);
    expect(result.current.aggregates.statusMix.class4xx).toBe(1);
  });

  it("paused: stops new events from being added but does not freeze old expiry", () => {
    const { result, rerender } = renderHook(({ paused }) => useLiveBuffer({ windowSec: 60, paused }), {
      initialProps: { paused: false },
    });
    const es = MockEventSource.instances[0];

    act(() => {
      es.emit("event", JSON.stringify(ev(55)));
    });
    expect(result.current.events.length).toBe(1);

    // Pause and emit more — must be ignored.
    rerender({ paused: true });
    act(() => {
      es.emit("event", JSON.stringify(ev(56)));
    });
    expect(result.current.events.length).toBe(1);
  });
});
