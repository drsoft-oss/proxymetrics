import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { AuditDetailView } from "@/components/audits/AuditDetailView";

export const Route = createFileRoute("/audits/$id")({
  component: DetailRoute,
});

function DetailRoute() {
  const { id } = Route.useParams();
  const nav = useNavigate();
  return (
    <AuditDetailView
      runId={id}
      onReRun={(runId) => nav({ to: "/audits", search: { run: runId } })}
    />
  );
}
