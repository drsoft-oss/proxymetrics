import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useProfiles } from "./useProfiles";

const sample = [
  {
    id: "default", label: "Default", vendor: "custom", type: "residential",
    upstream_url: "http://placeholder.invalid:1/", currency: "USD",
    created_at: "2026-04-15T00:00:00Z", updated_at: "2026-04-15T00:00:00Z",
    requests: 0, spend_usd: 0,
  },
];

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useProfiles", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } })
      )
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("calls /api/v1/profiles?with=usage and returns ProfileWithUsage[]", async () => {
    const { result } = renderHook(() => useProfiles(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    expect(calls[0][0]).toContain("/api/v1/profiles");
    expect(calls[0][0]).toContain("with=usage");
    expect(result.current.data).toEqual(sample);
  });
});
