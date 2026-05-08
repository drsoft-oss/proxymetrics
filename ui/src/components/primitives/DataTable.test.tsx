import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable, type Column } from "./DataTable";

type Row = { id: string; name: string; value: number; group: "a" | "b" };

const cols: Column<Row>[] = [
  { key: "name", header: "Name", render: (r) => r.name, sortable: true, sortValue: (r) => r.name },
  { key: "value", header: "Value", render: (r) => String(r.value), sortable: true, align: "right", sortValue: (r) => r.value },
];

const rows: Row[] = [
  { id: "1", name: "alpha", value: 30, group: "a" },
  { id: "2", name: "beta",  value: 10, group: "a" },
  { id: "3", name: "gamma", value: 20, group: "b" },
];

describe("DataTable", () => {
  it("renders headers and rows", () => {
    render(<DataTable columns={cols} rows={rows} />);
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
  });

  it("loading state renders 5 skeleton rows", () => {
    const { container } = render(<DataTable columns={cols} rows={[]} isLoading />);
    expect(container.querySelectorAll("[data-testid='dt-skeleton-row']")).toHaveLength(5);
  });

  it("error state renders Retry that calls onRetry", async () => {
    const onRetry = vi.fn();
    render(<DataTable columns={cols} rows={[]} error={new Error("boom")} onRetry={onRetry} />);
    await userEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("empty state renders the empty message", () => {
    render(<DataTable columns={cols} rows={[]} emptyMessage="Nothing here" />);
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("clicking a sortable header toggles direction", async () => {
    render(<DataTable columns={cols} rows={rows} sortBy={{ key: "value", dir: "desc" }} />);
    const valueHeader = screen.getByRole("button", { name: /value/i });
    // First column rendered values should currently be desc (30, 20, 10).
    const cells = screen.getAllByRole("cell").map((c) => c.textContent ?? "");
    // value cells appear at indices 1, 3, 5 (alternating with name cells).
    expect(cells[1]).toBe("30");
    expect(cells[5]).toBe("10");
    await userEvent.click(valueHeader);
    const cells2 = screen.getAllByRole("cell").map((c) => c.textContent ?? "");
    expect(cells2[1]).toBe("10");
    expect(cells2[5]).toBe("30");
  });

  it("onRowClick fires with the row", async () => {
    const onRowClick = vi.fn();
    render(<DataTable columns={cols} rows={rows} onRowClick={onRowClick} />);
    await userEvent.click(screen.getByText("alpha"));
    expect(onRowClick).toHaveBeenCalledWith(rows[0]);
  });

  it("groupBy renders a group header per cluster", () => {
    render(<DataTable columns={cols} rows={rows} groupBy={(r) => r.group} />);
    expect(screen.getByText("a")).toBeInTheDocument();
    expect(screen.getByText("b")).toBeInTheDocument();
  });

  it("paginates at pageSize and shows pager", () => {
    const big: Row[] = Array.from({ length: 30 }, (_, i) => ({ id: String(i), name: `n${i}`, value: i, group: "a" }));
    render(<DataTable columns={cols} rows={big} pageSize={10} />);
    // Only 10 rendered (plus header skeleton avoidance — count "row"s).
    expect(screen.getAllByTestId("dt-row")).toHaveLength(10);
    expect(screen.getByText(/page 1/i)).toBeInTheDocument();
    expect(screen.getByText(/30 total/i)).toBeInTheDocument();
  });
});
