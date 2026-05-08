import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Copy, X } from "lucide-react";
import { cn } from "@/lib/utils";

const DISMISS_KEY = "proxymetrics:banner:dismissed";

export function OnboardingBanner({
  requestCount,
  fingerprint,
}: {
  requestCount: number;
  fingerprint?: string;
}) {
  const [dismissed, setDismissed] = useState(() => {
    try {
      return window.localStorage.getItem(DISMISS_KEY) === "true";
    } catch {
      return false;
    }
  });

  useEffect(() => {
    if (requestCount === 0 && dismissed) {
      // Spec: dismissal persists until next zero-traffic-hour. v1 just respects the flag.
    }
  }, [requestCount, dismissed]);

  if (requestCount > 0 || dismissed) return null;

  const curlCmd = "curl http://localhost:8081/cacert -o proxymetrics-ca.crt";
  const profileCmd = "proxymetrics profile create --from-file profile.yaml";

  return (
    <div className="rounded-lg border border-primary/30 bg-primary/5 p-4 text-sm">
      <div className="mb-3 flex items-center">
        <div className="font-medium">Get started in two steps</div>
        <button
          aria-label="Dismiss banner"
          onClick={() => {
            try {
              window.localStorage.setItem(DISMISS_KEY, "true");
            } catch {
              // ignore
            }
            setDismissed(true);
          }}
          className="ml-auto rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <Step n={1} title="Install the CA cert" command={curlCmd} note={fingerprint && `Fingerprint: ${fingerprint}`} />
        <Step n={2} title="Register a profile" command={profileCmd} />
      </div>
    </div>
  );
}

function Step({ n, title, command, note }: { n: number; title: string; command: string; note?: string }) {
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
        {n}
      </div>
      <div className="min-w-0 flex-1">
        <div className="font-medium">{title}</div>
        <div className="mt-1 flex items-center gap-2">
          <code className={cn("flex-1 truncate rounded border border-border bg-background px-2 py-1 font-mono text-xs")}>
            {command}
          </code>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Copy command"
            onClick={() => {
              try {
                navigator.clipboard?.writeText(command);
              } catch {
                // ignore
              }
            }}
          >
            <Copy className="h-3.5 w-3.5" />
          </Button>
        </div>
        {note && <div className="mt-1 text-[11px] text-muted-foreground">{note}</div>}
      </div>
    </div>
  );
}
