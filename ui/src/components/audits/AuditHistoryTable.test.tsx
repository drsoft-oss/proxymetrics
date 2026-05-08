import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AuditHistoryTable } from "./AuditHistoryTable";
import type { RunSummary } from "@/types/audit";

function makeRun(overrides: Partial<RunSummary>): RunSummary {
  return {
    id: "r1", started_at: new Date().toISOString(), status: "completed",
    proxy_url: "x", expected_country: "RO", expected_lat: 44.43, expected_lon: 26.10,
    expected_type: "residential", check_level: "country", request_count: 100,
    completed_count: 100, location_match_count: 90, type_match_count: 100, error_count: 0,
    fallback_used: false, ...overrides,
  };
}

describe("AuditHistoryTable — pool column", () => {
  function summary(over: Partial<import("@/types/audit").RunSummary>): import("@/types/audit").RunSummary {
    return {
      id: "r1",
      started_at: "2026-05-07T00:00:00Z",
      status: "completed",
      proxy_url: "x",
      expected_lat: 0,
      expected_lon: 0,
      expected_type: "residential",
      check_level: "country",
      request_count: 100,
      completed_count: 100,
      location_match_count: 100,
      type_match_count: 100,
      error_count: 0,
      fallback_used: false,
      ...over,
    };
  }

  it("renders pct and color for healthy pool", () => {
    render(
      <AuditHistoryTable
        rows={[summary({ id: "rh", unique_ip_count: 95, completed_count: 100 })]}
        onRowClick={() => {}}
      />,
    );
    const cell = screen.getByTestId("pool-rh");
    expect(cell).toHaveTextContent("95%");
    expect(cell.className).toMatch(/green/i);
  });

  it("renders red for sticky pool", () => {
    render(
      <AuditHistoryTable
        rows={[summary({ id: "rs", unique_ip_count: 1, completed_count: 100 })]}
        onRowClick={() => {}}
      />,
    );
    const cell = screen.getByTestId("pool-rs");
    expect(cell).toHaveTextContent("1%");
    expect(cell.className).toMatch(/red/i);
  });

  it("renders — for old rows with no unique_ip_count", () => {
    render(
      <AuditHistoryTable
        rows={[summary({ id: "ro" })]} // no unique_ip_count
        onRowClick={() => {}}
      />,
    );
    expect(screen.getByTestId("pool-ro")).toHaveTextContent("—");
  });
});

describe("AuditHistoryTable", () => {
  it("renders rows", () => {
    render(<AuditHistoryTable rows={[makeRun({ id: "a" }), makeRun({ id: "b", status: "cancelled" })]} onRowClick={() => {}} />);
    expect(screen.getByText("a")).toBeInTheDocument();
    expect(screen.getByText("b")).toBeInTheDocument();
    expect(screen.getByText(/cancelled/i)).toBeInTheDocument();
  });

  it("invokes onRowClick", () => {
    const cb = vi.fn();
    render(<AuditHistoryTable rows={[makeRun({ id: "a" })]} onRowClick={cb} />);
    fireEvent.click(screen.getByRole("row", { name: /^a$/i }));
    expect(cb).toHaveBeenCalledWith("a");
  });
});
