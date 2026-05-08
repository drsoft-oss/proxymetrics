import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuditSetupForm } from "./AuditSetupForm";
import { queryKeys } from "@/lib/queryKeys";

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

function wrap(
  ui: React.ReactNode,
  providersData: { providers: string[]; hostname_suffixes: Record<string, string> } = {
    providers: [],
    hostname_suffixes: {},
  },
) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  // Pre-seed the providers query so the form doesn't issue a fetch for it.
  // Existing tests inherit the empty default; provider-input tests pass real data.
  qc.setQueryData(queryKeys.auditProviders(), providersData);
  return <QueryClientProvider client={qc}>{ui}</QueryClientProvider>;
}

const DATABAY_URL =
  "http://hardcore_oparin6667-zone-residential-countryCode-RO-sessionId@eu-gw.databay.co:8888";

describe("AuditSetupForm — token picker", () => {
  it("renders username tokens after URL is entered", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    await waitFor(() => {
      const preview = screen.getByTestId("username-tokens");
      expect(preview).toHaveTextContent(/hardcore_oparin6667/);
      expect(preview).toHaveTextContent(/countryCode/);
      expect(preview).toHaveTextContent(/sessionId/);
    });
  });

  it("country key dropdown lists all tokens plus '(none)'", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    const select = await screen.findByLabelText(/country key/i);
    const optionLabels = Array.from(select.querySelectorAll("option")).map((o) => o.textContent);
    expect(optionLabels).toEqual(
      expect.arrayContaining([
        expect.stringMatching(/none/i),
        "hardcore_oparin6667",
        "zone",
        "residential",
        "countryCode",
        "RO",
        "sessionId",
      ]),
    );
  });

  it("shows the resolved value next to a picked country key", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    const select = await screen.findByLabelText(/country key/i);
    await user.selectOptions(select, "countryCode");
    await waitFor(() => {
      expect(screen.getByTestId("country-value")).toHaveTextContent("RO");
    });
  });

  it("blocks submit when no country key is picked", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => {
      expect(screen.getByText(/country key is required/i)).toBeInTheDocument();
    });
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });

  it("blocks submit when picked country key has no following value", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    // Username's last token is `tail`. Picking `tail` as the country key
    // should be blocked.
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO-tail:p@h:1",
    );
    const select = await screen.findByLabelText(/country key/i);
    await user.selectOptions(select, "tail");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => {
      expect(screen.getByText(/country key has no following value/i)).toBeInTheDocument();
    });
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });

  it("submits the explicit-geo payload", async () => {
    const user = userEvent.setup();
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ id: "r1", status: "running" }),
    });
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    await user.selectOptions(await screen.findByLabelText(/country key/i), "countryCode");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("r1"));
    const call = (globalThis.fetch as any).mock.calls[0];
    expect(call[0]).toBe("/api/v1/audits");
    const body = JSON.parse(call[1].body);
    expect(body).toEqual({
      proxy_url: DATABAY_URL,
      expected_country: "RO",
      expected_type: "residential",
      request_count: 100,
    });
    expect(body.check_level).toBeUndefined();
    expect(body.expected_state).toBeUndefined();
    expect(body.expected_city).toBeUndefined();
  });

  it("includes state/city when their keys are picked", async () => {
    const user = userEvent.setup();
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ id: "r2", status: "running" }),
    });
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
    );
    await user.selectOptions(await screen.findByLabelText(/country key/i), "country");
    await user.selectOptions(await screen.findByLabelText(/state key/i), "state");
    await user.selectOptions(await screen.findByLabelText(/city key/i), "city");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("r2"));
    const body = JSON.parse((globalThis.fetch as any).mock.calls[0][1].body);
    expect(body.expected_country).toBe("RO");
    expect(body.expected_state).toBe("Bucharest");
    expect(body.expected_city).toBe("Bucharest");
  });

  it("rejects URL with no username", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), "http://host:1");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => {
      expect(screen.getByText(/url must include a username/i)).toBeInTheDocument();
    });
  });
});

// A URL where sessionId is followed by a value (abc), unlike DATABAY_URL where
// sessionId is the last token.
const DATABAY_URL_WITH_SESSION =
  "http://hardcore_oparin6667-zone-residential-countryCode-RO-sessionId-abc@eu-gw.databay.co:8888";

describe("AuditSetupForm — session key", () => {
  it("renders a Session key dropdown listing username tokens", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    const select = await screen.findByLabelText(/session key/i);
    const optionLabels = Array.from(select.querySelectorAll("option")).map((o) => o.textContent);
    expect(optionLabels).toEqual(
      expect.arrayContaining([
        expect.stringMatching(/none/i),
        "sessionId",
      ]),
    );
  });

  it("includes session_key in the submit payload when picked", async () => {
    const user = userEvent.setup();
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ id: "rsk", status: "running" }),
    });
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL_WITH_SESSION);
    await user.selectOptions(await screen.findByLabelText(/country key/i), "countryCode");
    await user.selectOptions(await screen.findByLabelText(/session key/i), "sessionId");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("rsk"));
    const body = JSON.parse((globalThis.fetch as any).mock.calls[0][1].body);
    expect(body.session_key).toBe("sessionId");
  });

  it("omits session_key when (none) is picked", async () => {
    const user = userEvent.setup();
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ id: "rno", status: "running" }),
    });
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    await user.selectOptions(await screen.findByLabelText(/country key/i), "countryCode");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("rno"));
    const body = JSON.parse((globalThis.fetch as any).mock.calls[0][1].body);
    expect(body.session_key).toBeUndefined();
  });

  it("blocks submit when picked Session key has no following value", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO-session:p@h:1",
    );
    await user.selectOptions(await screen.findByLabelText(/country key/i), "country");
    await user.selectOptions(await screen.findByLabelText(/session key/i), "session");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => {
      expect(screen.getByText(/session key has no following value/i)).toBeInTheDocument();
    });
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });
});

