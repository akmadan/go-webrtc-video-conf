import { RoomClient } from "@/components/RoomClient";

interface RoomPageProps {
  params: Promise<{ roomId: string }>;
  searchParams: Promise<{ name?: string }>;
}

export default async function RoomPage({ params, searchParams }: RoomPageProps) {
  const { roomId } = await params;
  const { name } = await searchParams;
  return (
    <RoomClient
      roomId={decodeURIComponent(roomId)}
      displayName={name ? decodeURIComponent(name) : "You"}
    />
  );
}

