import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import type { RunDetail, RunSummary } from "@/types/audit";

export function useAudits() {
  return useQuery({
    queryKey: queryKeys.audits(),
    queryFn: () => apiGet<RunSummary[]>("/api/v1/audits"),
    staleTime: 10_000,
  });
}

export function useAuditDetail(id: string | null) {
  return useQuery({
    queryKey: queryKeys.audit(id ?? "_"),
    queryFn: () => apiGet<RunDetail>(`/api/v1/audits/${id}`),
    enabled: id !== null,
    staleTime: 5_000,
  });
}
