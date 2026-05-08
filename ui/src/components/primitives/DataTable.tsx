import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export type Column<T> = {
  key: string;
  header: string;
  sortable?: boolean;
  align?: "left" | "right";
  render: (row: T) => ReactNode;
  sortValue?: (row: T) => number | string;
};

export type SortBy = { key: string; dir: "asc" | "desc" };

export type DataTableProps<T> = {
  columns: Column<T>[];
  rows: T[];
  isLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  onRowClick?: (row: T) => void;
  sortBy?: SortBy;
  pageSize?: number;
  groupBy?: (row: T) => string;
  emptyMessage?: string;
};

export function DataTable<T>({
  columns,
  rows,
  isLoading,
  error,
  onRetry,
  onRowClick,
  sortBy,
  pageSize = 25,
  groupBy,
  emptyMessage = "No data.",
}: DataTableProps<T>) {
  const [activeSort, setActiveSort] = useState<SortBy | undefined>(sortBy);
  const [page, setPage] = useState(0);

  // Keep internal state in sync if parent supplies a new default.
  useEffect(() => {
    setActiveSort(sortBy);
  }, [sortBy?.key, sortBy?.dir]);

  const sorted = useMemo(() => {
    if (!activeSort) return rows;
    const col = columns.find((c) => c.key === activeSort.key);
    if (!col?.sortValue) return rows;
    const dir = activeSort.dir === "asc" ? 1 : -1;
    return [...rows].sort((a, b) => {
      const av = col.sortValue!(a);
      const bv = col.sortValue!(b);
      if (av < bv) return -1 * dir;
      if (av > bv) return 1 * dir;
      return 0;
    });
  }, [rows, activeSort, columns]);

  const groups = useMemo(() => {
    if (!groupBy) return null;
    const m = new Map<string, T[]>();
    for (const r of sorted) {
      const k = groupBy(r);
      const arr = m.get(k) ?? [];
      arr.push(r);
      m.set(k, arr);
    }
    return [...m.entries()];
  }, [sorted, groupBy]);

  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize));
  // Clamp the active page so a row-count drop (filter change) doesn't strand the
  // user on a now-empty page.
  const safePage = Math.min(page, pageCount - 1);
  const pageStart = safePage * pageSize;
  const pageRows = groups ? sorted : sorted.slice(pageStart, pageStart + pageSize);

  if (isLoading) {
    return (
      <div className="rounded-md border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border">
              {columns.map((c) => (
                <th
                  key={c.key}
                  className={cn(
                    "px-3 py-2 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground",
                    c.align === "right" && "text-right"
                  )}
                >
                  {c.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 5 }).map((_, i) => (
              <tr key={i} data-testid="dt-skeleton-row" className="border-b border-border">
                {columns.map((c) => (
                  <td key={c.key} className="px-3 py-2">
                    <Skeleton className="h-4 w-full" />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md border border-destructive/40 bg-destructive/5 p-4 text-sm text-destructive">
        <div className="mb-2 font-medium">Couldn't load: {error.message}</div>
        {onRetry && (
          <Button size="sm" variant="outline" onClick={onRetry}>
            Retry
          </Button>
        )}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center rounded-md border border-border text-sm text-muted-foreground">
        {emptyMessage}
      </div>
    );
  }

  const handleHeaderClick = (col: Column<T>) => {
    if (!col.sortable) return;
    setActiveSort((prev) => {
      if (!prev || prev.key !== col.key) return { key: col.key, dir: "desc" };
      return { key: col.key, dir: prev.dir === "asc" ? "desc" : "asc" };
    });
  };

  return (
    <div className="rounded-md border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border">
            {columns.map((c) => (
              <th
                key={c.key}
                className={cn(
                  "px-3 py-2 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground",
                  c.align === "right" && "text-right"
                )}
              >
                {c.sortable ? (
                  <button
                    type="button"
                    onClick={() => handleHeaderClick(c)}
                    className="hover:text-foreground"
                  >
                    {c.header}
                    {activeSort?.key === c.key && (activeSort.dir === "asc" ? " ▲" : " ▼")}
                  </button>
                ) : (
                  c.header
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {groups
            ? groups.flatMap(([key, subRows]) => [
                <tr key={`group-${key}`} className="bg-accent/40">
                  <td
                    colSpan={columns.length}
                    className="px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground"
                  >
                    {key}
                  </td>
                </tr>,
                ...subRows.map((row, i) => (
                  <tr
                    key={`row-${key}-${i}`}
                    data-testid="dt-row"
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    className={cn(
                      "border-b border-border last:border-0",
                      onRowClick && "cursor-pointer hover:bg-accent/30"
                    )}
                  >
                    {columns.map((c) => (
                      <td
                        key={c.key}
                        className={cn("px-3 py-2", c.align === "right" && "text-right")}
                      >
                        {c.render(row)}
                      </td>
                    ))}
                  </tr>
                )),
              ])
            : pageRows.map((row, i) => (
                <tr
                  key={`row-${i}`}
                  data-testid="dt-row"
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(
                    "border-b border-border last:border-0",
                    onRowClick && "cursor-pointer hover:bg-accent/30"
                  )}
                >
                  {columns.map((c) => (
                    <td
                      key={c.key}
                      className={cn("px-3 py-2", c.align === "right" && "text-right")}
                    >
                      {c.render(row)}
                    </td>
                  ))}
                </tr>
              ))}
        </tbody>
      </table>

      {!groups && sorted.length > pageSize && (
        <div className="flex items-center justify-between border-t border-border px-3 py-2 text-xs text-muted-foreground">
          <span>
            Page {safePage + 1} / {pageCount} · {sorted.length} total
          </span>
          <div className="flex gap-1">
            <Button
              size="sm"
              variant="outline"
              disabled={safePage === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
            >
              Prev
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={safePage >= pageCount - 1}
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
