import { useMemo } from "react";
import { LineChart, Line, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip } from "recharts";
import type { RequestRow } from "@/types/audit";

type Props = {
  rows: RequestRow[];
  total: number;
  expectedCity?: string;
  sessionKey?: string;
  uniqueIPCount?: number;
  completedCount?: number;
};

export function AuditResultsSheet({ rows, total, expectedCity, sessionKey, uniqueIPCount, completedCount }: Props) {
  const completed = rows.filter((r) => !r.error).length;
  const locMatches = rows.filter((r) => !r.error && r.location_match).length;
  const typeMismatches = rows.filter((r) => !r.error && !r.type_match).length;
  const errors = rows.filter((r) => r.error);

  const accuracyPct = completed === 0 ? 0 : Math.round((locMatches / completed) * 100);
  const mismatchPct = completed === 0 ? 0 : Math.round((typeMismatches / completed) * 100);

  const matchMix = useMemo(() => {
    const ok = rows.filter((r) => !r.error && r.location_match && r.type_match).length;
    const locFail = rows.filter((r) => !r.error && !r.location_match).length;
    const typeFail = rows.filter((r) => !r.error && !r.type_match).length - 0;
    return [
      { name: "ok", value: ok },
      { name: "loc✗", value: locFail },
      { name: "type✗", value: typeFail },
      { name: "err", value: errors.length },
    ];
  }, [rows, errors.length]);

  const latency = rows.filter((r) => !r.error).map((r) => ({ seq: r.seq, ms: r.duration_ms }));

  const topCities = useMemo(() => {
    const tally = new Map<string, number>();
    for (const r of rows) {
      if (!r.observed_city) continue;
      tally.set(r.observed_city, (tally.get(r.observed_city) ?? 0) + 1);
    }
    return Array.from(tally.entries())
      .map(([city, count]) => ({ city, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5);
  }, [rows]);

  const colour = (pct: number) => (pct >= 95 ? "text-green-500" : pct >= 80 ? "text-amber-500" : "text-red-500");

  const liveUnique = useMemo(() => {
    const set = new Set<string>();
    for (const r of rows) {
      if (!r.error && r.observed_ip) set.add(r.observed_ip);
    }
    return set.size;
  }, [rows]);

  const pool = (() => {
    if (rows.length > 0) {
      return { unique: liveUnique, completed };
    }
    if (uniqueIPCount !== undefined && completedCount !== undefined) {
      return { unique: uniqueIPCount, completed: completedCount };
    }
    return null;
  })();

  const richnessPct = pool && pool.completed > 1
    ? Math.round((pool.unique / pool.completed) * 100)
    : null;

  const richnessClass =
    richnessPct === null ? "text-muted-foreground" :
    richnessPct >= 90 ? "text-green-500" :
    richnessPct >= 50 ? "text-amber-500" :
                        "text-red-500";

  return (
    <div className="space-y-4 text-sm">
      <div className="grid grid-cols-3 gap-2">
        <Stat label="completed" value={`${completed} / ${total}`} />
        <Stat label="accuracy" value={`${accuracyPct}%`} className={colour(accuracyPct)} />
        <Stat label="mismatch" value={`${mismatchPct}%`} className={colour(100 - mismatchPct)} />
      </div>

      <div data-testid="pool-richness" className={"rounded border p-2 " + richnessClass}>
        <div className="text-xs uppercase tracking-wide text-muted-foreground">
          pool richness
        </div>
        <div className="text-lg font-semibold">
          {pool === null
            ? "—"
            : richnessPct === null
              ? "—"
              : `${pool.unique} / ${pool.completed} unique exits (${richnessPct}%)`}
        </div>
        {pool !== null && richnessPct !== null && richnessPct < 100 && (
          <div className="mt-1 text-xs text-muted-foreground">
            {sessionKey
              ? "Rotated session per request, but provider returned duplicate exits — actual pool size may be smaller than your request count."
              : "No session rotation. Low richness usually means the URL is sticky — pick a Session key to test the provider's pool."}
          </div>
        )}
      </div>

      <div>
        <div className="text-xs uppercase tracking-wide text-muted-foreground mb-1">Match mix</div>
        <ResponsiveContainer width="100%" height={80}>
          <BarChart data={matchMix} layout="vertical">
            <XAxis type="number" hide />
            <YAxis type="category" dataKey="name" width={42} />
            <Tooltip />
            <Bar dataKey="value" />
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div>
        <div className="text-xs uppercase tracking-wide text-muted-foreground mb-1">Latency</div>
        <ResponsiveContainer width="100%" height={60}>
          <LineChart data={latency}>
            <Line type="monotone" dataKey="ms" dot={false} />
            <Tooltip />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div>
        <div className="text-xs uppercase tracking-wide text-muted-foreground mb-1">Top observed cities</div>
        <ul className="space-y-0.5">
          {topCities.map((c) => (
            <li key={c.city} className={c.city === expectedCity ? "font-bold" : ""}>
              {c.city} — {c.count}
            </li>
          ))}
        </ul>
      </div>

      {errors.length > 0 && (
        <details>
          <summary className="cursor-pointer text-xs text-red-500">{errors.length} errors</summary>
          <ul className="mt-1 space-y-0.5 text-xs">
            {errors.slice(0, 20).map((r) => (
              <li key={r.seq} className="truncate">#{r.seq}: {r.error}</li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}

function Stat({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className="rounded border p-2">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={"text-lg font-semibold " + (className ?? "")}>{value}</div>
    </div>
  );
}
