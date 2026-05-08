import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useStatusCodeDetail } from "./useStatusCodeDetail";

const sample = { code: 429, class: "4xx", requests: 5, wasted_usd: 1, spend_usd: 1, top_providers: [], top_targets: [], from: "", to: "" };

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useStatusCodeDetail", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } }))
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("does NOT fetch when code is null (enabled gate)", () => {
    const filter = { from: "x", to: "y" };
    renderHook(() => useStatusCodeDetail(null, filter), { wrapper: wrapper() });
    const calls = (globalThis.fetch as unknown as { mock: { calls: unknown[] } }).mock.calls;
    expect(calls.length).toBe(0);
  });

  it("fetches /api/v1/status-codes/{code} when code is set", async () => {
    const filter = { from: "x", to: "y" };
    const { result } = renderHook(() => useStatusCodeDetail(429, filter), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    expect(calls[0][0]).toContain("/api/v1/status-codes/429");
  });
});
