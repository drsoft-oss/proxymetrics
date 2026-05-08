import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuditDetailView } from "./AuditDetailView";

function makeRun(overrides: Record<string, unknown> = {}) {
  return {
    id: "r1",
    started_at: new Date().toISOString(),
    status: "completed",
    proxy_url: "http://proxy:8080",
    expected_country: "RO",
    expected_lat: 44.43,
    expected_lon: 26.1,
    expected_type: "residential",
    check_level: "country",
    request_count: 4,
    completed_count: 4,
    location_match_count: 4,
    type_match_count: 4,
    error_count: 0,
    fallback_used: false,
    ...overrides,
  };
}

function makeRequest(seq: number, ip: string) {
  return {
    seq,
    started_at: new Date().toISOString(),
    duration_ms: 100,
    observed_ip: ip,
    is_datacenter: false,
    is_mobile: false,
    is_proxy: false,
    is_vpn: false,
    location_match: true,
    type_match: true,
    geo_source: "ipapi",
  };
}

function wrapper(fetchResponse: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({ ok: true, json: async () => fetchResponse }),
  );
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  // AuditMap uses Leaflet which needs a DOM. Stub it out so tests stay fast.
  vi.mock("./AuditMap", () => ({ AuditMap: () => <div data-testid="map-stub" /> }));
});

afterEach(() => vi.unstubAllGlobals());

describe("AuditDetailView — pool richness", () => {
  it("shows rotated-but-duplicate caption when session_key is set and pool richness < 100%", async () => {
    // 2 unique IPs out of 4 = 50% (amber) with session_key set → rotated caption
    const response = {
      run: makeRun({ session_key: "mysession", unique_ip_count: 2, completed_count: 4 }),
      requests: [
        makeRequest(1, "1.1.1.1"),
        makeRequest(2, "1.1.1.1"),
        makeRequest(3, "2.2.2.2"),
        makeRequest(4, "1.1.1.1"),
      ],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() =>
      expect(screen.getByText(/provider returned duplicate exits/i)).toBeInTheDocument(),
    );
  });

  it("shows no-rotation caption when session_key is absent and pool richness < 100%", async () => {
    const response = {
      run: makeRun({ session_key: undefined, unique_ip_count: undefined, completed_count: 4 }),
      requests: [
        makeRequest(1, "1.1.1.1"),
        makeRequest(2, "1.1.1.1"),
        makeRequest(3, "2.2.2.2"),
        makeRequest(4, "1.1.1.1"),
      ],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() =>
      expect(screen.getByText(/no session rotation/i)).toBeInTheDocument(),
    );
  });
});

describe("AuditDetailView — header", () => {
  it("renders setup fields: proxy, location, type, provider, session", async () => {
    const response = {
      run: makeRun({
        proxy_url: "http://user:***@host:8080",
        expected_country: "RO",
        expected_state: "Bucharest",
        expected_city: "Bucharest",
        expected_type: "residential",
        provider: "acme",
        session_key: "sid",
      }),
      requests: [],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() => expect(screen.getByTestId("audit-header-row2")).toBeInTheDocument());
    const row = screen.getByTestId("audit-header-row2");
    expect(row).toHaveTextContent("http://user:***@host:8080");
    expect(row).toHaveTextContent("RO / Bucharest / Bucharest");
    expect(row).toHaveTextContent("residential");
    expect(row).toHaveTextContent("acme");
    expect(row).toHaveTextContent("rotating (sid)");
  });

  it("running run: shows X/Y checks, errors red, started + duration, no finished, no outcomes when 0 completed", async () => {
    const response = {
      run: makeRun({
        status: "running",
        started_at: new Date(Date.now() - 2000).toISOString(),
        finished_at: undefined,
        request_count: 100,
        completed_count: 0,
        error_count: 3,
        location_match_count: 0,
        type_match_count: 0,
        unique_ip_count: undefined,
      }),
      requests: [],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() => expect(screen.getByTestId("audit-header-row2")).toBeInTheDocument());
    const row = screen.getByTestId("audit-header-row2");
    expect(row).toHaveTextContent("Checks:");
    expect(row).toHaveTextContent("3 / 100");
    expect(screen.getByTestId("audit-header-errors")).toHaveTextContent("3");
    expect(screen.getByTestId("audit-header-errors")).toHaveClass("text-red-500");
    expect(row).toHaveTextContent("Started:");
    expect(row).toHaveTextContent("static");
    expect(row).toHaveTextContent("Duration:");
    expect(row).not.toHaveTextContent("Finished:");
    expect(row).not.toHaveTextContent("Accuracy:");
    expect(row).not.toHaveTextContent("Type-honest:");
    expect(row).not.toHaveTextContent("Pool:");
  });

  it("completed run: shows total checks, no errors when 0, finished, accuracy/type-honest/pool when completed > 0", async () => {
    const response = {
      run: makeRun({
        status: "completed",
        started_at: new Date(Date.now() - 60000).toISOString(),
        finished_at: new Date().toISOString(),
        request_count: 100,
        completed_count: 90,
        error_count: 0,
        location_match_count: 85,
        type_match_count: 80,
        unique_ip_count: 82,
      }),
      requests: [],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() => expect(screen.getByTestId("audit-header-row2")).toBeInTheDocument());
    const row = screen.getByTestId("audit-header-row2");
    expect(row).toHaveTextContent("Checks:");
    expect(row).toHaveTextContent("100");
    expect(row).not.toHaveTextContent("Errors:");
    expect(row).toHaveTextContent("Finished:");
    expect(row).toHaveTextContent("Accuracy:");
    expect(row).toHaveTextContent("94%");
    expect(row).toHaveTextContent("Type-honest:");
    expect(row).toHaveTextContent("89%");
    expect(row).toHaveTextContent("Pool:");
    expect(row).toHaveTextContent("91%");
  });

  it("hides Provider when run.provider is undefined", async () => {
    const response = {
      run: makeRun({ provider: undefined }),
      requests: [],
    };
    render(<AuditDetailView runId="r1" onReRun={() => {}} />, { wrapper: wrapper(response) });
    await waitFor(() => expect(screen.getByTestId("audit-header-row2")).toBeInTheDocument());
    const row = screen.getByTestId("audit-header-row2");
    expect(row).not.toHaveTextContent("Provider:");
  });
});
