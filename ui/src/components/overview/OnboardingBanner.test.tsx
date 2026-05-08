import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OnboardingBanner } from "./OnboardingBanner";

describe("OnboardingBanner", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders when requestCount === 0 and not dismissed", () => {
    render(<OnboardingBanner requestCount={0} fingerprint="AB:CD" />);
    expect(screen.getByText(/Install the CA cert/i)).toBeInTheDocument();
    expect(screen.getByText(/Register a profile/i)).toBeInTheDocument();
  });

  it("hides when requestCount > 0", () => {
    const { container } = render(<OnboardingBanner requestCount={42} fingerprint="AB:CD" />);
    expect(container.firstChild).toBeNull();
  });

  it("dismiss persists to localStorage and hides", async () => {
    render(<OnboardingBanner requestCount={0} fingerprint="AB:CD" />);
    await userEvent.click(screen.getByLabelText(/Dismiss banner/i));
    expect(window.localStorage.getItem("proxymetrics:banner:dismissed")).toBe("true");
  });

  it("copy button writes to clipboard", async () => {
    const writeText = vi.fn();
    Object.assign(navigator, { clipboard: { writeText } });
    render(<OnboardingBanner requestCount={0} fingerprint="AB:CD" />);
    await userEvent.click(screen.getAllByLabelText(/Copy command/i)[0]);
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining("curl"));
  });
});
