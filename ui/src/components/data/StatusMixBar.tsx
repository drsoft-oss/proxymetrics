import { statusFill } from "@/lib/colors";
import type { StatusMix } from "@/types/api";

export function StatusMixBar({ mix, width = 80 }: { mix: StatusMix; width?: number }) {
  const total = mix.class2xx + mix.class3xx + mix.class4xx + mix.class5xx;
  if (total === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  const segs = [
    { cls: "2xx" as const, n: mix.class2xx },
    { cls: "3xx" as const, n: mix.class3xx },
    { cls: "4xx" as const, n: mix.class4xx },
    { cls: "5xx" as const, n: mix.class5xx },
  ];
  return (
    <div
      role="img"
      aria-label={`status mix: ${total} requests`}
      className="flex h-2.5 overflow-hidden rounded-sm"
      style={{ width }}
    >
      {segs.map((s) => {
        const pct = (s.n / total) * 100;
        return (
          <div
            key={s.cls}
            data-testid="mix-seg"
            data-class={s.cls}
            title={`${s.cls}: ${s.n} (${pct.toFixed(1)}%)`}
            style={{ width: `${pct}%`, background: statusFill(s.cls) }}
          />
        );
      })}
    </div>
  );
}
