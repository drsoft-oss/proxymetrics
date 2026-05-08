import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import type { ProfileWithUsage } from "@/types/profile";

export function useProfiles() {
  const params = new URLSearchParams({ with: "usage" });
  return useQuery({
    queryKey: queryKeys.profiles(),
    queryFn: () => apiGet<ProfileWithUsage[]>("/api/v1/profiles", params),
    staleTime: 30_000,
  });
}
