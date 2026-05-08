import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiGet, ApiError } from "./api";

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("apiGet", () => {
  it("returns parsed JSON on 2xx", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    );
    const v = await apiGet<{ ok: boolean }>("/api/v1/overview");
    expect(v.ok).toBe(true);
  });

  it("throws ApiError with parsed body on 4xx/5xx", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "bad_request", detail: "nope" }), { status: 400 })
      )
    );
    await expect(apiGet("/api/v1/x")).rejects.toMatchObject({
      status: 400,
      code: "bad_request",
      detail: "nope",
    });
  });

  it("wraps network errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(apiGet("/api/v1/x")).rejects.toBeInstanceOf(ApiError);
  });
});
