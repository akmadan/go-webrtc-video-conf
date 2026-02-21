"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { CallControls } from "@/components/CallControls";
import { VideoTile } from "@/components/VideoTile";
import { useVideoRoom } from "@/hooks/useVideoRoom";

interface RoomClientProps {
  roomId: string;
  displayName: string;
}

export function RoomClient({ roomId, displayName }: RoomClientProps) {
  const [joined, setJoined] = useState(false);
  const [lobbyName, setLobbyName] = useState(displayName || "You");
  const [previewStream, setPreviewStream] = useState<MediaStream | null>(null);
  const [previewMicId, setPreviewMicId] = useState("");
  const [previewCamId, setPreviewCamId] = useState("");
  const [previewMics, setPreviewMics] = useState<Array<{ id: string; label: string }>>([]);
  const [previewCams, setPreviewCams] = useState<Array<{ id: string; label: string }>>([]);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const previewStreamRef = useRef<MediaStream | null>(null);

  useEffect(() => {
    const setupPreview = async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
        previewStreamRef.current = stream;
        setPreviewStream(stream);
        const devices = await navigator.mediaDevices.enumerateDevices();
        const mics = devices
          .filter((d) => d.kind === "audioinput")
          .map((d, i) => ({ id: d.deviceId, label: d.label || `Microphone ${i + 1}` }));
        const cams = devices
          .filter((d) => d.kind === "videoinput")
          .map((d, i) => ({ id: d.deviceId, label: d.label || `Camera ${i + 1}` }));
        setPreviewMics(mics);
        setPreviewCams(cams);
        setPreviewMicId((prev) => prev || mics[0]?.id || "");
        setPreviewCamId((prev) => prev || cams[0]?.id || "");
      } catch (err) {
        setPreviewError(err instanceof Error ? err.message : "Failed to access devices");
      }
    };
    setupPreview();
    return () => {
      if (previewStreamRef.current) {
        previewStreamRef.current.getTracks().forEach((t) => t.stop());
      }
    };
  }, []);

  const switchPreviewDevices = async (micId: string, camId: string) => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: micId ? { deviceId: { exact: micId } } : true,
        video: camId ? { deviceId: { exact: camId } } : true,
      });
      if (previewStreamRef.current) {
        previewStreamRef.current.getTracks().forEach((t) => t.stop());
      }
      previewStreamRef.current = stream;
      setPreviewStream(stream);
      setPreviewMicId(micId);
      setPreviewCamId(camId);
      setPreviewError(null);
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : "Failed to switch preview device");
    }
  };

  if (!joined) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-5 px-4 py-6 md:px-8">
        <header className="glass flex items-center justify-between rounded-2xl px-4 py-3">
          <div>
            <h1 className="text-lg font-medium">Pre-Join Lobby</h1>
            <p className="text-sm text-muted">Room: {roomId}</p>
          </div>
        </header>
        {previewError && (
          <div className="rounded-xl border border-red-400/40 bg-red-500/15 px-4 py-3 text-sm">
            {previewError}
          </div>
        )}
        <VideoTile stream={previewStream} label={lobbyName || "You"} muted isLocal />
        <section className="glass grid gap-3 rounded-2xl p-4 md:grid-cols-3">
          <label className="space-y-1">
            <span className="text-xs text-muted">Name</span>
            <input
              value={lobbyName}
              onChange={(e) => setLobbyName(e.target.value)}
              className="w-full rounded-lg border border-white/15 bg-black/25 px-3 py-2 text-sm"
            />
          </label>
          <label className="space-y-1">
            <span className="text-xs text-muted">Microphone</span>
            <select
              value={previewMicId}
              onChange={(e) => switchPreviewDevices(e.target.value, previewCamId)}
              className="w-full rounded-lg border border-white/15 bg-black/25 px-3 py-2 text-sm"
            >
              {previewMics.map((mic) => (
                <option key={mic.id} value={mic.id}>
                  {mic.label}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-1">
            <span className="text-xs text-muted">Camera</span>
            <select
              value={previewCamId}
              onChange={(e) => switchPreviewDevices(previewMicId, e.target.value)}
              className="w-full rounded-lg border border-white/15 bg-black/25 px-3 py-2 text-sm"
            >
              {previewCams.map((cam) => (
                <option key={cam.id} value={cam.id}>
                  {cam.label}
                </option>
              ))}
            </select>
          </label>
        </section>
        <div className="flex justify-end">
          <button
            onClick={() => {
              if (previewStreamRef.current) {
                previewStreamRef.current.getTracks().forEach((t) => t.stop());
                previewStreamRef.current = null;
              }
              setJoined(true);
            }}
            className="rounded-xl bg-accent px-5 py-3 font-medium text-black transition hover:bg-accent-2"
          >
            Join Room
          </button>
        </div>
      </main>
    );
  }

  return (
    <ActiveRoomClient
      roomId={roomId}
      displayName={lobbyName || "You"}
      preferredMicId={previewMicId}
      preferredCameraId={previewCamId}
    />
  );
}

interface ActiveRoomClientProps {
  roomId: string;
  displayName: string;
  preferredMicId?: string;
  preferredCameraId?: string;
}

