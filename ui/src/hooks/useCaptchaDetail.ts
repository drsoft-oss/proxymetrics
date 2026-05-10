import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { filterToParams, type Filter } from "@/lib/filters";
import { queryKeys } from "@/lib/queryKeys";
import type { CaptchaDetailResponse, CaptchaKind } from "@/types/api";

export function useCaptchaDetail(kind: CaptchaKind | null, filter: Filter) {
  return useQuery({
    queryKey: queryKeys.captchaDetail(kind ?? "", filter),
    queryFn: () => apiGet<CaptchaDetailResponse>(`/api/v1/captchas/${kind}`, filterToParams(filter)),
    enabled: kind != null,
  });
}
