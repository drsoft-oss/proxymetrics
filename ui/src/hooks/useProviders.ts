import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { ProvidersResponse } from "@/types/api";

export function useProviders(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.providers(filter),
    queryFn: () => apiGet<ProvidersResponse>("/api/v1/providers", filterToParams(filter)),
  });
}
