import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Combobox } from "./combobox";

describe("Combobox", () => {
  it("renders the current value on the trigger", () => {
    render(
      <Combobox
        value="brightdata"
        onChange={() => {}}
        options={["brightdata", "databay"]}
        ariaLabel="provider"
      />,
    );
    expect(
      screen.getByRole("combobox", { name: "provider" }),
    ).toHaveTextContent("brightdata");
  });

  it("filters options by typed query and commits a click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Combobox
        value=""
        onChange={onChange}
        options={["brightdata", "databay", "oxylabs"]}
        ariaLabel="provider"
      />,
    );
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const input = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(input, "data");
    // "databay" matches "data"; "brightdata" also matches via "data".
    expect(screen.getByText("databay")).toBeInTheDocument();
    await user.click(screen.getByText("databay"));
    expect(onChange).toHaveBeenCalledWith("databay");
  });

  it("with allowCustom shows a Create-row for unmatched queries and commits it", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Combobox
        value=""
        onChange={onChange}
        options={["brightdata"]}
        allowCustom
        ariaLabel="provider"
      />,
    );
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const input = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(input, "custom-pool");
    const createRow = await screen.findByText(/create.*custom-pool/i);
    await user.click(createRow);
    expect(onChange).toHaveBeenCalledWith("custom-pool");
  });

  it("without allowCustom does not commit unmatched typed values", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Combobox
        value=""
        onChange={onChange}
        options={["brightdata"]}
        ariaLabel="provider"
      />,
    );
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const input = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(input, "nope{Enter}");
    expect(onChange).not.toHaveBeenCalled();
  });
});
