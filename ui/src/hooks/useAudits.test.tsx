import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAudits, useAuditDetail } from "./useAudits";

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

describe("useAudits", () => {
  it("fetches the list", async () => {
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => [{ id: "r1", status: "completed" }],
    });
    const { result } = renderHook(() => useAudits(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.[0].id).toBe("r1");
  });
});

describe("useAuditDetail", () => {
  it("fetches detail when id is provided", async () => {
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ run: { id: "r1", status: "completed" }, requests: [] }),
    });
    const { result } = renderHook(() => useAuditDetail("r1"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.run.id).toBe("r1");
  });

  it("does not fetch when id is null", () => {
    (globalThis.fetch as any).mockResolvedValue({ ok: true, json: async () => ({}) });
    renderHook(() => useAuditDetail(null), { wrapper: wrapper() });
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });
});
