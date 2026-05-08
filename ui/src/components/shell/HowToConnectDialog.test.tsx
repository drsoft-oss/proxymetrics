import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { HowToConnectDialog } from "./HowToConnectDialog";

function renderWith(fingerprintFetch: () => Promise<Response>) {
  vi.stubGlobal("fetch", vi.fn().mockImplementation(fingerprintFetch));
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  return render(<HowToConnectDialog />, { wrapper });
}

beforeEach(() => {
  Object.defineProperty(window, "location", {
    value: { hostname: "proxy.example.com" },
    writable: true,
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("HowToConnectDialog", () => {
  it("renders the trigger button with visible label", () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    expect(screen.getByRole("button", { name: /how to connect/i })).toBeInTheDocument();
  });

  it("opens the dialog with all three sections on click", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /install the ca certificate/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /build the proxy url/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /make a request/i })).toBeInTheDocument();
  });

  it("renders all five language tab triggers", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    for (const label of ["curl", "Python", "Node.js", "Go", "Ruby"]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument();
    }
  });

  it("defaults to the curl tab and shows its snippet content", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    expect(screen.getByRole("tab", { name: "curl", selected: true })).toBeInTheDocument();
    expect(screen.getByText(/--cacert \.\/proxymetrics-ca\.crt/)).toBeInTheDocument();
  });

  it("switching to the Python tab swaps the snippet content", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    await userEvent.click(screen.getByRole("tab", { name: "Python" }));
    expect(screen.getByText(/import base64, os, requests/)).toBeInTheDocument();
  });

  it("substitutes window.location.hostname into the curl snippet", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    // hostname appears in multiple nodes (description, build-url section, snippet pre); any match suffices
    expect(screen.getAllByText(/proxy\.example\.com:8080/).length).toBeGreaterThan(0);
  });

  it("renders the fingerprint when the query resolves", async () => {
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD:EF", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    await waitFor(() => expect(screen.getByText(/AB:CD:EF/)).toBeInTheDocument());
  });

  it("hides the fingerprint line when the query fails", async () => {
    renderWith(async () => new Response("boom", { status: 500 }));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.queryByText(/Fingerprint/i)).not.toBeInTheDocument();
  });

  it("copy button writes the curl snippet to clipboard", async () => {
    const writeText = vi.fn();
    Object.assign(navigator, { clipboard: { writeText } });
    renderWith(async () => new Response(JSON.stringify({ sha256: "AB:CD", valid_until: "2099-01-01" })));
    await userEvent.click(screen.getByRole("button", { name: /how to connect/i }));
    await userEvent.click(screen.getByRole("button", { name: /copy snippet/i }));
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining("--cacert"));
  });
});
