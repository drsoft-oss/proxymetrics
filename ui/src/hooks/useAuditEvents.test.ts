import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useAuditEvents } from "./useAuditEvents";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  url: string;
  onmessage?: (ev: { data: string }) => void;
  onerror?: () => void;
  closed = false;
  constructor(url: string) {
    this.url = url;
    FakeEventSource.instances.push(this);
  }
  close() { this.closed = true; }
}

beforeEach(() => {
  FakeEventSource.instances = [];
  vi.stubGlobal("EventSource", FakeEventSource as any);
});
afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useAuditEvents", () => {
  it("records request events and stops on finished", () => {
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useAuditEvents(id), {
      initialProps: { id: "r1" as string | null },
    });
    const es = FakeEventSource.instances[0];
    act(() => {
      es.onmessage?.({ data: JSON.stringify({ type: "request", seq: 1, location_match: true, type_match: true, error: "" }) });
    });
    expect(result.current.requests.length).toBe(1);
    act(() => {
      es.onmessage?.({ data: JSON.stringify({ type: "finished", status: "completed" }) });
    });
    expect(result.current.finished).toBe(true);
    expect(result.current.status).toBe("completed");
    expect(es.closed).toBe(true);

    rerender({ id: null });
  });

  it("does not connect when id is null", () => {
    renderHook(() => useAuditEvents(null));
    expect(FakeEventSource.instances.length).toBe(0);
  });
});
