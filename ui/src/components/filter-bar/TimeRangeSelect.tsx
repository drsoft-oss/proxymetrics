import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Calendar } from "lucide-react";
import { useFilter } from "@/hooks/useFilter";

const PRESETS = [
  { label: "Last 1h", ms: 60 * 60 * 1000 },
  { label: "Last 24h", ms: 24 * 60 * 60 * 1000 },
  { label: "Last 7 days", ms: 7 * 24 * 60 * 60 * 1000 },
  { label: "Last 30 days", ms: 30 * 24 * 60 * 60 * 1000 },
];

export function TimeRangeSelect() {
  const { filter, setFilter } = useFilter();

  const apply = (ms: number) => {
    const to = new Date();
    const from = new Date(to.getTime() - ms);
    setFilter({ from: from.toISOString(), to: to.toISOString() });
  };

  const labelFromFilter = (() => {
    if (!filter.from || !filter.to) return "Select range";
    const span = new Date(filter.to).getTime() - new Date(filter.from).getTime();
    for (const p of PRESETS) if (Math.abs(p.ms - span) < 60_000) return p.label;
    return "Custom";
  })();

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2">
          <Calendar className="h-3.5 w-3.5" />
          {labelFromFilter}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-1">
        <div className="flex flex-col">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              onClick={() => apply(p.ms)}
              className="rounded-sm px-3 py-1.5 text-left text-sm hover:bg-accent"
            >
              {p.label}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}
