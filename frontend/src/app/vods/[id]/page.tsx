import { notFound } from "next/navigation";
import { getVodDetail } from "@/lib/api";
import { VodView } from "./VodView";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";

interface VodPageProps {
  params: Promise<{ id: string }>;
}

export async function generateMetadata({
  params,
}: VodPageProps): Promise<Metadata> {
  const { id } = await params;
  try {
    const vod = await getVodDetail(id);
    return {
      title: `${vod.title || "Untitled stream"} — Past Stream`,
      description: `Watch ${vod.userName}'s past stream on DeepCut`,
    };
  } catch {
    return { title: "VOD not found — DeepCut" };
  }
}

export default async function VodPage({ params }: VodPageProps) {
  const { id } = await params;

  let vod;
  try {
    vod = await getVodDetail(id);
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

  // Prefer the backend-provided HLS URL (spec: use vod.hlsUrl); fall back
  // to the path convention only when the API hasn't set it yet.
  const hlsUrl = vod.hlsUrl ?? `/hls/vods/${id}/index.m3u8`;

  return <VodView vod={vod} hlsUrl={hlsUrl} />;
}
