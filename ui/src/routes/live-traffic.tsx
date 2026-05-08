import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useLiveBuffer } from "@/hooks/useLiveBuffer";
import { LiveKpiStrip } from "@/components/live/LiveKpiStrip";
import { ThroughputChart } from "@/components/live/ThroughputChart";
import { StatusMixDonut } from "@/components/live/StatusMixDonut";
import { LatencyAggregate } from "@/components/live/LatencyAggregate";
import { TopProvidersBar } from "@/components/live/TopProvidersBar";
import { TopTargetsBar } from "@/components/live/TopTargetsBar";
import { WindowPill, type LiveWindow } from "@/components/live/WindowPill";

export const Route = createFileRoute("/live-traffic")({
  component: LiveTrafficPage,
});

const STORAGE_KEY = "pm.live.window";

function readStoredWindow(): LiveWindow {
  if (typeof window === "undefined") return 60;
  const raw = window.localStorage.getItem(STORAGE_KEY);
  const n = raw ? Number(raw) : 60;
  return n === 30 || n === 60 || n === 300 ? n : 60;
}

function LiveTrafficPage() {
  const [windowSec, setWindowSec] = useState<LiveWindow>(() => readStoredWindow());
  const [paused, setPaused] = useState(false);
  const { aggregates } = useLiveBuffer({ windowSec, paused });

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, String(windowSec));
  }, [windowSec]);

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">Live Traffic</h1>
        <WindowPill
          value={windowSec}
          onChange={setWindowSec}
          paused={paused}
          onTogglePause={() => setPaused((p) => !p)}
        />
      </div>
      <LiveKpiStrip a={aggregates} />
      <ThroughputChart a={aggregates} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <StatusMixDonut a={aggregates} />
        <LatencyAggregate a={aggregates} />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <TopProvidersBar a={aggregates} />
        <TopTargetsBar a={aggregates} />
      </div>
    </div>
  );
}
