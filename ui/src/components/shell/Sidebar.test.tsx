import { describe, it, expect } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } from "@tanstack/react-router";
import { Sidebar } from "./Sidebar";

async function setup(initial: string) {
  const rootRoute = createRootRoute({
    component: () => (
      <>
        <Sidebar />
        <Outlet />
      </>
    ),
  });
  const childRoutes = ["/", "/providers", "/targets", "/status-codes", "/live-traffic", "/audits", "/profiles", "/settings"].map(
    (p) =>
      createRoute({
        getParentRoute: () => rootRoute,
        path: p,
        component: () => <div>{p}</div>,
      })
  );
  const tree = rootRoute.addChildren(childRoutes);
  const router = createRouter({
    routeTree: tree,
    history: createMemoryHistory({ initialEntries: [initial] }),
  });
  await act(async () => {
    render(<RouterProvider router={router} />);
  });
}

describe("Sidebar", () => {
  it("renders all 8 nav items with the right labels", async () => {
    await setup("/");
    for (const label of [
      "Overview",
      "Providers",
      "Targets",
      "Status Codes",
      "Live Traffic",
      "Audits",
      "Profiles",
      "Settings",
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });

  it("marks 1 of them with .soon.", async () => {
    await setup("/");
    expect(screen.getAllByText(/soon/i)).toHaveLength(1);
  });
});
