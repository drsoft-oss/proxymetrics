import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { ProviderDetailResponse } from "@/types/api";

export function useProviderDetail(profileId: string | null, filter: Filter) {
  return useQuery({
    queryKey: queryKeys.providerDetail(profileId ?? "", filter),
    queryFn: () => apiGet<ProviderDetailResponse>(`/api/v1/providers/${profileId}`, filterToParams(filter)),
    enabled: profileId != null,
  });
}
