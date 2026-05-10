import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CaptchaByVendorChart } from "./CaptchaByVendorChart";

describe("CaptchaByVendorChart", () => {
  it("shows the empty placeholder when no rows", () => {
    render(<CaptchaByVendorChart rows={[]} />);
    expect(screen.getByText(/No captcha hits/)).toBeInTheDocument();
  });

  it("renders the chart container when rows present", () => {
    render(
      <CaptchaByVendorChart
        rows={[{ vendor: "brightdata", type: "residential", total: 3, by_kind: { recaptcha: 3 } }]}
      />,
    );
    expect(screen.getByText(/Captcha hits by vendor and type/)).toBeInTheDocument();
  });
});
