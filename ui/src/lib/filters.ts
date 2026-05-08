export type Filter = {
  from: string; // ISO8601
  to: string;   // ISO8601
  profile_id?: string[];
  vendor?: string[];
  type?: string[];
  region?: string[];
  team?: string[];
  project?: string[];
  status_class?: string[];
};

export function defaultFilter(now: Date = new Date()): Filter {
  const to = now.toISOString();
  const from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString();
  return { from, to };
}

const ARRAY_KEYS: Array<keyof Filter> = [
  "profile_id",
  "vendor",
  "type",
  "region",
  "team",
  "project",
  "status_class",
];

export function filterToParams(f: Filter): URLSearchParams {
  const p = new URLSearchParams();
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  for (const key of ARRAY_KEYS) {
    const vals = f[key];
    if (Array.isArray(vals)) {
      for (const v of vals) p.append(key, v);
    }
  }
  return p;
}

export function paramsToFilter(p: URLSearchParams, fallback: Filter = defaultFilter()): Filter {
  const out: Filter = {
    from: p.get("from") ?? fallback.from,
    to: p.get("to") ?? fallback.to,
  };
  for (const key of ARRAY_KEYS) {
    const vals = p.getAll(key);
    if (vals.length > 0) (out as unknown as Record<string, string[]>)[key] = vals;
  }
  return out;
}
