import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { OverviewResponse } from "@/types/api";

export function useOverview(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.overview(filter),
    queryFn: () => apiGet<OverviewResponse>("/api/v1/overview", filterToParams(filter)),
  });
}
