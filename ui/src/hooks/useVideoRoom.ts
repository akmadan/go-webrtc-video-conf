"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { fetchIceServers } from "@/lib/ice";
import { getSignalingWsUrl } from "@/lib/env";
import type {
  JoinMessageData,
  PeerJoinedData,
  RoomJoinedData,
  SignalingMessage,
} from "@/types/signaling";

type ConnectionStatus =
  | "idle"
  | "requesting-media"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "error";

interface RemotePeer {
  id: string;
  name: string;
  stream: MediaStream;
  stats?: PeerStats;
}

interface DeviceOption {
  id: string;
  label: string;
}

interface PeerStats {
  connectionState: RTCPeerConnectionState;
  bitrateKbps: number;
  rttMs: number;
  packetLossPct: number;
  fps: number;
  resolution: string;
  quality: "good" | "fair" | "poor" | "connecting";
}

interface UseVideoRoomResult {
  localStream: MediaStream | null;
  remotePeers: RemotePeer[];
  status: ConnectionStatus;
  error: string | null;
  isMicEnabled: boolean;
  isCamEnabled: boolean;
  microphones: DeviceOption[];
  cameras: DeviceOption[];
  selectedMicId: string;
  selectedCameraId: string;
  switchDevices: (audioDeviceId: string, videoDeviceId: string) => Promise<void>;
  toggleMic: () => void;
  toggleCam: () => void;
  leaveRoom: () => void;
}

interface UseVideoRoomOptions {
  roomId: string;
  displayName: string;
  preferredMicId?: string;
  preferredCameraId?: string;
}

