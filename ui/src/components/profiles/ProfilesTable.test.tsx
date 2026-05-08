import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ProfilesTable } from "./ProfilesTable";
import type { ProfileWithUsage } from "@/types/profile";

const rows: ProfileWithUsage[] = [
  {
    id: "p1",
    label: "BrightData EU",
    vendor: "brightdata",
    type: "residential",
    upstream_url: "http://u1",
    currency: "USD",
    price_per_gb: 8.5,
    price_per_gb_overage: null,
    included_gb: null,
    region: "EU",
    created_at: "x",
    updated_at: "y",
    requests: 12000,
    spend_usd: 42.1,
  },
  {
    id: "p2",
    label: "Oxylabs",
    vendor: "oxylabs",
    type: "isp",
    upstream_url: "http://u2",
    currency: "USD",
    price_per_gb: 3.0,
    price_per_gb_overage: null,
    included_gb: null,
    created_at: "x",
    updated_at: "y",
    requests: 8000,
    spend_usd: 11.3,
  },
];

describe("ProfilesTable", () => {
  it("renders rows with id, label, vendor, type, $/GB, requests, spend", () => {
    render(<ProfilesTable rows={rows} loading={false} onRowClick={() => {}} />);
    expect(screen.getByText("p1")).toBeInTheDocument();
    expect(screen.getByText("BrightData EU")).toBeInTheDocument();
    expect(screen.getByText("brightdata")).toBeInTheDocument();
    expect(screen.getByText("residential")).toBeInTheDocument();
    expect(screen.getByText(/8\.50/)).toBeInTheDocument();
  });

  it("invokes onRowClick with the id when a row is clicked", () => {
    const cb = vi.fn();
    render(<ProfilesTable rows={rows} loading={false} onRowClick={cb} />);
    fireEvent.click(screen.getByText("BrightData EU").closest("tr")!);
    expect(cb).toHaveBeenCalledWith("p1");
  });

  it("renders an empty-state message when there are no rows", () => {
    render(<ProfilesTable rows={[]} loading={false} onRowClick={() => {}} />);
    expect(screen.getByText(/no profiles yet/i)).toBeInTheDocument();
  });
});
