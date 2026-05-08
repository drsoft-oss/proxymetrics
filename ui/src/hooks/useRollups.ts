import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { RollupsResponse } from "@/types/api";

export function useRollups(level: "1min" | "1hour" | "1day", filter: Filter) {
  return useQuery({
    queryKey: queryKeys.rollups(level, filter),
    queryFn: () => apiGet<RollupsResponse>(`/api/v1/rollups/${level}`, filterToParams(filter)),
  });
}
