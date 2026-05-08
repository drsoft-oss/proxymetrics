import { useEffect, useId, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Combobox } from "@/components/ui/combobox";
import { queryKeys } from "@/lib/queryKeys";
import type { AuditProvidersResponse } from "@/types/audit";

type Props = {
  onStarted: (runId: string) => void;
  initial?: {
    proxyUrl?: string;
    countryKey?: string;
    stateKey?: string;
    cityKey?: string;
    sessionKey?: string;
    type?: "residential" | "mobile" | "datacenter";
    count?: number;
  };
};

const NONE = "";

const SESSION_LIKE_KEYS = new Set(["session", "sessionid", "sessid", "sid"]);

function findSessionLikeToken(tokens: string[]): string | undefined {
  return tokens.find((t) => SESSION_LIKE_KEYS.has(t.toLowerCase()));
}

type ParsedURL =
  | { ok: true; tokens: string[] }
  | { ok: false; reason: "malformed" | "no-username" };

function parseURL(raw: string): ParsedURL {
  if (!raw) return { ok: false, reason: "malformed" };
  let u: URL;
  try {
    u = new URL(raw);
  } catch {
    return { ok: false, reason: "malformed" };
  }
  if (!u.username) return { ok: false, reason: "no-username" };
  return { ok: true, tokens: u.username.split("-") };
}

function valueAfter(tokens: string[], key: string): string | undefined {
  if (!key) return undefined;
  const i = tokens.indexOf(key);
  if (i < 0 || i === tokens.length - 1) return undefined;
  return tokens[i + 1];
}

function guessProviderFromHost(
  host: string,
  suffixMap: Record<string, string>,
): string {
  const h = host.trim().toLowerCase();
  if (!h) return "";
  let cur = h;
  while (cur) {
    if (suffixMap[cur]) return suffixMap[cur];
    const i = cur.indexOf(".");
    if (i < 0) return "";
    cur = cur.slice(i + 1);
  }
  return "";
}

