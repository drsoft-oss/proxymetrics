import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { CaptchasResponse } from "@/types/api";

export function useCaptchas(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.captchas(filter),
    queryFn: () => apiGet<CaptchasResponse>("/api/v1/captchas", filterToParams(filter)),
  });
}
