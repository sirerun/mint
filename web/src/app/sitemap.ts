import type { MetadataRoute } from "next";
import { listServers } from "@/lib/api";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    { url: "https://mint.sire.run", lastModified: new Date(), priority: 1 },
    {
      url: "https://mint.sire.run/servers",
      lastModified: new Date(),
      priority: 0.9,
    },
    {
      url: "https://mint.sire.run/publish",
      lastModified: new Date(),
      priority: 0.7,
    },
  ];

  try {
    const data = await listServers({ page_size: 100 });
    for (const server of data.servers) {
      entries.push({
        url: `https://mint.sire.run/servers/${server.name}`,
        lastModified: new Date(server.updated_at),
        priority: 0.8,
      });
    }
  } catch {
    // If API is unreachable, return static entries only.
  }

  return entries;
}
