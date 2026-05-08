import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export type KpiTileProps = {
  label: string;
  value: string;
  sub?: string;
  delta?: { value: string; direction: "up" | "down" | "flat" };
  variant?: "default" | "hero";
};

export function KpiTile({ label, value, sub, delta, variant = "default" }: KpiTileProps) {
  const isHero = variant === "hero";
  return (
    <Card
      data-hero={isHero ? "true" : undefined}
      className={cn(
        "min-h-[6rem]",
        isHero && "border-destructive/60"
      )}
    >
      <CardContent className="p-4">
        <div className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</div>
        <div
          className={cn(
            "mt-1 text-2xl font-semibold tracking-tight",
            isHero && "text-destructive"
          )}
        >
          {value}
        </div>
        {sub && <div className={cn("mt-1 text-xs", isHero ? "text-destructive/80" : "text-muted-foreground")}>{sub}</div>}
        {delta && (
          <div
            className={cn(
              "mt-1 text-xs",
              delta.direction === "up" && "text-success",
              delta.direction === "down" && "text-destructive",
              delta.direction === "flat" && "text-muted-foreground"
            )}
          >
            {delta.direction === "up" ? "↑ " : delta.direction === "down" ? "↓ " : "→ "}
            {delta.value}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
