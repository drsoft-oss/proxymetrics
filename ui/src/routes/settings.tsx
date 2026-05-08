import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/shell/ComingSoon";

export const Route = createFileRoute("/settings")({
  component: () => <ComingSoon name="Settings" />,
});