function ActiveRoomClient({
  roomId,
  displayName,
  preferredMicId,
  preferredCameraId,
}: ActiveRoomClientProps) {
  const router = useRouter();
  const {
    localStream,
    remotePeers,
    status,
    error,
    isCamEnabled,
    isMicEnabled,
    microphones,
    cameras,
    selectedMicId,
    selectedCameraId,
    switchDevices,
    toggleCam,
    toggleMic,
    leaveRoom,
  } = useVideoRoom({
    roomId,
    displayName,
    preferredMicId,
    preferredCameraId,
  });

  const qualityBadgeClass = (quality?: string) => {
    if (quality === "good") return "bg-emerald-500/20 text-emerald-300 border-emerald-400/40";
    if (quality === "fair") return "bg-amber-500/20 text-amber-300 border-amber-400/40";
    if (quality === "poor") return "bg-red-500/20 text-red-300 border-red-400/40";
    return "bg-white/10 text-muted border-white/20";
  };

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-5 px-4 py-6 md:px-8">
      <header className="glass flex flex-wrap items-center justify-between gap-3 rounded-2xl px-4 py-3">
        <div>
          <h1 className="text-lg font-medium">Room: {roomId}</h1>
          <p className="text-sm text-muted">
            Status:{" "}
            <span className="text-foreground capitalize">{status}</span> · Peers:{" "}
            {remotePeers.length}
          </p>
          <p className="text-xs text-muted">Name: {displayName}</p>
        </div>
        <button
          onClick={() => {
            leaveRoom();
            router.push("/");
          }}
          className="rounded-xl border border-white/15 px-3 py-2 text-sm transition hover:border-white/35"
        >
          Back to Home
        </button>
      </header>

      {error && (
        <div className="rounded-xl border border-red-400/40 bg-red-500/15 px-4 py-3 text-sm">
          {error}
        </div>
      )}

      <section className="glass grid gap-3 rounded-2xl p-4 md:grid-cols-2">
        <label className="space-y-1">
          <span className="text-xs text-muted">Microphone</span>
          <select
            value={selectedMicId}
            onChange={(e) => switchDevices(e.target.value, selectedCameraId)}
            className="w-full rounded-lg border border-white/15 bg-black/25 px-3 py-2 text-sm"
          >
            {microphones.map((mic) => (
              <option key={mic.id} value={mic.id}>
                {mic.label}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-1">
          <span className="text-xs text-muted">Camera</span>
          <select
            value={selectedCameraId}
            onChange={(e) => switchDevices(selectedMicId, e.target.value)}
            className="w-full rounded-lg border border-white/15 bg-black/25 px-3 py-2 text-sm"
          >
            {cameras.map((cam) => (
              <option key={cam.id} value={cam.id}>
                {cam.label}
              </option>
            ))}
          </select>
        </label>
      </section>

      <section className="relative grid gap-4 md:grid-cols-2">
        {status !== "connected" && (
          <div className="glass absolute inset-0 z-10 grid place-items-center rounded-2xl text-center">
            <div>
              <p className="text-lg font-medium">
                {status === "requesting-media" && "Requesting camera/microphone..."}
                {status === "connecting" && "Connecting to room..."}
                {status === "reconnecting" && "Reconnecting signaling..."}
                {status === "error" && "Connection issue"}
                {status === "idle" && "Preparing call..."}
              </p>
              <p className="mt-1 text-sm text-muted">
                {status === "error"
                  ? "Check backend and retry by refreshing this page."
                  : "Please wait a moment."}
              </p>
            </div>
          </div>
        )}

        <VideoTile stream={localStream} label={displayName || "You"} muted isLocal />
        {remotePeers.length === 0 ? (
          <div className="glass grid h-[260px] place-items-center rounded-2xl text-center text-sm text-muted md:h-[320px]">
            Waiting for another peer to join this room...
          </div>
        ) : (
          remotePeers.map((peer) => (
            <VideoTile
              key={peer.id}
              stream={peer.stream}
              label={peer.name || "Guest"}
            />
          ))
        )}
      </section>

      <section className="glass rounded-2xl p-4">
        <h2 className="text-sm font-medium text-foreground">Connection Quality</h2>
        {remotePeers.length === 0 ? (
          <p className="mt-2 text-sm text-muted">
            Quality metrics will appear when another peer joins.
          </p>
        ) : (
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {remotePeers.map((peer) => (
              <div key={`${peer.id}-stats`} className="rounded-xl border border-white/10 p-3">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-medium">{peer.name || "Guest"}</p>
                  <span
                    className={`rounded-full border px-2 py-0.5 text-xs ${qualityBadgeClass(peer.stats?.quality)}`}
                  >
                    {peer.stats?.quality ?? "connecting"}
                  </span>
                </div>
                <div className="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs text-muted">
                  <span>Bitrate: {peer.stats?.bitrateKbps ?? 0} kbps</span>
                  <span>RTT: {peer.stats?.rttMs ?? 0} ms</span>
                  <span>Loss: {peer.stats?.packetLossPct ?? 0}%</span>
                  <span>FPS: {peer.stats?.fps ?? 0}</span>
                  <span>Resolution: {peer.stats?.resolution ?? "-"}</span>
                  <span>State: {peer.stats?.connectionState ?? "new"}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <CallControls
        isMicEnabled={isMicEnabled}
        isCamEnabled={isCamEnabled}
        onToggleMic={toggleMic}
        onToggleCam={toggleCam}
        onLeave={() => {
          leaveRoom();
          router.push("/");
        }}
      />
    </main>
  );
}

