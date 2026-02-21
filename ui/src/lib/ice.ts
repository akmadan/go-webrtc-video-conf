import { getBackendHttpBase } from "@/lib/env";
import type { ICEServerConfig } from "@/types/signaling";

interface IceServersResponse {
  iceServers?: ICEServerConfig[];
}

export async function fetchIceServers(): Promise<RTCIceServer[]> {
  try {
    const response = await fetch(`${getBackendHttpBase()}/ice-servers`, {
      cache: "no-store",
    });
    if (!response.ok) {
      throw new Error("failed to fetch ice servers");
    }
    const payload = (await response.json()) as IceServersResponse;
    return (payload.iceServers ?? []).map((server) => ({
      urls: server.urls,
      username: server.username,
      credential: server.credential,
    }));
  } catch {
    return [{ urls: "stun:stun.l.google.com:19302" }];
  }
}

