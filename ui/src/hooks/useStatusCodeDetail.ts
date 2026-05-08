import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { StatusCodeDetailResponse } from "@/types/api";

export function useStatusCodeDetail(code: number | null, filter: Filter) {
  return useQuery({
    queryKey: queryKeys.statusCodeDetail(code ?? -1, filter),
    queryFn: () => apiGet<StatusCodeDetailResponse>(`/api/v1/status-codes/${code}`, filterToParams(filter)),
    enabled: code != null,
  });
}
