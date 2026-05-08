import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AuditResultsSheet } from "./AuditResultsSheet";
import type { RequestRow } from "@/types/audit";

function makeRow(overrides: Partial<RequestRow>): RequestRow {
  return {
    seq: 1, started_at: new Date().toISOString(), duration_ms: 100,
    is_datacenter: false, is_mobile: false, is_proxy: false, is_vpn: false,
    location_match: true, type_match: true, geo_source: "ipapi", ...overrides,
  };
}

describe("AuditResultsSheet", () => {
  it("renders counters", () => {
    const rows = [
      makeRow({ seq: 1, location_match: true, type_match: true }),
      makeRow({ seq: 2, location_match: false, type_match: true }),
      makeRow({ seq: 3, location_match: true, type_match: false }),
    ];
    render(<AuditResultsSheet rows={rows} total={10} expectedCity="Bucharest" />);
    expect(screen.getByText(/3 \/ 10/)).toBeInTheDocument();
    // 2 of 3 location matches = ~67% (Math.round(66.66...) = 67)
    expect(screen.getByText(/67%/)).toBeInTheDocument();
  });

  it("shows error rows in errors panel", () => {
    const rows = [
      makeRow({ seq: 1, error: "boom" }),
    ];
    render(<AuditResultsSheet rows={rows} total={1} />);
    expect(screen.getByText(/boom/)).toBeInTheDocument();
  });
});

function row(seq: number, ip: string, opts: Partial<RequestRow> = {}): RequestRow {
  return {
    seq,
    started_at: "2026-05-07T00:00:00Z",
    duration_ms: 100,
    observed_ip: ip,
    is_datacenter: false,
    is_mobile: false,
    is_proxy: false,
    is_vpn: false,
    location_match: true,
    type_match: true,
    geo_source: "ipapi",
    ...opts,
  };
}

describe("AuditResultsSheet — pool richness", () => {
  it("renders 100% green when every IP is unique (live mode)", () => {
    const rows = [row(1, "1.1.1.1"), row(2, "2.2.2.2"), row(3, "3.3.3.3")];
    render(<AuditResultsSheet rows={rows} total={3} sessionKey="session" />);
    const badge = screen.getByTestId("pool-richness");
    expect(badge).toHaveTextContent(/3 \/ 3/);
    expect(badge).toHaveTextContent(/100%/);
    expect(badge.className).toMatch(/green/i);
  });

  it("renders red < 50%", () => {
    const rows = [
      row(1, "1.1.1.1"),
      row(2, "1.1.1.1"),
      row(3, "1.1.1.1"),
    ];
    render(<AuditResultsSheet rows={rows} total={3} sessionKey="session" />);
    const badge = screen.getByTestId("pool-richness");
    expect(badge).toHaveTextContent(/1 \/ 3/);
    expect(badge.className).toMatch(/red/i);
  });

  it("renders amber at exactly 50%", () => {
    const rows = [row(1, "1.1.1.1"), row(2, "1.1.1.1"), row(3, "2.2.2.2"), row(4, "1.1.1.1")];
    render(<AuditResultsSheet rows={rows} total={4} sessionKey="session" />);
    const badge = screen.getByTestId("pool-richness");
    expect(badge).toHaveTextContent(/2 \/ 4/);
    expect(badge.className).toMatch(/amber|yellow/i);
  });

  it("uses uniqueIPCount prop post-finalization when rows are not present", () => {
    render(
      <AuditResultsSheet
        rows={[]}
        total={100}
        sessionKey="session"
        uniqueIPCount={87}
        completedCount={100}
      />,
    );
    expect(screen.getByTestId("pool-richness")).toHaveTextContent(/87 \/ 100/);
  });

  it("renders — when n=0 or n=1", () => {
    render(<AuditResultsSheet rows={[row(1, "1.1.1.1")]} total={1} />);
    expect(screen.getByTestId("pool-richness")).toHaveTextContent("—");
  });

  it("shows the rotated-but-duplicate caption when sessionKey was set", () => {
    const rows = [row(1, "1.1.1.1"), row(2, "1.1.1.1")];
    render(<AuditResultsSheet rows={rows} total={2} sessionKey="session" />);
    expect(screen.getByText(/provider returned duplicate exits/i)).toBeInTheDocument();
  });

  it("shows the no-rotation caption when sessionKey was unset", () => {
    const rows = [row(1, "1.1.1.1"), row(2, "1.1.1.1")];
    render(<AuditResultsSheet rows={rows} total={2} />);
    expect(screen.getByText(/no session rotation/i)).toBeInTheDocument();
  });

  it("hides the caption when richness is 100%", () => {
    const rows = [row(1, "1.1.1.1"), row(2, "2.2.2.2")];
    render(<AuditResultsSheet rows={rows} total={2} sessionKey="session" />);
    expect(screen.queryByText(/provider returned duplicate/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/no session rotation/i)).not.toBeInTheDocument();
  });
});
