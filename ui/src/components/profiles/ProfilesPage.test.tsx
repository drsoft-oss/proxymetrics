import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { ProfilesPage } from "./ProfilesPage";
import { queryKeys } from "@/lib/queryKeys";
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
    region: "",
    default_team: "",
    default_project: "",
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
    region: "",
    default_team: "",
    default_project: "",
    created_at: "x",
    updated_at: "y",
    requests: 8000,
    spend_usd: 11.3,
  },
];

function wrap(ui: ReactNode, seeded: ProfileWithUsage[] | null = rows) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  if (seeded) qc.setQueryData(queryKeys.profiles(), seeded);
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe("ProfilesPage", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify(rows), {
          status: 200,
          headers: { "content-type": "application/json" },
        })
      )
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("renders the page heading and subtitle", () => {
    wrap(<ProfilesPage />);
    expect(screen.getByRole("heading", { name: /^profiles$/i })).toBeInTheDocument();
    expect(screen.getByText(/auto-discovered/i)).toBeInTheDocument();
  });

  it("does not render an Add profile button", () => {
    wrap(<ProfilesPage />);
    expect(screen.queryByRole("button", { name: /add profile/i })).not.toBeInTheDocument();
  });

  it("renders rows from the seeded query cache", () => {
    wrap(<ProfilesPage />);
    expect(screen.getByText("BrightData EU")).toBeInTheDocument();
    expect(screen.getByText("Oxylabs")).toBeInTheDocument();
  });

  it("opens the detail drawer with the clicked row's label as title", async () => {
    wrap(<ProfilesPage />);
    await userEvent.click(screen.getByText("BrightData EU").closest("tr")!);
    // Drawer title is rendered inside the Radix Dialog portal — assertion finds the
    // second occurrence (one in the table cell, one in the drawer header).
    const matches = screen.getAllByText("BrightData EU");
    expect(matches.length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/identity/i)).toBeInTheDocument();
  });

  it("closes the drawer when the close button is clicked", async () => {
    wrap(<ProfilesPage />);
    await userEvent.click(screen.getByText("Oxylabs").closest("tr")!);
    expect(screen.getByText(/identity/i)).toBeInTheDocument();
    await userEvent.click(screen.getByLabelText(/close/i));
    expect(screen.queryByText(/identity/i)).not.toBeInTheDocument();
  });
});
