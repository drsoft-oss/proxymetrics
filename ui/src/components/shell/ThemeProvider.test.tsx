import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider, useTheme } from "./ThemeProvider";

function Probe() {
  const { theme, toggle } = useTheme();
  return (
    <div>
      <span data-testid="t">{theme}</span>
      <button onClick={toggle}>toggle</button>
    </div>
  );
}

describe("ThemeProvider", () => {
  beforeEach(() => window.localStorage.clear());

  it("defaults to dark", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    expect(screen.getByTestId("t").textContent).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("toggle flips theme and persists to localStorage", async () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    await userEvent.click(screen.getByText("toggle"));
    expect(screen.getByTestId("t").textContent).toBe("light");
    expect(window.localStorage.getItem("proxymetrics:theme")).toBe("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("hydrates from localStorage on mount", () => {
    window.localStorage.setItem("proxymetrics:theme", "light");
    act(() => {
      render(
        <ThemeProvider>
          <Probe />
        </ThemeProvider>
      );
    });
    expect(screen.getByTestId("t").textContent).toBe("light");
  });
});
