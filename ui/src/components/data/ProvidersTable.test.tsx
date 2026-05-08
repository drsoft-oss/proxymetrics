import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProvidersTable } from "./ProvidersTable";
import type { ProviderItem } from "@/types/api";

const rows: ProviderItem[] = [
  {
    profile_id: "p1", label: "Bright Data", vendor: "brightdata", type: "residential",
    request_count: 100, bytes_total: 1, success_rate: 0.9, spend_usd: 10, wasted_usd: 1,
    last_seen: new Date().toISOString(),
    status_mix: { class2xx: 90, class3xx: 0, class4xx: 7, class5xx: 3 },
    requests_by_hour: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24],
  },
];

describe("ProvidersTable", () => {
  it("renders the provider name and stack-bar/sparkline cells", () => {
    render(<ProvidersTable rows={rows} onRowClick={() => {}} />);
    expect(screen.getByText("brightdata")).toBeInTheDocument();
    expect(screen.getAllByTestId("mix-seg").length).toBe(4);
    // 100 requests; check the value column rendered.
    expect(screen.getByText("100")).toBeInTheDocument();
  });

  it("fires onRowClick with the row", async () => {
    const cb = vi.fn();
    render(<ProvidersTable rows={rows} onRowClick={cb} />);
    await userEvent.click(screen.getByText("brightdata"));
    expect(cb).toHaveBeenCalledWith(rows[0]);
  });
});