describe("AuditSetupForm — session warning banner", () => {
  it("shows a banner when username has 'sessionId' but picker is (none)", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    expect(await screen.findByTestId("session-warning")).toHaveTextContent(/sessionId/);
  });

  it("disappears when the user picks the suggested key", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    expect(await screen.findByTestId("session-warning")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /use it/i }));
    expect(screen.queryByTestId("session-warning")).not.toBeInTheDocument();
    expect(await screen.findByLabelText(/session key/i)).toHaveValue("sessionId");
  });

  it("disappears when dismissed via 'Run anyway'", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(screen.getByLabelText(/proxy url/i), DATABAY_URL);
    expect(await screen.findByTestId("session-warning")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /run anyway/i }));
    expect(screen.queryByTestId("session-warning")).not.toBeInTheDocument();
  });

  it("never appears when the username has no session-like token", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-zone-residential-country-RO:p@h:1",
    );
    expect(screen.queryByTestId("session-warning")).not.toBeInTheDocument();
  });
});

describe("AuditSetupForm — provider input", () => {
  const PROVIDERS_DATA = {
    providers: ["brightdata", "databay", "oxylabs"],
    hostname_suffixes: {
      "superproxy.io": "brightdata",
      "databay.co": "databay",
      "oxylabs.io": "oxylabs",
    },
  };

  function mockPostResponse() {
    (globalThis.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ id: "r1", status: "running" }),
    });
  }

  it("prefills provider from the URL credtag's provider-X token", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-provider-acme-country-RO:p@some.host:1",
    );
    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "provider" })).toHaveTextContent("acme");
    });
  });

  it("prefills provider from the hostname-suffix map when no credtag", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO:p@brd.superproxy.io:22225",
    );
    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "provider" })).toHaveTextContent(
        "brightdata",
      );
    });
  });

  it("leaves provider empty when neither credtag nor hostname matches", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO:p@unknown.example:1",
    );
    await waitFor(() => {
      const trigger = screen.getByRole("combobox", { name: "provider" });
      expect(trigger).not.toHaveTextContent(/brightdata|databay|oxylabs/);
    });
  });

  it("keeps the user's typed provider when the URL changes (sticky)", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO:p@brd.superproxy.io:1",
    );
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const cbInput = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(cbInput, "custom-pool");
    await user.click(await screen.findByText(/create.*custom-pool/i));
    const urlInput = screen.getByLabelText(/proxy url/i);
    await user.clear(urlInput);
    await user.type(urlInput, "http://user-country-RO:p@brd.superproxy.io:2");
    expect(screen.getByRole("combobox", { name: "provider" })).toHaveTextContent("custom-pool");
  });

  it("re-engages prefill after the URL is cleared to empty", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    const urlInput = screen.getByLabelText(/proxy url/i);
    await user.type(urlInput, "http://user-country-RO:p@brd.superproxy.io:1");
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const cbInput = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(cbInput, "custom-pool");
    await user.click(await screen.findByText(/create.*custom-pool/i));
    await user.clear(urlInput);
    await user.type(urlInput, "http://user-country-RO:p@gw.databay.co:1");
    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "provider" })).toHaveTextContent("databay");
    });
  });

  it("preserves the user's provider pick across URL clear (no auto-clear)", async () => {
    const user = userEvent.setup();
    render(wrap(<AuditSetupForm onStarted={() => {}} />, PROVIDERS_DATA));
    const urlInput = screen.getByLabelText(/proxy url/i);
    // Seed: paste a known-host URL so prefill engages.
    await user.type(urlInput, "http://user-country-RO:p@brd.superproxy.io:1");
    // Override with a custom value.
    await user.click(screen.getByRole("combobox", { name: "provider" }));
    const cbInput = await screen.findByRole("combobox", { name: "provider search" });
    await user.type(cbInput, "custom-pool");
    await user.click(await screen.findByText(/create.*custom-pool/i));
    // Clearing the URL alone must NOT reset the provider.
    await user.clear(urlInput);
    expect(screen.getByRole("combobox", { name: "provider" })).toHaveTextContent("custom-pool");
  });

  it("includes provider in submit body, omitted when empty", async () => {
    mockPostResponse();
    const user = userEvent.setup();
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO:p@brd.superproxy.io:1",
    );
    await user.selectOptions(await screen.findByLabelText(/country key/i), "country");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("r1"));
    const calls = (globalThis.fetch as any).mock.calls.filter(
      (c: any[]) => c[0] === "/api/v1/audits" && c[1]?.method === "POST",
    );
    expect(calls.length).toBe(1);
    const body = JSON.parse(calls[0][1].body);
    expect(body.provider).toBe("brightdata");
  });

  it("omits provider when empty (unknown host, no credtag)", async () => {
    mockPostResponse();
    const user = userEvent.setup();
    const onStarted = vi.fn();
    render(wrap(<AuditSetupForm onStarted={onStarted} />, PROVIDERS_DATA));
    await user.type(
      screen.getByLabelText(/proxy url/i),
      "http://user-country-RO:p@unknown.example:1",
    );
    await user.selectOptions(await screen.findByLabelText(/country key/i), "country");
    fireEvent.click(screen.getByRole("button", { name: /start audit/i }));
    await waitFor(() => expect(onStarted).toHaveBeenCalledWith("r1"));
    const calls = (globalThis.fetch as any).mock.calls.filter(
      (c: any[]) => c[0] === "/api/v1/audits" && c[1]?.method === "POST",
    );
    const body = JSON.parse(calls[0][1].body);
    expect(body.provider).toBeUndefined();
  });
});
