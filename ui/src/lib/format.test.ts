import { describe, it, expect } from "vitest";
import { formatMoney, formatPercent, formatBytes, formatLatency, formatTimeShort } from "./format";
import { formatNumber } from "./format";

describe("format", () => {
  it("formatMoney", () => {
    expect(formatMoney(1847.32)).toBe("$1,847.32");
    expect(formatMoney(0)).toBe("$0.00");
    expect(formatMoney(NaN)).toBe("$0.00");
  });

  it("formatPercent", () => {
    expect(formatPercent(0.169)).toBe("16.9%");
    expect(formatPercent(0)).toBe("0%");
    expect(formatPercent(1)).toBe("100%");
    expect(formatPercent(NaN)).toBe("0%");
  });

  it("formatBytes", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(1024)).toBe("1.00 KB");
    expect(formatBytes(1024 * 1024 * 1024)).toBe("1.00 GB");
    expect(formatBytes(4_823_100_245)).toBe("4.49 GB");
  });

  it("formatLatency", () => {
    expect(formatLatency(50)).toBe("50 ms");
    expect(formatLatency(2500)).toBe("2.5 s");
    expect(formatLatency(0.4)).toBe("< 1 ms");
    expect(formatLatency(-1)).toBe("—");
  });

  it("formatTimeShort", () => {
    expect(formatTimeShort("2026-04-30T12:34:56Z")).toMatch(/^\d{2}:\d{2}:\d{2}$/);
    expect(formatTimeShort("garbage")).toBe("—");
  });
});

describe("formatNumber", () => {
  it("uses thousand separators", () => {
    expect(formatNumber(1234)).toBe("1,234");
    expect(formatNumber(0)).toBe("0");
    expect(formatNumber(1234567)).toBe("1,234,567");
  });
  it("returns '0' for non-finite input", () => {
    expect(formatNumber(NaN)).toBe("0");
    expect(formatNumber(Infinity)).toBe("0");
  });
});