export function useVideoRoom({
  roomId,
  displayName,
  preferredMicId,
  preferredCameraId,
}: UseVideoRoomOptions): UseVideoRoomResult {
  const [status, setStatus] = useState<ConnectionStatus>("idle");
  const [error, setError] = useState<string | null>(null);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remotePeers, setRemotePeers] = useState<RemotePeer[]>([]);
  const [isMicEnabled, setIsMicEnabled] = useState(true);
  const [isCamEnabled, setIsCamEnabled] = useState(true);
  const [microphones, setMicrophones] = useState<DeviceOption[]>([]);
  const [cameras, setCameras] = useState<DeviceOption[]>([]);
  const [selectedMicId, setSelectedMicId] = useState("");
  const [selectedCameraId, setSelectedCameraId] = useState("");

  const wsRef = useRef<WebSocket | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const pcsRef = useRef<Map<string, RTCPeerConnection>>(new Map());
  const remoteStreamsRef = useRef<Map<string, MediaStream>>(new Map());
  const iceServersRef = useRef<RTCIceServer[]>([{ urls: "stun:stun.l.google.com:19302" }]);
  const shouldReconnectRef = useRef(true);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const statsIntervalRef = useRef<number | null>(null);
  const micEnabledRef = useRef(true);
  const camEnabledRef = useRef(true);
  const peerNamesRef = useRef<Map<string, string>>(new Map());
  const peerStatsRef = useRef<Map<string, PeerStats>>(new Map());
  const peerPrevBytesRef = useRef<Map<string, { bytes: number; ts: number }>>(
    new Map(),
  );

  const syncRemotePeers = useCallback(() => {
    const entries: RemotePeer[] = Array.from(remoteStreamsRef.current.entries()).map(
      ([id, stream]) => ({
        id,
        stream,
        name: peerNamesRef.current.get(id) ?? "Guest",
        stats: peerStatsRef.current.get(id),
      }),
    );
    setRemotePeers(entries);
  }, []);

  const sendMessage = useCallback((msg: SignalingMessage) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify(msg));
  }, []);

  const refreshDevices = useCallback(async () => {
    const devices = await navigator.mediaDevices.enumerateDevices();
    const micOptions = devices
      .filter((d) => d.kind === "audioinput")
      .map((d, idx) => ({
        id: d.deviceId,
        label: d.label || `Microphone ${idx + 1}`,
      }));
    const camOptions = devices
      .filter((d) => d.kind === "videoinput")
      .map((d, idx) => ({
        id: d.deviceId,
        label: d.label || `Camera ${idx + 1}`,
      }));
    setMicrophones(micOptions);
    setCameras(camOptions);
    setSelectedMicId((prev) => prev || micOptions[0]?.id || "");
    setSelectedCameraId((prev) => prev || camOptions[0]?.id || "");
  }, []);

  const closePeerConnection = useCallback((peerId: string) => {
    const pc = pcsRef.current.get(peerId);
    if (pc) {
      pc.onicecandidate = null;
      pc.ontrack = null;
      pc.close();
      pcsRef.current.delete(peerId);
    }
    const remote = remoteStreamsRef.current.get(peerId);
    if (remote) {
      remote.getTracks().forEach((t) => t.stop());
      remoteStreamsRef.current.delete(peerId);
      peerNamesRef.current.delete(peerId);
      peerStatsRef.current.delete(peerId);
      peerPrevBytesRef.current.delete(peerId);
      syncRemotePeers();
    }
  }, [syncRemotePeers]);

  const qualityFromStats = (
    connectionState: RTCPeerConnectionState,
    bitrateKbps: number,
    rttMs: number,
    packetLossPct: number,
  ): PeerStats["quality"] => {
    if (connectionState !== "connected") return "connecting";
    if (packetLossPct > 8 || rttMs > 300 || bitrateKbps < 250) return "poor";
    if (packetLossPct > 3 || rttMs > 180 || bitrateKbps < 600) return "fair";
    return "good";
  };

  const collectPeerStats = useCallback(async () => {
    const entries = Array.from(pcsRef.current.entries());
    for (const [peerId, pc] of entries) {
      try {
        const report = await pc.getStats();
        let bytes = 0;
        let packetsLost = 0;
        let packetsReceived = 0;
        let rttMs = 0;
        let fps = 0;
        let frameWidth = 0;
        let frameHeight = 0;

        report.forEach((item) => {
          if (
            item.type === "inbound-rtp" &&
            !("isRemote" in item && item.isRemote)
          ) {
            const inbound = item as RTCInboundRtpStreamStats;
            bytes += inbound.bytesReceived ?? 0;
            packetsLost += inbound.packetsLost ?? 0;
            packetsReceived += inbound.packetsReceived ?? 0;
            if (inbound.kind === "video") {
              fps = (inbound.framesPerSecond as number | undefined) ?? fps;
              frameWidth = (inbound.frameWidth as number | undefined) ?? frameWidth;
              frameHeight = (inbound.frameHeight as number | undefined) ?? frameHeight;
            }
          }
          if (item.type === "candidate-pair") {
            const pair = item as RTCIceCandidatePairStats;
            if (
              pair.state === "succeeded" &&
              (pair.nominated || pair.selected)
            ) {
              rttMs = Math.round((pair.currentRoundTripTime ?? 0) * 1000);
            }
          }
        });

        const now = Date.now();
        const prev = peerPrevBytesRef.current.get(peerId);
        let bitrateKbps = 0;
        if (prev && now > prev.ts) {
          const deltaBytes = Math.max(bytes - prev.bytes, 0);
          const deltaSec = (now - prev.ts) / 1000;
          bitrateKbps = Math.round((deltaBytes * 8) / 1000 / deltaSec);
        }
        peerPrevBytesRef.current.set(peerId, { bytes, ts: now });

        const totalPackets = packetsLost + packetsReceived;
        const packetLossPct =
          totalPackets > 0 ? Math.round((packetsLost / totalPackets) * 1000) / 10 : 0;

        const resolution =
          frameWidth > 0 && frameHeight > 0 ? `${frameWidth}x${frameHeight}` : "-";
        const connectionState = pc.connectionState;

        peerStatsRef.current.set(peerId, {
          connectionState,
          bitrateKbps,
          rttMs,
          packetLossPct,
          fps: Math.round(fps),
          resolution,
          quality: qualityFromStats(
            connectionState,
            bitrateKbps,
            rttMs,
            packetLossPct,
          ),
        });
      } catch {
        // Skip transient getStats errors.
      }
    }
    syncRemotePeers();
  }, [syncRemotePeers]);

  const getOrCreatePeerConnection = useCallback(
    (peerId: string) => {
      const existing = pcsRef.current.get(peerId);
      if (existing) return existing;

      const pc = new RTCPeerConnection({ iceServers: iceServersRef.current });
      pcsRef.current.set(peerId, pc);

      const local = localStreamRef.current;
      if (local) {
        local.getTracks().forEach((track) => pc.addTrack(track, local));
      }

      pc.onicecandidate = (event) => {
        if (!event.candidate) return;
        sendMessage({
          type: "ice-candidate",
          roomId,
          targetId: peerId,
          data: {
            candidate: event.candidate.candidate,
            sdpMid: event.candidate.sdpMid,
            sdpMLineIndex: event.candidate.sdpMLineIndex,
          },
        });
      };

      pc.ontrack = (event) => {
        const [stream] = event.streams;
        if (!stream) return;
        remoteStreamsRef.current.set(peerId, stream);
        syncRemotePeers();
      };

      return pc;
    },
    [roomId, sendMessage, syncRemotePeers],
  );

  const createOfferForPeer = useCallback(
    async (peerId: string) => {
      const pc = getOrCreatePeerConnection(peerId);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      sendMessage({
        type: "offer",
        roomId,
        targetId: peerId,
        data: {
          type: offer.type,
          sdp: offer.sdp,
        },
      });
    },
    [getOrCreatePeerConnection, roomId, sendMessage],
  );

  const leaveRoom = useCallback(() => {
    shouldReconnectRef.current = false;
    if (reconnectTimeoutRef.current) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    sendMessage({ type: "leave", roomId });

    pcsRef.current.forEach((_, peerId) => {
      closePeerConnection(peerId);
    });
    pcsRef.current.clear();

    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((t) => t.stop());
      localStreamRef.current = null;
      setLocalStream(null);
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    if (statsIntervalRef.current) {
      window.clearInterval(statsIntervalRef.current);
      statsIntervalRef.current = null;
    }

    setStatus("idle");
  }, [closePeerConnection, roomId, sendMessage]);

  const applyStream = useCallback((nextStream: MediaStream) => {
    const prevStream = localStreamRef.current;
    localStreamRef.current = nextStream;
    nextStream.getAudioTracks().forEach((t) => {
      t.enabled = micEnabledRef.current;
    });
    nextStream.getVideoTracks().forEach((t) => {
      t.enabled = camEnabledRef.current;
    });
    setLocalStream(nextStream);

    pcsRef.current.forEach((pc) => {
      const audioTrack = nextStream.getAudioTracks()[0];
      const videoTrack = nextStream.getVideoTracks()[0];
      pc.getSenders().forEach((sender) => {
        if (sender.track?.kind === "audio" && audioTrack) {
          sender.replaceTrack(audioTrack).catch(() => {});
        }
        if (sender.track?.kind === "video" && videoTrack) {
          sender.replaceTrack(videoTrack).catch(() => {});
        }
      });
    });

    if (prevStream) {
      prevStream.getTracks().forEach((t) => t.stop());
    }
  }, []);

  const getMediaStream = useCallback(
    async (audioDeviceId?: string, videoDeviceId?: string) => {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: audioDeviceId
          ? { deviceId: { exact: audioDeviceId } }
          : true,
        video: videoDeviceId
          ? {
              deviceId: { exact: videoDeviceId },
              width: { ideal: 1280 },
              height: { ideal: 720 },
            }
          : {
              width: { ideal: 1280 },
              height: { ideal: 720 },
            },
      });
      return stream;
    },
    [],
  );

  const switchDevices = useCallback(
    async (audioDeviceId: string, videoDeviceId: string) => {
      const stream = await getMediaStream(audioDeviceId || undefined, videoDeviceId || undefined);
      applyStream(stream);
      setSelectedMicId(audioDeviceId);
      setSelectedCameraId(videoDeviceId);
      await refreshDevices();
    },
    [applyStream, getMediaStream, refreshDevices],
  );

  const connectSocket = useCallback(() => {
    if (!shouldReconnectRef.current) return;
    const socket = new WebSocket(getSignalingWsUrl());
    wsRef.current = socket;

    socket.onopen = () => {
      reconnectAttemptsRef.current = 0;
      setStatus("connected");
      setError(null);
      const payload: JoinMessageData = {
        name: displayName,
      };
      sendMessage({ type: "join", roomId, data: payload });
    };

    socket.onmessage = async (event) => {
      try {
        const message = JSON.parse(event.data) as SignalingMessage;
        const peerId = message.peerId;

        if (message.type === "error") {
          setError(message.error ?? "signaling error");
          return;
        }

        if (message.type === "room-joined") {
          const payload = (message.data as RoomJoinedData | undefined) ?? {};
          for (const peer of payload.peers ?? []) {
            if (!peer.peerId) continue;
            peerNamesRef.current.set(peer.peerId, peer.name || "Guest");
          }
          syncRemotePeers();
          return;
        }

        if (message.type === "peer-joined" && peerId) {
          const payload = message.data as PeerJoinedData | undefined;
          peerNamesRef.current.set(peerId, payload?.name || "Guest");
          await createOfferForPeer(peerId);
          return;
        }

        if (message.type === "peer-left" && peerId) {
          closePeerConnection(peerId);
          return;
        }

        if (message.type === "offer" && peerId) {
          const offer = message.data as RTCSessionDescriptionInit;
          const pc = getOrCreatePeerConnection(peerId);
          await pc.setRemoteDescription(offer);
          const answer = await pc.createAnswer();
          await pc.setLocalDescription(answer);
          sendMessage({
            type: "answer",
            roomId,
            targetId: peerId,
            data: {
              type: answer.type,
              sdp: answer.sdp,
            },
          });
          return;
        }

        if (message.type === "answer" && peerId) {
          const answer = message.data as RTCSessionDescriptionInit;
          const pc = pcsRef.current.get(peerId);
          if (!pc) return;
          // Ignore stale duplicate answers that can arrive after renegotiation.
          if (pc.signalingState !== "have-local-offer") return;
          await pc.setRemoteDescription(answer);
          return;
        }

        if (message.type === "ice-candidate" && peerId) {
          const candidateData = message.data as RTCIceCandidateInit;
          const pc = getOrCreatePeerConnection(peerId);
          await pc.addIceCandidate(new RTCIceCandidate(candidateData));
          return;
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? `Failed to process signaling message: ${err.message}`
            : "Failed to process signaling message",
        );
      }
    };

    socket.onerror = () => {
      setError("WebSocket connection issue");
    };

    socket.onclose = () => {
      if (!shouldReconnectRef.current) return;
      const nextAttempt = reconnectAttemptsRef.current + 1;
      reconnectAttemptsRef.current = nextAttempt;
      if (nextAttempt > 6) {
        setStatus("error");
        setError("Could not reconnect to signaling server");
        return;
      }
      setStatus("reconnecting");
      const delay = Math.min(1200 * nextAttempt, 6000);
      reconnectTimeoutRef.current = window.setTimeout(() => {
        connectSocket();
      }, delay);
    };
  }, [
    closePeerConnection,
    createOfferForPeer,
    getOrCreatePeerConnection,
    roomId,
    sendMessage,
    displayName,
    syncRemotePeers,
  ]);

  useEffect(() => {
    let cancelled = false;
    shouldReconnectRef.current = true;

    const setup = async () => {
      try {
        setStatus("requesting-media");
        setError(null);
        const iceServers = await fetchIceServers();
        if (cancelled) return;
        iceServersRef.current = iceServers;

        const stream = await getMediaStream(preferredMicId || undefined, preferredCameraId || undefined);
        if (cancelled) return;
        applyStream(stream);
        await refreshDevices();
        setStatus("connecting");
        connectSocket();

        if (!statsIntervalRef.current) {
          statsIntervalRef.current = window.setInterval(() => {
            collectPeerStats();
          }, 2000);
        }
      } catch (err) {
        setStatus("error");
        setError(
          err instanceof Error
            ? err.message
            : "Failed to initialize camera or signaling",
        );
      }
    };

    setup();

    return () => {
      cancelled = true;
      leaveRoom();
    };
  }, [
    applyStream,
    collectPeerStats,
    connectSocket,
    getMediaStream,
    leaveRoom,
    preferredCameraId,
    preferredMicId,
    refreshDevices,
  ]);

  const toggleMic = useCallback(() => {
    const stream = localStreamRef.current;
    if (!stream) return;
    const next = !isMicEnabled;
    micEnabledRef.current = next;
    stream.getAudioTracks().forEach((track) => {
      track.enabled = next;
    });
    setIsMicEnabled(next);
  }, [isMicEnabled]);

  const toggleCam = useCallback(() => {
    const stream = localStreamRef.current;
    if (!stream) return;
    const next = !isCamEnabled;
    camEnabledRef.current = next;
    stream.getVideoTracks().forEach((track) => {
      track.enabled = next;
    });
    setIsCamEnabled(next);
  }, [isCamEnabled]);

  return useMemo(
    () => ({
      localStream,
      remotePeers,
      status,
      error,
      isMicEnabled,
      isCamEnabled,
      microphones,
      cameras,
      selectedMicId,
      selectedCameraId,
      switchDevices,
      toggleMic,
      toggleCam,
      leaveRoom,
    }),
    [
      localStream,
      remotePeers,
      status,
      error,
      isMicEnabled,
      isCamEnabled,
      microphones,
      cameras,
      selectedMicId,
      selectedCameraId,
      switchDevices,
      toggleMic,
      toggleCam,
      leaveRoom,
    ],
  );
}

