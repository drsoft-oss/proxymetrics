import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProfileDetailDrawer } from "./ProfileDetailDrawer";
import type { ProfileWithUsage, TestResult } from "@/types/profile";

const sampleProfile: ProfileWithUsage = {
  id: "p1",
  label: "BrightData EU",
  vendor: "brightdata",
  type: "residential",
  upstream_url: "http://u",
  currency: "USD",
  price_per_gb: 8.5,
  price_per_gb_overage: null,
  included_gb: null,
  region: "",
  default_team: "",
  default_project: "",
  created_at: "x",
  updated_at: "y",
  requests: 12000,
  spend_usd: 42.1,
};

function mockFetchOnce(body: unknown, init: ResponseInit = { status: 200 }) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      new Response(JSON.stringify(body), {
        headers: { "content-type": "application/json" },
        ...init,
      })
    )
  );
}

function mockFetchReject(message = "network down") {
  vi.stubGlobal("fetch", vi.fn(async () => { throw new Error(message); }));
}

describe("ProfileDetailDrawer", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("renders nothing when profile is null", () => {
    render(<ProfileDetailDrawer profile={null} open={true} onClose={() => {}} />);
    expect(screen.queryByText(/identity/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/activity/i)).not.toBeInTheDocument();
  });

  it("renders nothing when open is false", () => {
    render(<ProfileDetailDrawer profile={sampleProfile} open={false} onClose={() => {}} />);
    expect(screen.queryByText("BrightData EU")).not.toBeInTheDocument();
  });

  it("renders title, subtitle, and the three sections when open with a profile", () => {
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    expect(screen.getByText("BrightData EU")).toBeInTheDocument();
    expect(screen.getByText("p1")).toBeInTheDocument();
    expect(screen.getByText(/identity/i)).toBeInTheDocument();
    expect(screen.getByText(/activity/i)).toBeInTheDocument();
    expect(screen.getByText(/^test$/i)).toBeInTheDocument();
  });

  it("renders Identity fields: provider, type, $/GB", () => {
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    expect(screen.getByText("brightdata")).toBeInTheDocument();
    expect(screen.getByText("residential")).toBeInTheDocument();
    expect(screen.getByText(/8\.50/)).toBeInTheDocument();
  });

  it("renders Activity fields: requests and spend", () => {
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    expect(screen.getByText(/12,?000/)).toBeInTheDocument();
    expect(screen.getByText(/\$42\.10/)).toBeInTheDocument();
  });

  it("calls onClose when the X is clicked", async () => {
    const onClose = vi.fn();
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={onClose} />);
    await userEvent.click(screen.getByLabelText(/close/i));
    expect(onClose).toHaveBeenCalled();
  });

  it("runs a test and renders the result on success", async () => {
    const result: TestResult = {
      exit_ip: "1.2.3.4",
      latency_ms: 350,
      status_code: 200,
      api_used: "ipify",
    };
    mockFetchOnce(result);
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /run test/i }));

    await waitFor(() => expect(screen.getByText("1.2.3.4")).toBeInTheDocument());
    expect(screen.getByText(/350/)).toBeInTheDocument();
    expect(screen.getByText("ipify")).toBeInTheDocument();
    const calls = (globalThis.fetch as unknown as { mock: { calls: [string, RequestInit][] } }).mock.calls;
    expect(calls[0][0]).toBe("/api/v1/profiles/p1/test");
    expect(calls[0][1]?.method).toBe("POST");
  });

  it("renders the upstream error inline when the result body carries an error field", async () => {
    const result: TestResult = {
      exit_ip: "",
      latency_ms: 250,
      status_code: 0,
      api_used: "",
      error: "connection refused",
    };
    mockFetchOnce(result);
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    await userEvent.click(screen.getByRole("button", { name: /run test/i }));
    await waitFor(() => expect(screen.getByText(/connection refused/)).toBeInTheDocument());
  });

  it("renders the hint when present", async () => {
    const result: TestResult = {
      exit_ip: "1.2.3.4",
      latency_ms: 100,
      status_code: 200,
      api_used: "ipify",
      hint: "price_per_gb is unset; cost tracking is disabled for this profile",
    };
    mockFetchOnce(result);
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    await userEvent.click(screen.getByRole("button", { name: /run test/i }));
    await waitFor(() => expect(screen.getByText(/cost tracking is disabled/)).toBeInTheDocument());
  });

  it("renders an HTTP error message when the response is non-2xx", async () => {
    mockFetchOnce({ title: "not_found" }, { status: 404 });
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    await userEvent.click(screen.getByRole("button", { name: /run test/i }));
    await waitFor(() => expect(screen.getByText(/HTTP 404/)).toBeInTheDocument());
    expect(screen.queryByText("ipify")).not.toBeInTheDocument();
  });

  it("renders a network error message when fetch rejects", async () => {
    mockFetchReject("network down");
    render(<ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />);
    await userEvent.click(screen.getByRole("button", { name: /run test/i }));
    await waitFor(() => expect(screen.getByText(/network down/)).toBeInTheDocument());
  });

  it("resets the test state when the profile changes", async () => {
    const result: TestResult = {
      exit_ip: "1.2.3.4",
      latency_ms: 100,
      status_code: 200,
      api_used: "ipify",
    };
    mockFetchOnce(result);
    const { rerender } = render(
      <ProfileDetailDrawer profile={sampleProfile} open={true} onClose={() => {}} />
    );
    await userEvent.click(screen.getByRole("button", { name: /run test/i }));
    await waitFor(() => expect(screen.getByText("1.2.3.4")).toBeInTheDocument());

    rerender(
      <ProfileDetailDrawer
        profile={{ ...sampleProfile, id: "p2", label: "Other" }}
        open={true}
        onClose={() => {}}
      />
    );
    expect(screen.queryByText("1.2.3.4")).not.toBeInTheDocument();
    expect(screen.queryByText("ipify")).not.toBeInTheDocument();
  });
});
