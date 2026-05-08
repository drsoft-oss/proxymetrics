import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuditActiveRun } from "./AuditActiveRun";

class FakeES {
  static instances: FakeES[] = [];
  onmessage?: (e: { data: string }) => void;
  closed = false;
  constructor() { FakeES.instances.push(this); }
  close() { this.closed = true; }
}

beforeEach(() => {
  FakeES.instances = [];
  vi.stubGlobal("EventSource", FakeES as any);
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({
      run: {
        id: "r1", started_at: new Date().toISOString(), status: "running",
        proxy_url: "x", expected_country: "RO", expected_lat: 44.43, expected_lon: 26.10,
        expected_type: "residential", check_level: "country", request_count: 5,
        completed_count: 0, location_match_count: 0, type_match_count: 0, error_count: 0,
        fallback_used: false,
      },
      requests: [],
    }),
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("AuditActiveRun", () => {
  it("subscribes to SSE for the given run", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <AuditActiveRun runId="r1" onCancelled={() => {}} />
      </QueryClientProvider>
    );
    await waitFor(() => expect(FakeES.instances.length).toBe(1));
    expect(screen.getByText(/Run/i)).toBeInTheDocument();
  });
});
