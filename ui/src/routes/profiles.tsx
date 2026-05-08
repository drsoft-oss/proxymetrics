import { createFileRoute } from "@tanstack/react-router";
import { ProfilesPage } from "@/components/profiles/ProfilesPage";

export const Route = createFileRoute("/profiles")({
  component: ProfilesPage,
});
