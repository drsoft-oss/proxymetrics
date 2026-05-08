export function Sparkline({
  data,
  width = 80,
  height = 22,
  stroke = "hsl(var(--muted-foreground))",
}: {
  data: number[];
  width?: number;
  height?: number;
  stroke?: string;
}) {
  if (data.length <= 1) {
    return <span className="text-muted-foreground">—</span>;
  }
  const max = Math.max(...data);
  if (max === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  const stepX = data.length > 1 ? width / (data.length - 1) : 0;
  const points = data.map((v, i) => {
    const x = i * stepX;
    const y = height - (v / max) * (height - 2) - 1;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const d = "M" + points.join(" L");
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-label="sparkline">
      <path d={d} fill="none" stroke={stroke} strokeWidth="1.25" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  );
}
