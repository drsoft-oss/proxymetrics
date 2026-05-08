import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { StatusCodeDistributionResponse } from "@/types/api";

export function useStatusCodeDistribution(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.statusCodeDistribution(filter),
    queryFn: () => apiGet<StatusCodeDistributionResponse>("/api/v1/status-codes/distribution", filterToParams(filter)),
  });
}
