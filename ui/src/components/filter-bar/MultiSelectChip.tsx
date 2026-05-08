import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export type MultiSelectOption = { value: string; label: string };

export function MultiSelectChip({
  label,
  options,
  selected,
  onChange,
  disabled,
}: {
  label: string;
  options: MultiSelectOption[];
  selected: string[];
  onChange: (next: string[]) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);

  const trigger =
    selected.length === 0 ? `All ${label.toLowerCase()}` : `${selected.length} ${label.toLowerCase()}`;

  return (
    <Popover open={open} onOpenChange={(v) => !disabled && setOpen(v)}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          disabled={disabled}
          className={cn("gap-2", disabled && "opacity-50")}
          title={disabled ? `Loading ${label.toLowerCase()}…` : undefined}
        >
          {trigger}
          <ChevronDown className="h-3 w-3" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-0">
        <Command>
          <CommandInput placeholder={`Search ${label.toLowerCase()}...`} />
          <CommandEmpty>No options.</CommandEmpty>
          <CommandList>
            {options.map((opt) => {
              const isSelected = selected.includes(opt.value);
              return (
                <CommandItem
                  key={opt.value}
                  onSelect={() => {
                    onChange(
                      isSelected ? selected.filter((s) => s !== opt.value) : [...selected, opt.value]
                    );
                  }}
                >
                  <Check className={cn("mr-2 h-4 w-4", isSelected ? "opacity-100" : "opacity-0")} />
                  {opt.label}
                </CommandItem>
              );
            })}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
