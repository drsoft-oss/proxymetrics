import { useMemo } from "react";
import { MapContainer, TileLayer, Marker, Popup } from "react-leaflet";
import L from "leaflet";
import type { RequestRow } from "@/types/audit";
import { AuditPinPopover } from "./AuditPinPopover";
import "leaflet/dist/leaflet.css";

type Props = {
  rows: RequestRow[];
  center: [number, number];
  zoom: number;
  expected?: { country?: string; state?: string; city?: string; type?: string };
};

const greenIcon = L.divIcon({
  className: "audit-pin audit-pin-green",
  html: `<span style="display:block;width:10px;height:10px;border-radius:9999px;background:#22c55e;border:2px solid #064e3b"></span>`,
  iconSize: [14, 14],
});
const orangeIcon = L.divIcon({
  className: "audit-pin audit-pin-orange",
  html: `<span style="display:block;width:10px;height:10px;border-radius:9999px;background:#f59e0b;border:2px solid #78350f"></span>`,
  iconSize: [14, 14],
});
const redIcon = L.divIcon({
  className: "audit-pin audit-pin-red",
  html: `<span style="display:block;width:10px;height:10px;border-radius:9999px;background:#ef4444;border:2px solid #7f1d1d"></span>`,
  iconSize: [14, 14],
});

// Orange is reserved for the soft-fail case: location is right, the only thing
// wrong is residential-was-promised-but-we-got-mobile. Every other failure
// combo (location wrong, datacenter when residential expected, etc.) is red.
function pinIcon(r: RequestRow, expectedType?: string) {
  if (r.location_match && r.type_match) return greenIcon;
  if (
    r.location_match &&
    !r.type_match &&
    expectedType === "residential" &&
    r.is_mobile &&
    !r.is_datacenter
  ) {
    return orangeIcon;
  }
  return redIcon;
}

export function AuditMap({ rows, center, zoom, expected }: Props) {
  const points = useMemo(
    () => rows.filter((r) => r.observed_lat && r.observed_lon),
    [rows]
  );
  return (
    <MapContainer center={center} zoom={zoom} style={{ height: "100%", width: "100%" }} scrollWheelZoom>
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      {points.map((r) => (
        <Marker
          key={r.seq}
          position={[r.observed_lat!, r.observed_lon!]}
          icon={pinIcon(r, expected?.type)}
        >
          <Popup>
            <AuditPinPopover row={r} expected={expected ?? {}} />
          </Popup>
        </Marker>
      ))}
    </MapContainer>
  );
}
