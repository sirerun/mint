export function ManagedBanner() {
  return (
    <section className="mt-20 rounded-xl border border-[var(--color-border)] bg-gradient-to-r from-[var(--color-bg-secondary)] to-[var(--color-bg-tertiary)] p-8 text-center">
      <p className="text-lg font-semibold">
        Build MCP servers with Mint. Orchestrate them in production with{" "}
        <a
          href="https://mint.sire.run/managed?utm_source=mint-registry&utm_medium=banner&utm_campaign=managed-promo"
          className="text-[var(--color-accent)] hover:underline"
        >
          Managed Hosting
        </a>
        .
      </p>
      <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
        Go from local development to managed production deployment in minutes.
      </p>
      <a
        href="https://mint.sire.run/signup?source=mint-registry&utm_source=mint-registry&utm_medium=banner&utm_campaign=managed-promo"
        className="mt-4 inline-block rounded-lg bg-[var(--color-accent)] px-6 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
      >
        Get started for free
      </a>
    </section>
  );
}
