import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { AuditMap } from "./AuditMap";
import type { RequestRow } from "@/types/audit";

function row(seq: number, lat: number, lon: number, ok: boolean): RequestRow {
  return {
    seq, started_at: new Date().toISOString(), duration_ms: 1,
    is_datacenter: false, is_mobile: false, is_proxy: false, is_vpn: false,
    location_match: ok, type_match: ok, geo_source: "ipapi",
    observed_lat: lat, observed_lon: lon,
  };
}

describe("AuditMap", () => {
  it("renders a marker per request with observed coords", () => {
    const { container } = render(
      <AuditMap rows={[row(1, 44.4, 26.1, true), row(2, 50.1, 8.7, false)]}
        center={[44.43, 26.10]} zoom={5} />
    );
    expect(container.querySelectorAll(".audit-pin").length).toBe(2);
  });

  it("paints residential-expected-but-mobile-observed pins orange when location matches", () => {
    const softFail: RequestRow = {
      ...row(1, 44.4, 26.1, false),
      location_match: true,
      type_match: false,
      is_mobile: true,
    };
    const { container } = render(
      <AuditMap rows={[softFail]} center={[44.43, 26.10]} zoom={5}
        expected={{ country: "RO", type: "residential" }} />
    );
    expect(container.querySelectorAll(".audit-pin-orange").length).toBe(1);
    expect(container.querySelectorAll(".audit-pin-red").length).toBe(0);
  });

  it("keeps red when location also fails, even for residential-vs-mobile", () => {
    const hardFail: RequestRow = {
      ...row(1, 44.4, 26.1, false),
      location_match: false,
      type_match: false,
      is_mobile: true,
    };
    const { container } = render(
      <AuditMap rows={[hardFail]} center={[44.43, 26.10]} zoom={5}
        expected={{ country: "RO", type: "residential" }} />
    );
    expect(container.querySelectorAll(".audit-pin-orange").length).toBe(0);
    expect(container.querySelectorAll(".audit-pin-red").length).toBe(1);
  });

  it("keeps red for residential-vs-datacenter mismatches", () => {
    const dcFail: RequestRow = {
      ...row(1, 44.4, 26.1, false),
      location_match: true,
      type_match: false,
      is_datacenter: true,
    };
    const { container } = render(
      <AuditMap rows={[dcFail]} center={[44.43, 26.10]} zoom={5}
        expected={{ country: "RO", type: "residential" }} />
    );
    expect(container.querySelectorAll(".audit-pin-orange").length).toBe(0);
    expect(container.querySelectorAll(".audit-pin-red").length).toBe(1);
  });
});
