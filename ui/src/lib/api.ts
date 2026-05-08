export class ApiError extends Error {
  constructor(public status: number, public code: string, public detail: string) {
    super(`${code}: ${detail}`);
    this.name = "ApiError";
  }
}

const TIMEOUT_MS = 15_000;

export async function apiGet<T>(path: string, params?: URLSearchParams): Promise<T> {
  const url = params && params.toString() ? `${path}?${params.toString()}` : path;
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), TIMEOUT_MS);
  try {
    const res = await fetch(url, {
      method: "GET",
      headers: { Accept: "application/json" },
      signal: ctrl.signal,
    });
    if (!res.ok) {
      let code = "http_" + res.status;
      let detail = res.statusText;
      try {
        const body = (await res.json()) as { error?: string; detail?: string };
        if (body.error) code = body.error;
        if (body.detail) detail = body.detail;
      } catch {
        // non-JSON body — keep statusText
      }
      throw new ApiError(res.status, code, detail);
    }
    return (await res.json()) as T;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if ((err as Error).name === "AbortError") {
      throw new ApiError(0, "timeout", "request timed out");
    }
    throw new ApiError(0, "network", (err as Error).message);
  } finally {
    clearTimeout(timer);
  }
}
