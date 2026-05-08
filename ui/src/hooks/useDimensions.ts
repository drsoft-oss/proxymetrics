import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import type { DimensionsResponse } from "@/types/api";

// staleTime matches the backend cache TTL so we don't pound the endpoint while
// chips are open.
const STALE_MS = 30_000;

export function useDimensions(from: string, to: string) {
  const params = new URLSearchParams({ from, to });
  return useQuery({
    queryKey: queryKeys.dimensions(from, to),
    queryFn: () => apiGet<DimensionsResponse>("/api/v1/dimensions", params),
    staleTime: STALE_MS,
  });
}
