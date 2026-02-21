const HTTP_BASE =
  process.env.NEXT_PUBLIC_BACKEND_HTTP_URL ?? "http://localhost:8080";
const WS_BASE =
  process.env.NEXT_PUBLIC_SIGNALING_WS_URL ?? "ws://localhost:8080/ws";
const SIGNALING_TOKEN = process.env.NEXT_PUBLIC_SIGNALING_TOKEN ?? "";

export function getBackendHttpBase() {
  return HTTP_BASE;
}

export function getSignalingWsUrl() {
  const url = new URL(WS_BASE);
  if (SIGNALING_TOKEN) {
    url.searchParams.set("token", SIGNALING_TOKEN);
  }
  return url.toString();
}

