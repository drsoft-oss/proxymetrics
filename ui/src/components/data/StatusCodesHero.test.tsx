import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusCodesHero } from "./StatusCodesHero";
import type { StatusCodeDistItem } from "@/types/api";

const items: StatusCodeDistItem[] = [
  { code: 200, class: "2xx", requests: 800, wasted_usd: 0,    spend_usd: 100, top_provider_vendor: "v" },
  { code: 429, class: "4xx", requests: 100, wasted_usd: 5.0,  spend_usd: 5.0, top_provider_vendor: "v" },
  { code: 500, class: "5xx", requests: 50,  wasted_usd: 2.50, spend_usd: 2.5, top_provider_vendor: "v" },
];

describe("StatusCodesHero", () => {
  it("computes 'if eliminated' total from non-2xx wasted", () => {
    render(<StatusCodesHero items={items} />);
    // 5.00 + 2.50 = 7.50
    expect(screen.getByText("$7.50")).toBeInTheDocument();
  });

  it("renders all 4 class tiles", () => {
    render(<StatusCodesHero items={items} />);
    for (const cls of ["2xx", "3xx", "4xx", "5xx"]) {
      expect(screen.getAllByText(cls).length).toBeGreaterThan(0);
    }
  });

  it("zero state still shows hero with $0.00", () => {
    render(<StatusCodesHero items={[]} />);
    expect(screen.getByText("$0.00")).toBeInTheDocument();
  });
});
