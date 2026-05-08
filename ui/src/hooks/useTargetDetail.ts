import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { TargetDetailResponse } from "@/types/api";

export function useTargetDetail(host: string | null, filter: Filter) {
  return useQuery({
    queryKey: queryKeys.targetDetail(host ?? "", filter),
    queryFn: () => apiGet<TargetDetailResponse>(`/api/v1/targets/${encodeURIComponent(host ?? "")}`, filterToParams(filter)),
    enabled: host != null,
  });
}
