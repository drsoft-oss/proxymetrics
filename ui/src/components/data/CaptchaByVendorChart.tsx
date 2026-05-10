import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { captchaFill } from "@/lib/colors";
import { CAPTCHA_KINDS, type CaptchaKind, type CaptchaRow } from "@/types/api";

const config: ChartConfig = CAPTCHA_KINDS.reduce((acc, k) => {
  acc[k] = { label: k, color: captchaFill(k) };
  return acc;
}, {} as ChartConfig);

type Bucket = {
  label: string;
  vendor: string;
  type: string;
  recaptcha: number;
  turnstile: number;
  hcaptcha: number;
  datadome: number;
  arkose: number;
};

function toBuckets(rows: CaptchaRow[]): Bucket[] {
  return rows.map((r) => ({
    label: `${r.vendor} · ${r.type}`,
    vendor: r.vendor,
    type: r.type,
    recaptcha: r.by_kind.recaptcha ?? 0,
    turnstile: r.by_kind.turnstile ?? 0,
    hcaptcha:  r.by_kind.hcaptcha  ?? 0,
    datadome:  r.by_kind.datadome  ?? 0,
    arkose:    r.by_kind.arkose    ?? 0,
  }));
}

export function CaptchaByVendorChart({
  rows,
  isLoading,
}: {
  rows: CaptchaRow[];
  isLoading?: boolean;
}) {
  const buckets = toBuckets(rows);
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Captcha hits by vendor and type</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-60 w-full" />
        ) : buckets.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">
            No captcha hits in this range.
          </div>
        ) : (
          <ChartContainer config={config}>
            <BarChart data={buckets}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
              <XAxis dataKey="label" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
              <ChartTooltip content={<ChartTooltipContent />} />
              {CAPTCHA_KINDS.map((k: CaptchaKind) => (
                <Bar key={k} dataKey={k} stackId="kind" fill={captchaFill(k)} />
              ))}
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
