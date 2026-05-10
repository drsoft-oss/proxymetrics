import type { StatusClass, CaptchaKind } from "@/types/api";

// HSL CSS variables defined in styles/index.css. Recharts wants strings, so we
// resolve them via getComputedStyle at render time. For static fills (e.g.,
// donut slices), use these tokens with `hsl(var(--token))`.
export const statusClassToken: Record<StatusClass, string> = {
  "2xx": "var(--success)",
  "3xx": "var(--muted-foreground)",
  "4xx": "var(--destructive)",
  "5xx": "var(--warning)",
  timeout: "var(--warning)",
  connection_error: "var(--destructive)",
};

export function statusFill(sc: StatusClass): string {
  return `hsl(${statusClassToken[sc]})`;
}

// Per-vendor captcha colors. We map each kind to a distinct semantic token
// from the dashboard palette so the captcha page sits next to the rest of
// the app visually. There are exactly 5 kinds and we use 5 distinct tokens.
export const captchaToken: Record<CaptchaKind, string> = {
  recaptcha: "var(--primary)",
  turnstile: "var(--success)",
  hcaptcha:  "var(--warning)",
  datadome:  "var(--destructive)",
  arkose:    "var(--muted-foreground)",
};

export function captchaFill(kind: CaptchaKind): string {
  return `hsl(${captchaToken[kind]})`;
}
