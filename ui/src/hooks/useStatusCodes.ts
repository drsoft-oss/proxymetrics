import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { StatusCodesResponse } from "@/types/api";

export function useStatusCodes(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.statusCodes(filter),
    queryFn: () => apiGet<StatusCodesResponse>("/api/v1/status-codes", filterToParams(filter)),
  });
}
