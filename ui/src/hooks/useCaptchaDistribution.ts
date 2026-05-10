import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { CaptchaDistributionResponse } from "@/types/api";

export function useCaptchaDistribution(filter: Filter) {
  return useQuery({
    queryKey: queryKeys.captchaDistribution(filter),
    queryFn: () => apiGet<CaptchaDistributionResponse>("/api/v1/captchas/distribution", filterToParams(filter)),
  });
}
