"use client";

export function Pagination({
  page,
  totalPages,
  baseUrl,
}: {
  page: number;
  totalPages: number;
  baseUrl: string;
}) {
  if (totalPages <= 1) return null;

  const pages: number[] = [];
  const start = Math.max(1, page - 2);
  const end = Math.min(totalPages, page + 2);
  for (let i = start; i <= end; i++) pages.push(i);

  function href(p: number) {
    const url = new URL(baseUrl, "https://mint.sire.run");
    url.searchParams.set("page", String(p));
    return `${url.pathname}?${url.searchParams}`;
  }

  return (
    <nav aria-label="Pagination" className="mt-8 flex items-center justify-center gap-2">
      {page > 1 && (
        <a
          href={href(page - 1)}
          className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] transition-colors"
        >
          Previous
        </a>
      )}
      {pages.map((p) => (
        <a
          key={p}
          href={href(p)}
          className={`rounded-lg px-3 py-2 text-sm transition-colors ${
            p === page
              ? "bg-[var(--color-accent)] text-white"
              : "border border-[var(--color-border)] hover:bg-[var(--color-bg-tertiary)]"
          }`}
        >
          {p}
        </a>
      ))}
      {page < totalPages && (
        <a
          href={href(page + 1)}
          className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] transition-colors"
        >
          Next
        </a>
      )}
    </nav>
  );
}
