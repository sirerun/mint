import { listServers, CATEGORIES } from "@/lib/api";
import { ServerCard } from "@/components/ServerCard";
import { Pagination } from "@/components/Pagination";
import { SearchBar } from "@/components/SearchBar";
import type { Metadata } from "next";

interface Props {
  searchParams: Promise<{
    category?: string;
    sort?: string;
    q?: string;
    page?: string;
    publisher_id?: string;
  }>;
}

export async function generateMetadata({ searchParams }: Props): Promise<Metadata> {
  const params = await searchParams;
  const title = params.category
    ? `${params.category} MCP Servers - Mint Registry`
    : "Browse MCP Servers - Mint Registry";
  return { title };
}

const SORT_OPTIONS = [
  { value: "downloads", label: "Most downloaded" },
  { value: "stars", label: "Most starred" },
  { value: "recent", label: "Recently updated" },
  { value: "name", label: "Name" },
];

export default async function ServersPage({ searchParams }: Props) {
  const params = await searchParams;
  const page = Math.max(1, Number(params.page) || 1);
  const pageSize = 24;

  let data;
  try {
    data = await listServers({
      page,
      page_size: pageSize,
      category: params.category,
      sort: params.sort || "downloads",
      q: params.q,
      publisher_id: params.publisher_id,
    });
  } catch {
    data = { servers: [], total: 0, page: 1, page_size: pageSize };
  }

  const totalPages = Math.ceil(data.total / pageSize);

  function buildUrl(overrides: Record<string, string | undefined>) {
    const p = new URLSearchParams();
    const merged = { ...params, ...overrides };
    for (const [k, v] of Object.entries(merged)) {
      if (v) p.set(k, v);
    }
    return `/servers?${p}`;
  }

  return (
    <div className="py-10">
      <div className="mb-8">
        <SearchBar size="sm" />
      </div>

      <div className="flex flex-col gap-8 lg:flex-row">
        {/* Sidebar */}
        <aside className="w-full shrink-0 lg:w-56">
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-4">
            <h3 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Category
            </h3>
            <ul className="space-y-1">
              <li>
                <a
                  href={buildUrl({ category: undefined, page: undefined })}
                  className={`block rounded-lg px-3 py-1.5 text-sm transition-colors ${
                    !params.category
                      ? "bg-[var(--color-accent)] text-white"
                      : "hover:bg-[var(--color-bg-tertiary)]"
                  }`}
                >
                  All
                </a>
              </li>
              {CATEGORIES.map((cat) => (
                <li key={cat.slug}>
                  <a
                    href={buildUrl({ category: cat.slug, page: undefined })}
                    className={`block rounded-lg px-3 py-1.5 text-sm transition-colors ${
                      params.category === cat.slug
                        ? "bg-[var(--color-accent)] text-white"
                        : "hover:bg-[var(--color-bg-tertiary)]"
                    }`}
                  >
                    {cat.icon} {cat.label}
                  </a>
                </li>
              ))}
            </ul>

            <h3 className="mb-3 mt-6 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Sort
            </h3>
            <ul className="space-y-1">
              {SORT_OPTIONS.map((opt) => (
                <li key={opt.value}>
                  <a
                    href={buildUrl({ sort: opt.value, page: undefined })}
                    className={`block rounded-lg px-3 py-1.5 text-sm transition-colors ${
                      (params.sort || "downloads") === opt.value
                        ? "bg-[var(--color-accent)] text-white"
                        : "hover:bg-[var(--color-bg-tertiary)]"
                    }`}
                  >
                    {opt.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </aside>

        {/* Grid */}
        <div className="flex-1">
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-[var(--color-text-secondary)]">
              {data.total} server{data.total !== 1 ? "s" : ""} found
            </p>
          </div>
          {data.servers.length > 0 ? (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {data.servers.map((s) => (
                <ServerCard key={s.id} server={s} />
              ))}
            </div>
          ) : (
            <p className="py-20 text-center text-[var(--color-text-secondary)]">
              No servers found matching your criteria.
            </p>
          )}
          <Pagination
            page={page}
            totalPages={totalPages}
            baseUrl={buildUrl({})}
          />
        </div>
      </div>
    </div>
  );
}
