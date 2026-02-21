"use client";

interface CallControlsProps {
  isMicEnabled: boolean;
  isCamEnabled: boolean;
  onToggleMic: () => void;
  onToggleCam: () => void;
  onLeave: () => void;
}

export function CallControls({
  isMicEnabled,
  isCamEnabled,
  onToggleMic,
  onToggleCam,
  onLeave,
}: CallControlsProps) {
  const base =
    "rounded-xl border border-white/15 px-4 py-2 text-sm transition hover:border-white/35";

  return (
    <div className="glass flex flex-wrap items-center justify-center gap-3 rounded-2xl p-4">
      <button onClick={onToggleMic} className={base}>
        {isMicEnabled ? "Mute Mic" : "Unmute Mic"}
      </button>
      <button onClick={onToggleCam} className={base}>
        {isCamEnabled ? "Turn Camera Off" : "Turn Camera On"}
      </button>
      <button
        onClick={onLeave}
        className="rounded-xl border border-red-400/40 bg-red-500/20 px-4 py-2 text-sm transition hover:bg-red-500/35"
      >
        Leave Call
      </button>
    </div>
  );
}

