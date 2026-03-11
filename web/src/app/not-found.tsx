export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center py-32">
      <h1 className="text-6xl font-bold text-[var(--color-text-secondary)]">
        404
      </h1>
      <p className="mt-4 text-[var(--color-text-secondary)]">
        Page not found.
      </p>
      <a
        href="/"
        className="mt-6 rounded-lg bg-[var(--color-accent)] px-5 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
      >
        Go home
      </a>
    </div>
  );
}
