import * as React from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";

type Props = {
  value: string;
  onChange: (next: string) => void;
  options: string[];
  placeholder?: string;
  /** When true, an unmatched query is committable as the typed string. */
  allowCustom?: boolean;
  ariaLabel: string;
  className?: string;
  disabled?: boolean;
};

export function Combobox({
  value,
  onChange,
  options,
  placeholder,
  allowCustom = false,
  ariaLabel,
  className,
  disabled,
}: Props) {
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");

  // Reset the query whenever the popover closes — the next open should start
  // with the option list unfiltered.
  React.useEffect(() => {
    if (!open) setQuery("");
  }, [open]);

  const trimmed = query.trim();
  const exactMatchExists = options.some(
    (o) => o.toLowerCase() === trimmed.toLowerCase(),
  );
  const showCreateRow = allowCustom && trimmed.length > 0 && !exactMatchExists;

  function commit(next: string) {
    onChange(next);
    setOpen(false);
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          role="combobox"
          aria-label={ariaLabel}
          aria-expanded={open}
          disabled={disabled}
          className={cn(
            "flex h-8 w-56 items-center justify-between rounded border bg-transparent px-2 text-sm",
            !value && "text-muted-foreground",
            className,
          )}
        >
          <span className="truncate">{value || placeholder || "Select…"}</span>
          <span aria-hidden className="ml-2 opacity-50">▾</span>
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-0" align="start">
        <Command shouldFilter label={`${ariaLabel} search`}>
          <CommandInput
            value={query}
            onValueChange={setQuery}
            placeholder={placeholder ?? "Search…"}
          />
          <CommandList>
            <CommandEmpty>{showCreateRow ? null : "No matches."}</CommandEmpty>
            {options.map((opt) => (
              <CommandItem key={opt} value={opt} onSelect={() => commit(opt)}>
                {opt}
              </CommandItem>
            ))}
            {showCreateRow && (
              <CommandItem
                key={`__create__:${trimmed}`}
                value={trimmed}
                onSelect={() => commit(trimmed)}
              >
                Create &ldquo;{trimmed}&rdquo;
              </CommandItem>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
