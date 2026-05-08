import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DrillSheet, DrillSection } from "./DrillSheet";

describe("DrillSheet", () => {
  it("renders title, subtitle, and children when open", () => {
    render(
      <DrillSheet open onClose={() => {}} title="Provider X" subtitle="2 targets">
        <DrillSection title="KPIs"><span>4-up</span></DrillSection>
      </DrillSheet>
    );
    expect(screen.getByText("Provider X")).toBeInTheDocument();
    expect(screen.getByText("2 targets")).toBeInTheDocument();
    expect(screen.getByText("4-up")).toBeInTheDocument();
  });

  it("calls onClose when X button is clicked", async () => {
    const onClose = vi.fn();
    render(
      <DrillSheet open onClose={onClose} title="X">
        <div>body</div>
      </DrillSheet>
    );
    await userEvent.click(screen.getByLabelText(/close/i));
    expect(onClose).toHaveBeenCalled();
  });

  it("does not render content when open=false", () => {
    render(
      <DrillSheet open={false} onClose={() => {}} title="Hidden">
        <div>body</div>
      </DrillSheet>
    );
    expect(screen.queryByText("Hidden")).not.toBeInTheDocument();
  });

  it("DrillSection renders its title and children", () => {
    render(<DrillSection title="Trend"><div>chart</div></DrillSection>);
    expect(screen.getByText("Trend")).toBeInTheDocument();
    expect(screen.getByText("chart")).toBeInTheDocument();
  });
});
