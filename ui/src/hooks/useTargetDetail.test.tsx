import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useTargetDetail } from "./useTargetDetail";

const sample = { target_host: "x.com", by_provider: [], by_status_class: [], from: "x", to: "y", source: "events" };

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useTargetDetail", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } }))
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("does not fetch when host is null", () => {
    renderHook(() => useTargetDetail(null, { from: "x", to: "y" }), { wrapper: wrapper() });
    expect((globalThis.fetch as unknown as { mock: { calls: unknown[] } }).mock.calls.length).toBe(0);
  });

  it("fetches /api/v1/targets/{host} when host is set", async () => {
    const { result } = renderHook(() => useTargetDetail("x.com", { from: "x", to: "y" }), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    expect(calls[0][0]).toContain("/api/v1/targets/x.com");
  });
});
