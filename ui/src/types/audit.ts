export type RunStatus = "running" | "completed" | "cancelled" | "failed";

export type RunSummary = {
  id: string;
  started_at: string;
  finished_at?: string;
  status: RunStatus;
  proxy_url: string;
  provider?: string;
  expected_country?: string;
  expected_state?: string;
  expected_city?: string;
  expected_lat: number;
  expected_lon: number;
  expected_type: "residential" | "mobile" | "datacenter";
  check_level: "country" | "state" | "city";
  request_count: number;
  completed_count: number;
  location_match_count: number;
  type_match_count: number;
  error_count: number;
  latency_p50_ms?: number;
  latency_p95_ms?: number;
  fallback_used: boolean;
  session_key?: string;
  unique_ip_count?: number;
  error?: string;
};

export type RequestRow = {
  seq: number;
  started_at: string;
  duration_ms: number;
  observed_ip?: string;
  observed_country?: string;
  observed_state?: string;
  observed_city?: string;
  observed_lat?: number;
  observed_lon?: number;
  is_datacenter: boolean;
  is_mobile: boolean;
  is_proxy: boolean;
  is_vpn: boolean;
  asn?: number;
  company?: string;
  location_match: boolean;
  type_match: boolean;
  error?: string;
  geo_source: "ipapi" | "maxmind" | "none";
};

export type RunDetail = {
  run: RunSummary;
  requests: RequestRow[];
};

export type AuditEvent =
  | ({ type: "request" } & RequestRow)
  | {
      type: "summary";
      completed_count: number;
      location_match_count: number;
      type_match_count: number;
      error_count: number;
    }
  | { type: "finished"; status: RunStatus };

export type AuditProvidersResponse = {
  providers: string[];
  hostname_suffixes: Record<string, string>;
};
