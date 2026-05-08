import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { StatusMixBar } from "./StatusMixBar";
import { Sparkline } from "./Sparkline";

describe("StatusMixBar", () => {
  it("renders 4 segments with proportional widths", () => {
    const { container } = render(
      <StatusMixBar mix={{ class2xx: 70, class3xx: 0, class4xx: 20, class5xx: 10 }} />
    );
    const segs = container.querySelectorAll("[data-testid='mix-seg']");
    expect(segs.length).toBe(4);
    expect(segs[0].getAttribute("data-class")).toBe("2xx");
    expect((segs[0] as HTMLElement).style.width).toBe("70%");
    expect((segs[2] as HTMLElement).style.width).toBe("20%");
  });

  it("returns dash when total is 0", () => {
    const { getByText } = render(
      <StatusMixBar mix={{ class2xx: 0, class3xx: 0, class4xx: 0, class5xx: 0 }} />
    );
    expect(getByText("—")).toBeInTheDocument();
  });
});

describe("Sparkline", () => {
  it("renders an svg path for non-empty data", () => {
    const { container } = render(<Sparkline data={[1, 2, 3, 5, 8]} />);
    expect(container.querySelector("svg")).toBeTruthy();
    expect(container.querySelector("path")).toBeTruthy();
  });

  it("renders a dash for empty data", () => {
    const { getByText } = render(<Sparkline data={[]} />);
    expect(getByText("—")).toBeInTheDocument();
  });

  it("renders a dash when all values are zero", () => {
    const { getByText } = render(<Sparkline data={[0, 0, 0]} />);
    expect(getByText("—")).toBeInTheDocument();
  });
});
