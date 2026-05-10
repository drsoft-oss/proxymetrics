import type { Filter } from "./filters";

export const queryKeys = {
  overview: (filter: Filter) => ["overview", filter] as const,
  rollups: (level: "1min" | "1hour" | "1day", filter: Filter) =>
    ["rollups", level, filter] as const,
  providers: (filter: Filter) => ["providers", filter] as const,
  targets: (filter: Filter) => ["targets", filter] as const,
  statusCodes: (filter: Filter) => ["status-codes", filter] as const,
  dimensions: (from: string, to: string) => ["dimensions", from, to] as const,
  statusCodeDistribution: (filter: Filter) => ["status-code-distribution", filter] as const,
  statusCodeDetail: (code: number, filter: Filter) => ["status-code-detail", code, filter] as const,
  captchas: (filter: Filter) => ["captchas", filter] as const,
  captchaDistribution: (filter: Filter) => ["captcha-distribution", filter] as const,
  captchaDetail: (kind: string, filter: Filter) => ["captcha-detail", kind, filter] as const,
  providerDetail: (id: string, filter: Filter) => ["provider-detail", id, filter] as const,
  targetDetail: (host: string, filter: Filter) => ["target-detail", host, filter] as const,
  fingerprint: () => ["cacert-fingerprint"] as const,
  config: () => ["config"] as const,
  profiles: () => ["profiles"] as const,
  audits: () => ["audits"] as const,
  audit: (id: string) => ["audits", id] as const,
  auditProviders: () => ["audits", "providers"] as const,
};
