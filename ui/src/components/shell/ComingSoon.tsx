import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

export function ComingSoon({ name }: { name: string }) {
  return (
    <div className="flex min-h-[60vh] items-center justify-center p-6">
      <Card className="max-w-md">
        <CardHeader>
          <CardTitle>{name}</CardTitle>
          <CardDescription>Available in the next release.</CardDescription>
        </CardHeader>
      </Card>
    </div>
  );
}
