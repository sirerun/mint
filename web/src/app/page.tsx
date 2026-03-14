import { listServers, CATEGORIES } from "@/lib/api";
import { SearchBar } from "@/components/SearchBar";
import { ServerCard } from "@/components/ServerCard";
import { CategoryCard } from "@/components/CategoryCard";
import { ManagedBanner } from "@/components/ManagedBanner";

export default async function HomePage() {
  let featured;
  try {
    featured = await listServers({ sort: "downloads", page_size: 12 });
  } catch {
    featured = { servers: [], total: 0, page: 1, page_size: 12 };
  }

  return (
    <div className="py-16">
      {/* Hero */}
      <section className="mx-auto max-w-3xl text-center animate-fade-in">
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
          The MCP Server Registry
        </h1>
        <p className="mt-4 text-lg text-[var(--color-text-secondary)]">
          Discover, publish, and install MCP servers with a single command.
        </p>
        <div className="mt-8">
          <SearchBar size="lg" />
        </div>
        <p className="mt-3 text-sm text-[var(--color-text-secondary)]">
          <code className="rounded bg-[var(--color-bg-tertiary)] px-2 py-1">
            mint install {"<name>"}
          </code>
        </p>
      </section>

      {/* Categories */}
      <section className="mt-20">
        <h2 className="text-xl font-semibold">Browse by category</h2>
        <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
          {CATEGORIES.map((cat) => (
            <CategoryCard key={cat.slug} {...cat} />
          ))}
        </div>
      </section>

      {/* Featured */}
      <section className="mt-20">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Popular servers</h2>
          <a
            href="/servers?sort=downloads"
            className="text-sm text-[var(--color-accent)] hover:underline"
          >
            View all
          </a>
        </div>
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {featured.servers.map((s) => (
            <ServerCard key={s.id} server={s} />
          ))}
        </div>
        {featured.servers.length === 0 && (
          <p className="text-center text-[var(--color-text-secondary)] py-12">
            No servers published yet. Be the first to{" "}
            <a href="/publish" className="text-[var(--color-accent)] hover:underline">
              publish one
            </a>
            .
          </p>
        )}
      </section>

      {/* Managed promotion banner */}
      <ManagedBanner />
    </div>
  );
}