export function AuditSetupForm({ onStarted, initial }: Props) {
  const [proxyUrl, setProxyUrl] = useState(initial?.proxyUrl ?? "");
  const [countryKey, setCountryKey] = useState(initial?.countryKey ?? NONE);
  const [stateKey, setStateKey] = useState(initial?.stateKey ?? NONE);
  const [cityKey, setCityKey] = useState(initial?.cityKey ?? NONE);
  const [sessionKey, setSessionKey] = useState(initial?.sessionKey ?? NONE);
  const [type, setType] = useState<"residential" | "mobile" | "datacenter">(
    initial?.type ?? "residential",
  );
  const [count, setCount] = useState(initial?.count ?? 100);
  const [error, setError] = useState<string | null>(null);
  const [warningDismissed, setWarningDismissed] = useState(false);
  const [provider, setProvider] = useState("");
  // The last value the auto-prefill applied. We only re-prefill when the new
  // computed value differs — that lets a user override (custom provider) stick
  // across cosmetic URL changes, while still re-engaging when the URL points
  // at a genuinely different provider.
  const lastAutoPrefillRef = useRef("");

  const providersQuery = useQuery<AuditProvidersResponse>({
    queryKey: queryKeys.auditProviders(),
    queryFn: async () => {
      const res = await fetch("/api/v1/audits/providers");
      if (!res.ok) throw new Error(`http ${res.status}`);
      return res.json();
    },
    staleTime: Infinity,
  });
  const providerOptions = useMemo(
    () => providersQuery.data?.providers ?? [],
    [providersQuery.data],
  );
  const hostnameSuffixes = useMemo(
    () => providersQuery.data?.hostname_suffixes ?? {},
    [providersQuery.data],
  );

  const parsed = useMemo(() => parseURL(proxyUrl), [proxyUrl]);
  const tokens = parsed.ok ? parsed.tokens : [];
  const typeId = useId();

  // Reset key picks when the URL changes — stale tokens shouldn't linger.
  useEffect(() => {
    setCountryKey(NONE);
    setStateKey(NONE);
    setCityKey(NONE);
    setSessionKey(NONE);
    setWarningDismissed(false);
    setError(null);

    // Provider auto-prefill: only act when we have a parseable URL. Partially
    // typed URLs (intermediate states during user.type) and empty URLs leave
    // both `provider` and `lastAutoPrefillRef` alone — that lets a user's
    // custom override persist across cosmetic URL edits, while a URL pointing
    // at a *different* provider (different hostname) still wins because the
    // new auto-prefill differs from the recorded last one.
    if (!parsed.ok) return;
    let next = "";
    const i = tokens.indexOf("provider");
    if (i >= 0 && i < tokens.length - 1) {
      next = tokens[i + 1];
    } else {
      try {
        const u = new URL(proxyUrl);
        next = guessProviderFromHost(u.hostname, hostnameSuffixes);
      } catch {
        next = "";
      }
    }
    // We only commit a NON-EMPTY auto-prefill, and only when it differs from
    // the last one we committed. That handles two scenarios:
    //   - Mid-typing (URL host like "b", "br") yields next === "" — don't
    //     stomp on the user's existing pick.
    //   - Same host re-entered after a clear yields next === lastAutoPrefill
    //     — the user override (custom-pool) stays.
    if (next && next !== lastAutoPrefillRef.current) {
      lastAutoPrefillRef.current = next;
      setProvider(next);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [proxyUrl, hostnameSuffixes]);

  const countryValue = valueAfter(tokens, countryKey);
  const stateValue = valueAfter(tokens, stateKey);
  const cityValue = valueAfter(tokens, cityKey);
  const sessionValue = valueAfter(tokens, sessionKey);

  const sessionLike = findSessionLikeToken(tokens);
  const showSessionWarning = !!sessionLike && !sessionKey && !warningDismissed;

  const mut = useMutation({
    mutationFn: async () => {
      const body: Record<string, unknown> = {
        proxy_url: proxyUrl,
        expected_country: countryValue,
        expected_type: type,
        request_count: count,
      };
      if (stateValue) body.expected_state = stateValue;
      if (cityValue) body.expected_city = cityValue;
      if (sessionKey) body.session_key = sessionKey;
      if (provider) body.provider = provider;
      const res = await fetch("/api/v1/audits", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const b = await res.json().catch(() => ({}));
        throw new Error(b.detail ?? `http ${res.status}`);
      }
      return res.json() as Promise<{ id: string; status: string }>;
    },
    onSuccess: (data) => onStarted(data.id),
    onError: (err: Error) => setError(err.message),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!proxyUrl) {
      setError("URL must include a username (user@host)");
      return;
    }
    if (!parsed.ok) {
      setError(parsed.reason === "no-username"
        ? "URL must include a username (user@host)"
        : "URL is malformed");
      return;
    }
    if (!countryKey) {
      setError("country key is required");
      return;
    }
    if (countryValue === undefined) {
      setError("country key has no following value");
      return;
    }
    if (sessionKey && sessionValue === undefined) {
      setError("session key has no following value");
      return;
    }
    if (count < 100 || count > 1000) {
      setError("count must be in [100, 1000]");
      return;
    }
    mut.mutate();
  }

  function tokenSelect(
    label: string,
    value: string,
    onChange: (v: string) => void,
    derivedValue: string | undefined,
    required: boolean,
  ) {
    const id = label.replace(/\s+/g, "-").toLowerCase();
    const valueTestId = id.replace(/-key$/, "-value");
    return (
      <div className="flex items-center gap-3">
        <label htmlFor={id} className="block text-sm w-28">
          {label}{required ? " *" : ""}
        </label>
        <select
          id={id}
          aria-label={label}
          className="rounded border bg-transparent px-2 py-1 text-sm"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={tokens.length === 0}
        >
          <option value="">(none)</option>
          {tokens.map((t, i) => (
            <option key={`${t}-${i}`} value={t}>{t}</option>
          ))}
        </select>
        <span data-testid={valueTestId} className="text-xs text-muted-foreground">
          {value
            ? derivedValue !== undefined
              ? `→ value: ${derivedValue}`
              : "→ no following value"
            : ""}
        </span>
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4 max-w-2xl">
      <div>
        <label htmlFor="proxy-url" className="block text-sm">Proxy URL</label>
        <input
          id="proxy-url"
          aria-label="proxy url"
          className="w-full rounded border bg-transparent px-2 py-1"
          value={proxyUrl}
          onChange={(e) => setProxyUrl(e.target.value)}
          placeholder="http://user:pass@host:port"
        />
        <div data-testid="username-tokens" className="mt-1 text-xs text-muted-foreground">
          {tokens.length > 0
            ? <>Username tokens: {tokens.join(" · ")}</>
            : parsed.ok ? null : (
              proxyUrl ? <span className="text-red-500">{
                parsed.reason === "no-username"
                  ? "URL has no username"
                  : "URL is malformed"
              }</span> : null
            )}
        </div>
      </div>

      <div className="space-y-2">
        {tokenSelect("Country key", countryKey, setCountryKey, countryValue, true)}
        {tokenSelect("State key", stateKey, setStateKey, stateValue, false)}
        {tokenSelect("City key", cityKey, setCityKey, cityValue, false)}
        {tokenSelect("Session key", sessionKey, setSessionKey, sessionValue, false)}
        {sessionKey && sessionValue !== undefined && (
          <div className="ml-28 pl-3 text-xs text-muted-foreground">
            → will rotate per request (current value <code>{sessionValue}</code> is replaced)
          </div>
        )}
      </div>

      <div className="flex items-center gap-3">
        <label htmlFor="provider-combobox" className="block text-sm w-28">
          Provider
        </label>
        <Combobox
          value={provider}
          onChange={setProvider}
          options={providerOptions}
          placeholder="Select or type…"
          allowCustom
          ariaLabel="provider"
        />
      </div>

      <fieldset>
        <legend className="mb-2 text-xs uppercase tracking-wide text-muted-foreground">
          Expected type
        </legend>
        <RadioGroup
          value={type}
          onValueChange={(v) => setType(v as typeof type)}
          className="flex gap-4 text-sm"
        >
          {(["residential", "mobile", "datacenter"] as const).map((t) => {
            const id = `${typeId}-${t}`;
            return (
              <div key={t} className="flex items-center gap-2">
                <RadioGroupItem id={id} value={t} aria-label={`type: ${t}`} />
                <label htmlFor={id} className="cursor-pointer">{t}</label>
              </div>
            );
          })}
        </RadioGroup>
      </fieldset>

      <div>
        <label htmlFor="request-count" className="block text-sm">Request count</label>
        <input
          id="request-count"
          aria-label="request count"
          className="w-32 rounded border bg-transparent px-2 py-1"
          type="number"
          min={100}
          max={1000}
          value={count}
          onChange={(e) => setCount(Number(e.target.value))}
          onBlur={() => setCount((c) => Math.min(1000, Math.max(100, c || 100)))}
        />
      </div>

      {showSessionWarning && (
        <div
          data-testid="session-warning"
          className="rounded border border-amber-500/40 bg-amber-500/10 p-2 text-xs"
        >
          Looks like your URL has a <code>{sessionLike}</code> token. Rotating it
          per request usually gives a more honest audit.
          <span className="ml-2 inline-flex gap-2">
            <button
              type="button"
              onClick={() => {
                setSessionKey(sessionLike!);
                setWarningDismissed(true);
              }}
              className="rounded border px-2 py-0.5"
            >
              Use it
            </button>
            <button
              type="button"
              onClick={() => setWarningDismissed(true)}
              className="rounded border px-2 py-0.5"
            >
              Run anyway
            </button>
          </span>
        </div>
      )}

      {error && <div className="text-xs text-red-500">{error}</div>}

      <button
        type="submit"
        disabled={mut.isPending}
        className="rounded bg-primary px-4 py-1.5 text-sm text-primary-foreground disabled:opacity-50"
      >
        Start audit
      </button>
    </form>
  );
}
