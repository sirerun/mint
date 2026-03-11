import { getServer } from "@/lib/api";
import { DeployToSireCTA } from "@/components/DeployToSireCTA";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

interface Props {
  params: Promise<{ name: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { name } = await params;
  try {
    const server = await getServer(name);
    return {
      title: `${server.name} - Mint Registry`,
      description: server.description,
      openGraph: {
        title: `${server.name} - MCP Server`,
        description: server.description,
      },
    };
  } catch {
    return { title: "Server Not Found - Mint Registry" };
  }
}

export default async function ServerDetailPage({ params }: Props) {
  const { name } = await params;
  let server;
  try {
    server = await getServer(name);
  } catch {
    notFound();
  }

  return (
    <div className="py-10">
      <div className="flex flex-col gap-8 lg:flex-row">
        {/* Main content */}
        <div className="flex-1">
          <div className="flex items-start justify-between">
            <div>
              <h1 className="text-3xl font-bold">{server.name}</h1>
              <p className="mt-2 text-[var(--color-text-secondary)]">
                {server.description}
              </p>
            </div>
            {server.category && (
              <a
                href={`/servers?category=${server.category}`}
                className="rounded-full bg-[var(--color-bg-tertiary)] px-3 py-1 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
              >
                {server.category}
              </a>
            )}
          </div>

          {/* Install command */}
          <div className="mt-8 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-5">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              Install
            </h2>
            <code className="block rounded-lg bg-[var(--color-bg)] p-4 text-sm font-mono">
              mint install {server.name}
            </code>
          </div>

          {/* README placeholder */}
          <div className="mt-8 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-5">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
              README
            </h2>
            <div className="prose prose-invert max-w-none text-sm text-[var(--color-text-secondary)]">
              <p>
                This MCP server was generated from an OpenAPI specification
                {server.openapi_spec_url && (
                  <>
                    {" "}
                    available at{" "}
                    <a
                      href={server.openapi_spec_url}
                      className="text-[var(--color-accent)] hover:underline"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {server.openapi_spec_url}
                    </a>
                  </>
                )}
                .
              </p>
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <aside className="w-full shrink-0 lg:w-72">
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-5 space-y-4">
            <div>
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                Version
              </p>
              <p className="font-mono">{server.latest_version}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                Downloads
              </p>
              <p className="font-mono">{server.downloads.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                Stars
              </p>
              <p className="font-mono">{server.stars}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                Publisher
              </p>
              <a
                href={`/publishers/${server.publisher_id}`}
                className="text-[var(--color-accent)] hover:underline"
              >
                {server.publisher_id}
              </a>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-[var(--color-text-secondary)]">
                Updated
              </p>
              <p className="text-sm">
                {new Date(server.updated_at).toLocaleDateString()}
              </p>
            </div>
          </div>

          {/* Deploy to Sire CTA */}
          <DeployToSireCTA serverName={server.name} />
        </aside>
      </div>
    </div>
  );
}
