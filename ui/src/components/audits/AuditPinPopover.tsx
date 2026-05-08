import type { RequestRow } from "@/types/audit";

type Expected = {
  country?: string;
  state?: string;
  city?: string;
  type?: string;
};

export function AuditPinPopover({ row, expected }: { row: RequestRow; expected: Expected }) {
  const observedType = row.is_datacenter ? "datacenter" : row.is_mobile ? "mobile" : "residential";
  const expectedLoc = [expected.country, expected.state, expected.city].filter(Boolean).join(", ");
  const actualLoc = [row.observed_country, row.observed_state, row.observed_city].filter(Boolean).join(", ");
  return (
    <div className="text-xs space-y-1">
      <div><b>{row.observed_ip ?? "—"}</b> {row.asn ? <>· AS{row.asn}</> : null}</div>
      <div>{row.company ?? "—"}</div>
      <div>Expected: {expectedLoc}{expected.type ? ` · ${expected.type}` : ""}</div>
      <div>Actual: {actualLoc} · {observedType}</div>
      <div>{row.duration_ms} ms</div>
      <div className="pt-1 border-t">
        <span className={row.location_match ? "text-green-500" : "text-red-500"}>
          location {row.location_match ? "✓" : "✗"}
        </span>
        {" · "}
        <span className={row.type_match ? "text-green-500" : "text-red-500"}>
          type {row.type_match ? "✓" : "✗"}
        </span>
      </div>
    </div>
  );
}
