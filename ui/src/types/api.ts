// Types matching the backend response shapes from sub-projects #1 and #2.

export type StatusClass = "2xx" | "3xx" | "4xx" | "5xx" | "timeout" | "connection_error";

export type OverviewResponse = {
  from: string;
  to: string;
  spend_usd_total: number;
  wasted_usd_total: number;
  wasted_pct_of_spend: number;
  bytes_total: number;
  request_count: number;
  success_rate: number;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type RollupRow = {
  ts_bucket: string;
  profile_id: string;
  vendor: string;
  type: string;
  region?: string;
  status_class: StatusClass;
  target_host?: string;
  team?: string;
  project?: string;
  request_count: number;
  bytes_in_total: number;
  bytes_out_total: number;
  latency_ms_avg: number;
  latency_ms_p50: number;
  latency_ms_p95: number;
  latency_ms_p99: number;
  cost_usd_total: number;
  success_count: number;
  failure_count: number;
};

export type RollupsResponse = {
  items: RollupRow[];
  limit: number;
  offset: number;
  from: string;
  to: string;
};

export type StatusMix = {
  class2xx: number;
  class3xx: number;
  class4xx: number;
  class5xx: number;
};

export type ProviderItem = {
  profile_id: string;
  label: string;
  vendor: string;
  type: string;
  region?: string;
  request_count: number;
  bytes_total: number;
  success_rate: number;
  latency_ms_p50?: number;
  latency_ms_p95?: number;
  spend_usd: number;
  wasted_usd: number;
  last_seen: string;
  status_mix: StatusMix;
  requests_by_hour: number[]; // length 24, oldest first
};

export type ProvidersResponse = {
  items: ProviderItem[];
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type TargetItem = {
  target_host: string;
  request_count: number;
  bytes_total: number;
  success_rate: number;
  spend_usd: number;
  wasted_usd: number;
  last_seen: string;
  status_mix: StatusMix;
  requests_by_hour: number[];
};

export type TargetsResponse = {
  items: TargetItem[];
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type StatusByCell = { request_count: number; bytes_total: number; spend_usd: number };

export type StatusCodesRow = {
  profile_id: string;
  label: string;
  by_status: Partial<Record<StatusClass, StatusByCell>>;
};

export type StatusCodesResponse = {
  rows: StatusCodesRow[];
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type EventRow = {
  ts: string;
  request_id: string;
  profile_id: string;
  vendor: string;
  type: string;
  region?: string;
  target_host?: string;
  target_path_hash?: string;
  status_code: number;
  status_class: StatusClass;
  bytes_in: number;
  bytes_out: number;
  latency_ms: number;
  cost_usd: number;
  team?: string;
  project?: string;
  captcha_kind?: CaptchaKind | "";
};

export type EventsResponse = {
  items: EventRow[];
  limit: number;
  offset: number;
  total_estimate: number;
  from: string;
  to: string;
};

export type FingerprintResponse = {
  sha256: string;
  valid_until: string;
};

export type ConfigResponse = {
  deployment_secret: string;
};

export type DimensionsResponse = {
  vendor: string[];
  type: string[];
  region: string[];
  team: string[];
  project: string[];
  status_class: string[];
  captcha_kind: string[];
  profile_id: { id: string; name: string }[];
};

export type StatusCodeDistItem = {
  code: number;
  class: "2xx" | "3xx" | "4xx" | "5xx";
  requests: number;
  wasted_usd: number;
  spend_usd: number;
  top_provider_vendor: string;
};

export type StatusCodeDistributionResponse = {
  items: StatusCodeDistItem[];
  from: string;
  to: string;
  source: "events";
};

export type StatusCodeDetailProvider = {
  vendor: string;
  requests: number;
  wasted_usd: number;
  spend_usd: number;
};

export type StatusCodeDetailTarget = {
  host: string;
  requests: number;
  wasted_usd: number;
  spend_usd: number;
};

export type StatusCodeDetailResponse = {
  code: number;
  class: "2xx" | "3xx" | "4xx" | "5xx";
  requests: number;
  wasted_usd: number;
  spend_usd: number;
  top_providers: StatusCodeDetailProvider[];
  top_targets: StatusCodeDetailTarget[];
  from: string;
  to: string;
};

// Provider/Target detail responses (from existing /providers/{id} and /targets/{host}
// endpoints — typing what was previously consumed as `any`).

export type ProviderDetailResponse = {
  profile: {
    profile_id: string;
    label: string;
    vendor: string;
    type: string;
    region?: string;
  };
  by_status_class: Array<{
    status_class: StatusClass;
    request_count: number;
    bytes_total: number;
    spend_usd: number;
  }>;
  top_targets: Array<{
    target_host: string;
    request_count: number;
    success_rate: number;
    bytes_total: number;
    spend_usd: number;
  }>;
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type TargetDetailResponse = {
  target_host: string;
  by_provider: Array<{
    profile_id: string;
    vendor: string;
    type: string;
    region?: string;
    request_count: number;
    bytes_total: number;
    success_rate: number;
    spend_usd: number;
    wasted_usd: number;
  }>;
  by_status_class: Array<{
    status_class: StatusClass;
    request_count: number;
    bytes_total: number;
    spend_usd: number;
  }>;
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type CaptchaKind = "recaptcha" | "turnstile" | "hcaptcha" | "datadome" | "arkose";

export const CAPTCHA_KINDS: CaptchaKind[] = ["recaptcha", "turnstile", "hcaptcha", "datadome", "arkose"];

export type CaptchaRow = {
  vendor: string;
  type: string;
  total: number;
  by_kind: Partial<Record<CaptchaKind, number>>;
};

export type CaptchasResponse = {
  rows: CaptchaRow[];
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type CaptchaDistributionItem = { kind: CaptchaKind; requests: number };

export type CaptchaDistributionResponse = {
  items: CaptchaDistributionItem[];
  from: string;
  to: string;
  source: "events" | "rollups_1hour" | "rollups_1day";
};

export type CaptchaDetailProvider = { vendor: string; type: string; requests: number };
export type CaptchaDetailTarget = { host: string; requests: number };

export type CaptchaDetailResponse = {
  kind: CaptchaKind;
  requests: number;
  top_providers: CaptchaDetailProvider[];
  top_targets: CaptchaDetailTarget[];
  from: string;
  to: string;
};
