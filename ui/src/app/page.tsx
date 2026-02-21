"use client";

import { FormEvent, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

function makeRoomId() {
  return Math.random().toString(36).slice(2, 10);
}

export default function Home() {
  const router = useRouter();
  const [roomId, setRoomId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const suggestedRoom = useMemo(() => makeRoomId(), []);

  const joinRoom = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const value = roomId.trim().toLowerCase();
    if (!value) return;
    const name = displayName.trim();
    const query = name ? `?name=${encodeURIComponent(name)}` : "";
    router.push(`/room/${encodeURIComponent(value)}${query}`);
  };

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-6xl items-center justify-center px-6 py-14">
      <section className="glass grid w-full gap-8 rounded-3xl p-8 md:grid-cols-[1.1fr_1fr] md:p-12">
        <div className="space-y-5">
          <p className="inline-block rounded-full border border-white/15 px-3 py-1 text-xs text-muted">
            WebRTC P2P · Next.js + Go
          </p>
          <h1 className="text-4xl font-semibold leading-tight md:text-5xl">
            Beautiful video rooms for your private calls.
          </h1>
          <p className="max-w-xl text-muted">
            PulseMeet connects peers directly using WebRTC. Your Go backend
            handles signaling and ICE config while media stays peer-to-peer.
          </p>
          <div className="grid gap-4 text-sm text-muted md:grid-cols-2">
            <div className="rounded-2xl border border-white/10 p-4">
              <p className="text-foreground">Low latency</p>
              <p className="mt-1">Direct P2P media stream after negotiation.</p>
            </div>
            <div className="rounded-2xl border border-white/10 p-4">
              <p className="text-foreground">Secure signaling</p>
              <p className="mt-1">Token-ready signaling and room isolation.</p>
            </div>
          </div>
        </div>
        <div className="rounded-3xl border border-white/10 bg-black/20 p-6">
          <h2 className="text-xl font-medium">Join a room</h2>
          <p className="mt-1 text-sm text-muted">
            Share your room ID with anyone and start the call.
          </p>
          <form onSubmit={joinRoom} className="mt-6 space-y-4">
            <input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Your name (optional)"
              className="w-full rounded-xl border border-white/15 bg-black/25 px-4 py-3 outline-none transition focus:border-accent"
            />
            <input
              value={roomId}
              onChange={(e) => setRoomId(e.target.value)}
              placeholder={suggestedRoom}
              className="w-full rounded-xl border border-white/15 bg-black/25 px-4 py-3 outline-none transition focus:border-accent"
            />
            <button
              type="submit"
              className="w-full rounded-xl bg-accent px-4 py-3 font-medium text-black transition hover:bg-accent-2"
            >
              Enter Room
            </button>
          </form>
        </div>
      </section>
    </main>
  );
}
