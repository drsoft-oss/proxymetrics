export type Profile = {
  id: string;
  label: string;
  vendor: string;
  type: string;
  region?: string;
  upstream_url: string;
  price_per_gb?: number | null;
  price_per_gb_overage?: number | null;
  included_gb?: number | null;
  currency: string;
  default_team?: string;
  default_project?: string;
  created_at: string;
  updated_at: string;
};

export type ProfileWithUsage = Profile & {
  requests: number;
  spend_usd: number;
};

export type TestResult = {
  exit_ip: string;
  latency_ms: number;
  status_code: number;
  api_used: string;
  error?: string;
  hint?: string;
};

export const DEFAULT_PROFILE_ID = "default";
