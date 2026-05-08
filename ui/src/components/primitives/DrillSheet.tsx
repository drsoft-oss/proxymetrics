import * as React from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export type DrillSheetProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  children: React.ReactNode;
};

export function DrillSheet({ open, onClose, title, subtitle, children }: DrillSheetProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/60" />
        <Dialog.Content
          className={cn(
            "fixed inset-y-0 right-0 z-50 flex w-full max-w-md flex-col border-l border-border bg-background shadow-2xl outline-none",
            "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right"
          )}
        >
          <header className="flex items-start justify-between border-b border-border px-4 py-3">
            <div>
              <Dialog.Title className="text-base font-semibold leading-tight">{title}</Dialog.Title>
              {subtitle && <Dialog.Description className="text-xs text-muted-foreground">{subtitle}</Dialog.Description>}
            </div>
            <Dialog.Close
              aria-label="Close"
              className="rounded-md p-1 text-muted-foreground hover:bg-accent"
            >
              <X className="h-4 w-4" />
            </Dialog.Close>
          </header>
          <div className="flex-1 overflow-y-auto">{children}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

export function DrillSection({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <section className="border-b border-border px-4 py-3 last:border-0">
      {title && (
        <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </h3>
      )}
      {children}
    </section>
  );
}
