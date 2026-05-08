import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSSE } from "./useSSE";

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

  addEventListener(type: string, fn: (ev: MessageEvent) => void) {
    (this.listeners[type] ||= []).push(fn);
  }

  emit(type: string, data: string) {
    for (const fn of this.listeners[type] ?? []) fn({ data } as MessageEvent);
  }

  close() {
    this.closed = true;
  }
}

afterEach(() => {
  MockEventSource.instances = [];
});

describe("useSSE", () => {
  it("opens an EventSource on mount and closes on unmount", () => {
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
    const { unmount } = renderHook(() => useSSE("/api/v1/events/stream", (s) => JSON.parse(s)));
    expect(MockEventSource.instances).toHaveLength(1);
    unmount();
    expect(MockEventSource.instances[0].closed).toBe(true);
  });

  it("collects events into a ring buffer trimmed to ringSize", () => {
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
    const { result } = renderHook(() =>
      useSSE<{ id: number }>(
        "/api/v1/events/stream",
        (s) => JSON.parse(s) as { id: number },
        3
      )
    );
    const es = MockEventSource.instances[0];
    act(() => {
      es.emit("event", JSON.stringify({ id: 1 }));
      es.emit("event", JSON.stringify({ id: 2 }));
      es.emit("event", JSON.stringify({ id: 3 }));
      es.emit("event", JSON.stringify({ id: 4 }));
    });
    expect(result.current.events.map((e) => e.id)).toEqual([4, 3, 2]);
  });

  it("status transitions from connecting → open on onopen", () => {
    vi.stubGlobal("EventSource", MockEventSource as unknown as typeof EventSource);
    const { result } = renderHook(() => useSSE("/api/v1/events/stream", (s) => s));
    expect(result.current.status).toBe("connecting");
    const es = MockEventSource.instances[0];
    act(() => {
      es.onopen?.();
    });
    expect(result.current.status).toBe("open");
  });
});
