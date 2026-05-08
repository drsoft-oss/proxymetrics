import { TimeRangeSelect } from "./TimeRangeSelect";
import { MultiSelectChip } from "./MultiSelectChip";
import { useFilter } from "@/hooks/useFilter";
import { useDimensions } from "@/hooks/useDimensions";

const STATUS_CLASS_OPTIONS = [
  { value: "2xx", label: "2xx" },
  { value: "3xx", label: "3xx" },
  { value: "4xx", label: "4xx" },
  { value: "5xx", label: "5xx" },
];

export function FilterBar() {
  const { filter, setFilter } = useFilter();
  const dims = useDimensions(filter.from, filter.to);

  const ready = dims.isSuccess;
  const opts = (vals: string[] | undefined) =>
    (vals ?? []).map((v) => ({ value: v, label: v }));

  const profileOptions = (dims.data?.profile_id ?? []).map((p) => ({ value: p.id, label: p.name }));

  return (
    <div className="sticky top-0 z-10 flex flex-wrap items-center gap-2 border-b border-border bg-background px-4 py-2">
      <TimeRangeSelect />
      <MultiSelectChip
        label="Profiles"
        options={profileOptions}
        selected={filter.profile_id ?? []}
        onChange={(next) => setFilter({ profile_id: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Vendors"
        options={opts(dims.data?.vendor)}
        selected={filter.vendor ?? []}
        onChange={(next) => setFilter({ vendor: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Types"
        options={opts(dims.data?.type)}
        selected={filter.type ?? []}
        onChange={(next) => setFilter({ type: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Regions"
        options={opts(dims.data?.region)}
        selected={filter.region ?? []}
        onChange={(next) => setFilter({ region: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Teams"
        options={opts(dims.data?.team)}
        selected={filter.team ?? []}
        onChange={(next) => setFilter({ team: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Projects"
        options={opts(dims.data?.project)}
        selected={filter.project ?? []}
        onChange={(next) => setFilter({ project: next.length ? next : undefined })}
        disabled={!ready}
      />
      <MultiSelectChip
        label="Status classes"
        options={STATUS_CLASS_OPTIONS}
        selected={filter.status_class ?? []}
        onChange={(next) => setFilter({ status_class: next.length ? next : undefined })}
      />
    </div>
  );
}
