import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useStatusCodeDistribution } from "./useStatusCodeDistribution";

const sample = { items: [{ code: 200, class: "2xx", requests: 1, wasted_usd: 0, spend_usd: 1, top_provider_vendor: "v" }], from: "", to: "", source: "events" };

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useStatusCodeDistribution", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } }))
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("calls /api/v1/status-codes/distribution with filter params", async () => {
    const filter = { from: "2026-05-01T00:00:00Z", to: "2026-05-04T00:00:00Z", vendor: ["smartproxy"] };
    const { result } = renderHook(() => useStatusCodeDistribution(filter), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    const url = calls[0][0];
    expect(url).toContain("/api/v1/status-codes/distribution");
    expect(url).toContain("vendor=smartproxy");
    expect(url).toContain("from=2026-05-01T00%3A00%3A00Z");
  });
});
