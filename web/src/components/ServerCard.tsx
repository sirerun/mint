import type { Server } from "@/lib/api";

export function ServerCard({ server }: { server: Server }) {
  return (
    <a
      href={`/servers/${server.name}`}
      className="group block rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-5 hover:border-[var(--color-accent)] transition-colors"
    >
      <div className="flex items-start justify-between">
        <h3 className="font-semibold group-hover:text-[var(--color-accent)] transition-colors">
          {server.name}
        </h3>
        <span className="rounded-md bg-[var(--color-bg-tertiary)] px-2 py-0.5 text-xs text-[var(--color-text-secondary)]">
          v{server.latest_version}
        </span>
      </div>
      <p className="mt-2 text-sm text-[var(--color-text-secondary)] line-clamp-2">
        {server.description}
      </p>
      <div className="mt-4 flex items-center gap-4 text-xs text-[var(--color-text-secondary)]">
        {server.category && (
          <span className="rounded-full bg-[var(--color-bg-tertiary)] px-2 py-0.5">
            {server.category}
          </span>
        )}
        <span>{server.downloads.toLocaleString()} downloads</span>
        <span>{server.stars} stars</span>
      </div>
    </a>
  );
}
