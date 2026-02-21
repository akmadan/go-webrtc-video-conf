"use client";

import { useEffect, useRef } from "react";

interface VideoTileProps {
  stream: MediaStream | null;
  label: string;
  muted?: boolean;
  isLocal?: boolean;
}

export function VideoTile({
  stream,
  label,
  muted = false,
  isLocal = false,
}: VideoTileProps) {
  const ref = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    if (!ref.current) return;
    ref.current.srcObject = stream;
  }, [stream]);

  return (
    <div className="glass relative overflow-hidden rounded-2xl">
      <video
        ref={ref}
        autoPlay
        playsInline
        muted={muted}
        className="h-[260px] w-full bg-black object-cover md:h-[320px]"
      />
      <div className="absolute inset-x-0 bottom-0 flex items-center justify-between bg-gradient-to-t from-black/70 to-transparent px-3 py-2 text-sm">
        <span>{label}</span>
        {isLocal && (
          <span className="rounded-full bg-black/50 px-2 py-1 text-xs text-muted">
            You
          </span>
        )}
      </div>
      {!stream && (
        <div className="absolute inset-0 grid place-items-center text-sm text-muted">
          Waiting for video...
        </div>
      )}
    </div>
  );
}

