import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KpiTile } from "./KpiTile";

describe("KpiTile", () => {
  it("renders label and value", () => {
    render(<KpiTile label="Spend this cycle" value="$1,847.32" />);
    expect(screen.getByText("Spend this cycle")).toBeInTheDocument();
    expect(screen.getByText("$1,847.32")).toBeInTheDocument();
  });

  it("hero variant applies destructive border + value color", () => {
    const { container } = render(
      <KpiTile label="$ Wasted" value="$312.81" variant="hero" />
    );
    const card = container.querySelector('[data-hero="true"]');
    expect(card).toBeTruthy();
  });

  it("renders sub line when provided", () => {
    render(<KpiTile label="$ Wasted" value="$0.00" sub="0% of spend" variant="hero" />);
    expect(screen.getByText("0% of spend")).toBeInTheDocument();
  });
});
