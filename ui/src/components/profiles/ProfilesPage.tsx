import { useState } from "react";
import { ProfilesTable } from "./ProfilesTable";
import { ProfileDetailDrawer } from "./ProfileDetailDrawer";
import { useProfiles } from "@/hooks/useProfiles";

export function ProfilesPage() {
  const profiles = useProfiles();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selected = profiles.data?.find((p) => p.id === selectedId) ?? null;

  return (
    <div className="space-y-4 p-4">
      <header>
        <h1 className="text-base font-semibold">Profiles</h1>
        <p className="text-xs text-muted-foreground">
          Profiles are auto-discovered from proxy traffic. Click a row for details.
        </p>
      </header>

      <ProfilesTable
        rows={profiles.data ?? []}
        loading={profiles.isLoading}
        onRowClick={(id) => setSelectedId(id)}
      />

      <ProfileDetailDrawer
        profile={selected}
        open={selected !== null}
        onClose={() => setSelectedId(null)}
      />
    </div>
  );
}
