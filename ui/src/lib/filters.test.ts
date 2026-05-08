import { describe, it, expect } from "vitest";
import { defaultFilter, filterToParams, paramsToFilter } from "./filters";

describe("filters", () => {
  it("default window is 30 days", () => {
    const now = new Date("2026-04-30T12:00:00Z");
    const f = defaultFilter(now);
    expect(new Date(f.to).getTime() - new Date(f.from).getTime()).toBe(30 * 24 * 60 * 60 * 1000);
  });

  it("round-trips empty filter", () => {
    const f = defaultFilter(new Date("2026-04-30T12:00:00Z"));
    const p = filterToParams(f);
    expect(paramsToFilter(p, f)).toEqual(f);
  });

  it("round-trips multi-select", () => {
    const f = {
      ...defaultFilter(new Date("2026-04-30T12:00:00Z")),
      profile_id: ["p1", "p2"],
      vendor: ["brightdata"],
    };
    const p = filterToParams(f);
    expect(p.getAll("profile_id")).toEqual(["p1", "p2"]);
    expect(paramsToFilter(p, defaultFilter(new Date("2026-04-30T12:00:00Z")))).toEqual(f);
  });

  it("paramsToFilter falls back to defaults for missing keys", () => {
    const fallback = defaultFilter(new Date("2026-04-30T12:00:00Z"));
    const p = new URLSearchParams();
    expect(paramsToFilter(p, fallback)).toEqual(fallback);
  });
});
