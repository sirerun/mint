import { listServers } from "@/lib/api";
import { ServerCard } from "@/components/ServerCard";
import type { Metadata } from "next";

interface Props {
  params: Promise<{ handle: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { handle } = await params;
  return {
    title: `${handle} - Publisher - Mint Registry`,
    description: `MCP servers published by ${handle}.`,
  };
}

export default async function PublisherPage({ params }: Props) {
  const { handle } = await params;

  let data;
  try {
    data = await listServers({ publisher_id: handle, page_size: 50 });
  } catch {
    data = { servers: [], total: 0, page: 1, page_size: 50 };
  }

  return (
    <div className="py-10">
      <div className="mb-8">
        <h1 className="text-3xl font-bold">{handle}</h1>
        <p className="mt-2 text-[var(--color-text-secondary)]">
          {data.total} published server{data.total !== 1 ? "s" : ""}
        </p>
      </div>

      {data.servers.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.servers.map((s) => (
            <ServerCard key={s.id} server={s} />
          ))}
        </div>
      ) : (
        <p className="py-20 text-center text-[var(--color-text-secondary)]">
          This publisher has not published any servers yet.
        </p>
      )}
    </div>
  );
}
