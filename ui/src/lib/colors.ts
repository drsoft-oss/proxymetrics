import type { StatusClass } from "@/types/api";

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
