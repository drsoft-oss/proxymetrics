import { useEffect, useState } from "react";
import { DrillSheet, DrillSection } from "@/components/primitives/DrillSheet";
import type { ProfileWithUsage, TestResult } from "@/types/profile";

export type ProfileDetailDrawerProps = {
  profile: ProfileWithUsage | null;
  open: boolean;
  onClose: () => void;
};

type TestState =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "result"; result: TestResult }
  | { kind: "error"; message: string };

export function ProfileDetailDrawer({ profile, open, onClose }: ProfileDetailDrawerProps) {
  const [test, setTest] = useState<TestState>({ kind: "idle" });

  useEffect(() => {
    setTest({ kind: "idle" });
  }, [profile?.id]);

  if (!profile) return null;

  async function runTest() {
    if (!profile) return;
    setTest({ kind: "loading" });
    try {
      const res = await fetch(`/api/v1/profiles/${profile.id}/test`, { method: "POST" });
      if (!res.ok) {
        setTest({ kind: "error", message: `HTTP ${res.status}` });
        return;
      }
      const result = (await res.json()) as TestResult;
      setTest({ kind: "result", result });
    } catch (e) {
      setTest({ kind: "error", message: e instanceof Error ? e.message : "request failed" });
    }
  }

  return (
    <DrillSheet open={open} onClose={onClose} title={profile.label} subtitle={profile.id}>
      <DrillSection title="Identity">
        <KV label="Provider" value={profile.vendor} />
        <KV label="Type" value={profile.type} />
        <KV
          label="$/GB"
          value={profile.price_per_gb == null ? "—" : profile.price_per_gb.toFixed(2)}
        />
      </DrillSection>

      <DrillSection title="Activity (24h)">
        <KV label="Requests" value={profile.requests.toLocaleString()} />
        <KV label="Spend" value={`$${profile.spend_usd.toFixed(2)}`} />
      </DrillSection>

      <DrillSection title="Test">
        <button
          type="button"
          onClick={runTest}
          disabled={test.kind === "loading"}
          className="rounded-md bg-foreground px-3 py-1.5 text-xs font-medium text-background disabled:opacity-50"
        >
          {test.kind === "loading" ? "Running…" : "Run test"}
        </button>

        {test.kind === "result" && <TestResultView result={test.result} />}
        {test.kind === "error" && (
          <p className="mt-3 text-xs text-destructive">{test.message}</p>
        )}
      </DrillSection>
    </DrillSheet>
  );
}

function KV({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-1 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}

function TestResultView({ result }: { result: TestResult }) {
  return (
    <div className="mt-3 space-y-1">
      <KV label="Exit IP" value={result.exit_ip || "—"} />
      <KV label="Latency" value={`${result.latency_ms} ms`} />
      <KV label="Status" value={String(result.status_code)} />
      <KV label="API used" value={result.api_used || "—"} />
      {result.error && (
        <p className="pt-2 text-xs text-destructive">{result.error}</p>
      )}
      {result.hint && (
        <p className="pt-2 text-[11px] italic text-muted-foreground">{result.hint}</p>
      )}
    </div>
  );
}
