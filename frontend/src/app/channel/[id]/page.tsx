import { notFound } from "next/navigation";
import { cookies } from "next/headers";
import { getChannel } from "@/lib/api";
import { ChannelView } from "@/components/ChannelView";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";

interface ChannelPageProps {
  params: Promise<{ id: string }>;
}

export async function generateMetadata({
  params,
}: ChannelPageProps): Promise<Metadata> {
  const { id } = await params;
  try {
    const channel = await getChannel(id);
    return {
      title: `${channel.streamerName} — ${channel.streamTitle || "Live Stream"}`,
      description: `Watch ${channel.streamerName} live on DeepCut`,
    };
  } catch {
    return { title: "Channel not found — DeepCut" };
  }
}

export default async function ChannelPage({ params }: ChannelPageProps) {
  const { id } = await params;

  const cookieStore = await cookies();
  const isSignedIn = !!cookieStore.get("token");

  let channel;
  try {
    channel = await getChannel(id);
  } catch (error: unknown) {
    if (
      error instanceof Error &&
      "status" in error &&
      (error as { status: number }).status === 404
    ) {
      notFound();
    }
    throw error;
  }

  return <ChannelView id={id} initialChannel={channel} isSignedIn={isSignedIn} />;
}
