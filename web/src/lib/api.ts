const API_BASE =
  process.env.NEXT_PUBLIC_REGISTRY_API_URL || "http://localhost:8080";

export interface Server {
  id: string;
  name: string;
  description: string;
  latest_version: string;
  openapi_spec_url?: string;
  publisher_id: string;
  category?: string;
  downloads: number;
  stars: number;
  created_at: string;
  updated_at: string;
}

export interface ServerListResponse {
  servers: Server[];
  total: number;
  page: number;
  page_size: number;
}

export interface Publisher {
  id: string;
  github_handle: string;
  verified: boolean;
  created_at: string;
}

export interface Version {
  id: string;
  server_id: string;
  version: string;
  changelog?: string;
  created_at: string;
}

export interface ListServersParams {
  page?: number;
  page_size?: number;
  category?: string;
  sort?: string;
  q?: string;
  publisher_id?: string;
}

export async function listServers(
  params: ListServersParams = {}
): Promise<ServerListResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.page_size)
    searchParams.set("page_size", String(params.page_size));
  if (params.category) searchParams.set("category", params.category);
  if (params.sort) searchParams.set("sort", params.sort);
  if (params.q) searchParams.set("q", params.q);
  if (params.publisher_id)
    searchParams.set("publisher_id", params.publisher_id);

  const url = `${API_BASE}/api/v1/servers?${searchParams}`;
  const res = await fetch(url, { next: { revalidate: 60 } });
  if (!res.ok) throw new Error(`Failed to fetch servers: ${res.status}`);
  return res.json();
}

export async function getServer(name: string): Promise<Server> {
  const res = await fetch(`${API_BASE}/api/v1/servers/${encodeURIComponent(name)}`, {
    next: { revalidate: 60 },
  });
  if (!res.ok) throw new Error(`Failed to fetch server: ${res.status}`);
  return res.json();
}

export const CATEGORIES = [
  { slug: "payments", label: "Payments", icon: "💳" },
  { slug: "crm", label: "CRM", icon: "👥" },
  { slug: "communication", label: "Communication", icon: "💬" },
  { slug: "devops", label: "DevOps", icon: "⚙️" },
  { slug: "analytics", label: "Analytics", icon: "📊" },
  { slug: "storage", label: "Storage", icon: "💾" },
  { slug: "ai", label: "AI / ML", icon: "🤖" },
  { slug: "database", label: "Database", icon: "🗄️" },
  { slug: "monitoring", label: "Monitoring", icon: "📡" },
  { slug: "security", label: "Security", icon: "🔐" },
] as const;
