import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { TargetsResponse } from "@/types/api";

export function useTargets(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.targets(filter),
    queryFn: () => apiGet<TargetsResponse>("/api/v1/targets", filterToParams(filter)),
  });
}
