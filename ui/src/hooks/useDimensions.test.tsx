import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useDimensions } from "./useDimensions";

const sample = {
  vendor: ["a", "b"],
  type: ["residential"],
  region: [],
  team: [],
  project: [],
  status_class: ["2xx", "3xx", "4xx", "5xx"],
  profile_id: [{ id: "p1", name: "P1" }],
};

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useDimensions", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } }))
    );
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("calls /api/v1/dimensions and returns the parsed body", async () => {
    const { result } = renderHook(() => useDimensions("2026-05-01T00:00:00Z", "2026-05-04T00:00:00Z"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.vendor).toEqual(["a", "b"]);
    expect(result.current.data?.profile_id[0].name).toBe("P1");

    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    expect(calls[0][0]).toContain("/api/v1/dimensions?from=2026-05-01T00%3A00%3A00Z&to=2026-05-04T00%3A00%3A00Z");
  });
});
