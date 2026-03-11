export function CategoryCard({
  slug,
  label,
  icon,
}: {
  slug: string;
  label: string;
  icon: string;
}) {
  return (
    <a
      href={`/servers?category=${slug}`}
      className="flex items-center gap-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-5 py-4 hover:border-[var(--color-accent)] transition-colors"
    >
      <span className="text-2xl" role="img" aria-label={label}>
        {icon}
      </span>
      <span className="font-medium">{label}</span>
    </a>
  );
}
