import { useNavigate, useSearch } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { defaultFilter, paramsToFilter, type Filter } from "@/lib/filters";

// useFilter reads the current Filter from URL search params and gives back a setter
// that pushes partial updates into the URL.
export function useFilter(): { filter: Filter; setFilter: (partial: Partial<Filter>) => void } {
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as Record<string, unknown>;

  const filter = useMemo(() => {
    const p = new URLSearchParams();
    for (const [k, v] of Object.entries(search)) {
      if (v == null) continue;
      if (Array.isArray(v)) {
        for (const item of v) p.append(k, String(item));
      } else {
        p.set(k, String(v));
      }
    }
    return paramsToFilter(p, defaultFilter());
  }, [search]);

  const setFilter = useCallback(
    (partial: Partial<Filter>) => {
      navigate({
        to: ".",
        search: (prev: Record<string, unknown>) => ({ ...prev, ...partial }),
      } as never);
    },
    [navigate]
  );

  return { filter, setFilter };
}
