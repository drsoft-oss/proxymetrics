import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useCaptchas } from "./useCaptchas";
import { useCaptchaDistribution } from "./useCaptchaDistribution";
import { useCaptchaDetail } from "./useCaptchaDetail";

const captchasSample = {
  rows: [
    { vendor: "brightdata", type: "residential", total: 5, by_kind: { recaptcha: 3, turnstile: 2 } },
  ],
  from: "2026-05-09T00:00:00Z",
  to: "2026-05-10T00:00:00Z",
  source: "rollups_1hour" as const,
};
const distSample = {
  items: [{ kind: "recaptcha", requests: 3 }],
  from: "x", to: "y", source: "events" as const,
};
const detailSample = {
  kind: "recaptcha" as const, requests: 3,
  top_providers: [], top_targets: [],
  from: "x", to: "y",
};

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("captcha hooks", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        let body: object = captchasSample;
        if (url.includes("/distribution")) body = distSample;
        else if (url.includes("/captchas/recaptcha")) body = detailSample;
        return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
      }),
    );
  });
  afterEach(() => vi.unstubAllGlobals());

  it("useCaptchas returns rows", async () => {
    const filter = { from: "x", to: "y" };
    const { result } = renderHook(() => useCaptchas(filter), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data!.rows[0].by_kind.recaptcha).toBe(3);
  });

  it("useCaptchaDistribution returns items", async () => {
    const filter = { from: "x", to: "y" };
    const { result } = renderHook(() => useCaptchaDistribution(filter), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data!.items[0].kind).toBe("recaptcha");
  });

  it("useCaptchaDetail does NOT fetch when kind is null", () => {
    const filter = { from: "x", to: "y" };
    renderHook(() => useCaptchaDetail(null, filter), { wrapper: wrapper() });
    const calls = (globalThis.fetch as unknown as { mock: { calls: unknown[] } }).mock.calls;
    expect(calls.length).toBe(0);
  });

  it("useCaptchaDetail fetches /api/v1/captchas/{kind}", async () => {
    const filter = { from: "x", to: "y" };
    const { result } = renderHook(() => useCaptchaDetail("recaptcha", filter), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const calls = (globalThis.fetch as unknown as { mock: { calls: [string][] } }).mock.calls;
    expect(calls[0][0]).toContain("/api/v1/captchas/recaptcha");
  });
});
