export type SignalType =
  | "join"
  | "leave"
  | "offer"
  | "answer"
  | "ice-candidate"
  | "room-joined"
  | "peer-joined"
  | "peer-left"
  | "error";

export interface SignalingMessage {
  type: SignalType;
  roomId?: string;
  peerId?: string;
  targetId?: string;
  data?: unknown;
  error?: string;
  timestamp?: number;
}

export interface PeerSummary {
  peerId: string;
  name?: string;
}

export interface JoinMessageData {
  name?: string;
}

export interface RoomJoinedData {
  peers?: PeerSummary[];
}

export interface PeerJoinedData {
  name?: string;
}

export interface ICEServerConfig {
  urls: string[];
  username?: string;
  credential?: string;
}

