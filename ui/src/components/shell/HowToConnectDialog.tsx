import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BookOpen, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiGet } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import type { ConfigResponse, FingerprintResponse } from "@/types/api";
import { LANGUAGES, makeSnippet, type Language } from "./connectSnippets";

export function HowToConnectDialog() {
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<Language>("curl");

  const host = useMemo(() => {
    if (typeof window === "undefined") return "localhost";
    return window.location.hostname || "localhost";
  }, []);

  const fingerprint = useQuery({
    queryKey: queryKeys.fingerprint(),
    queryFn: () => apiGet<FingerprintResponse>("/cacert/fingerprint"),
    staleTime: Infinity,
  });

  const config = useQuery({
    queryKey: queryKeys.config(),
    queryFn: () => apiGet<ConfigResponse>("/api/v1/config"),
    staleTime: Infinity,
  });

  const secret = config.data?.deployment_secret;
  const snippet = makeSnippet(tab, host, secret);

  const copySnippet = () => {
    try {
      navigator.clipboard?.writeText(snippet);
    } catch {
      // ignore — same pattern as OnboardingBanner
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 gap-1.5 px-2 text-xs text-muted-foreground hover:text-foreground"
        >
          <BookOpen className="h-3.5 w-3.5" />
          How to connect
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>How to connect a client</DialogTitle>
          <DialogDescription>
            Install the CA, build the proxy URL, then point your HTTP client at{" "}
            <code className="break-all rounded bg-muted px-1 py-0.5 text-xs">{host}:8080</code>.
          </DialogDescription>
        </DialogHeader>

        <section className="flex min-w-0 flex-col gap-2">
          <h3 className="text-sm font-semibold">Install the CA certificate</h3>
          <p className="text-sm text-muted-foreground">
            ProxyMetrics intercepts HTTPS traffic, so clients need to trust its CA. Each snippet
            below points its HTTP client at the downloaded file. To install it system-wide instead,
            import this file into your OS or language trust store.
          </p>
          <pre className="overflow-x-auto rounded border border-border bg-muted/40 px-3 py-2 font-mono text-xs">
            curl http://{host}:8081/cacert -o proxymetrics-ca.crt
          </pre>
          {fingerprint.isPending && (
            <div className="text-[11px] text-muted-foreground">Fingerprint: loading…</div>
          )}
          {fingerprint.isSuccess && (
            <div className="text-[11px] text-muted-foreground">
              Fingerprint (sha256): <code>{fingerprint.data.sha256}</code>
            </div>
          )}
        </section>

        <section className="flex min-w-0 flex-col gap-2">
          <h3 className="text-sm font-semibold">Build the proxy URL</h3>
          <p className="text-sm text-muted-foreground">
            Point your HTTP client at{" "}
            <code className="break-all rounded bg-muted px-1 py-0.5 text-xs">
              http://&lt;base64url(upstream_url)&gt;:{secret ?? "<deployment-secret>"}@{host}:8080
            </code>
            . The username slot is the base64-encoded URL of your upstream proxy. The password is
            this deployment's{" "}
            <code className="break-all rounded bg-muted px-1 py-0.5 text-xs">PROXYMETRICS_SECRET</code>{" "}
            (shown above).
          </p>
          <p className="text-sm text-muted-foreground">
            Three reserved tags in the upstream username drive cost attribution:{" "}
            <code>provider-&lt;name&gt;</code>, <code>type-&lt;residential|datacenter|isp|mobile&gt;</code>,
            and <code>price-&lt;cents-per-gb&gt;</code>. Everything else in the username — customer,
            zone, country, session — is forwarded to the upstream verbatim.
          </p>
          <pre className="rounded border border-border bg-muted/40 px-3 py-2 font-mono text-xs whitespace-pre-wrap break-all">
            http://customer-acme-zone-residential-country-us-
            <strong>provider-brightdata</strong>-<strong>type-residential</strong>-
            <strong>price-1200</strong>:upstream_password@brd.superproxy.io:22225
          </pre>
          <p className="text-[11px] text-muted-foreground">price-1200 = $12.00 per GB.</p>
        </section>

        <section className="flex min-w-0 flex-col gap-2">
          <h3 className="text-sm font-semibold">Make a request</h3>
          <Tabs value={tab} onValueChange={(v) => setTab(v as Language)} className="min-w-0">
            <div className="flex items-center justify-between gap-2">
              <TabsList>
                {LANGUAGES.map((l) => (
                  <TabsTrigger key={l.id} value={l.id}>
                    {l.label}
                  </TabsTrigger>
                ))}
              </TabsList>
              <Button
                variant="ghost"
                size="sm"
                aria-label="Copy snippet"
                className="h-7 gap-1.5 px-2 text-xs"
                onClick={copySnippet}
              >
                <Copy className="h-3.5 w-3.5" />
                Copy
              </Button>
            </div>
            {LANGUAGES.map((l) => (
              <TabsContent key={l.id} value={l.id} className="min-w-0">
                <pre className="min-w-0 max-w-full overflow-x-auto rounded border border-border bg-muted/40 px-3 py-2 font-mono text-xs leading-relaxed">
                  {makeSnippet(l.id, host)}
                </pre>
              </TabsContent>
            ))}
          </Tabs>
        </section>
      </DialogContent>
    </Dialog>
  );
}
