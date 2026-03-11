import type { Metadata } from "next";
import { AnalyticsTracker } from "@/components/AnalyticsTracker";
import "./globals.css";

export const metadata: Metadata = {
  title: "Mint Registry - Discover MCP Servers",
  description:
    "The MCP Server Registry. Discover, publish, and install MCP servers with a single command.",
  openGraph: {
    title: "Mint Registry",
    description:
      "Discover, publish, and install MCP servers with a single command.",
    url: "https://mint.sire.run",
    siteName: "Mint Registry",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Mint Registry",
    description:
      "Discover, publish, and install MCP servers with a single command.",
  },
  metadataBase: new URL("https://mint.sire.run"),
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen antialiased">
        <AnalyticsTracker />
        <Header />
        <main className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          {children}
        </main>
        <Footer />
      </body>
    </html>
  );
}

function Header() {
  return (
    <header className="border-b border-[var(--color-border)] bg-[var(--color-bg)]/80 backdrop-blur-sm sticky top-0 z-50">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
        <a href="/" className="flex items-center gap-2 text-xl font-bold">
          <span className="text-[var(--color-accent)]">mint</span>
          <span className="text-[var(--color-text-secondary)] text-sm font-normal">
            registry
          </span>
        </a>
        <nav className="flex items-center gap-6">
          <a
            href="/servers"
            className="text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
          >
            Browse
          </a>
          <a
            href="/publish"
            className="text-sm rounded-lg bg-[var(--color-accent)] px-4 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
          >
            Publish
          </a>
        </nav>
      </div>
    </header>
  );
}

function Footer() {
  return (
    <footer className="mt-20 border-t border-[var(--color-border)] py-10">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
          <p className="text-sm text-[var(--color-text-secondary)]">
            Powered by{" "}
            <a
              href="https://sire.run?utm_source=mint-registry&utm_medium=footer&utm_campaign=powered-by"
              className="text-[var(--color-accent)] hover:underline"
            >
              Sire
            </a>
          </p>
          <div className="flex gap-6 text-sm text-[var(--color-text-secondary)]">
            <a href="https://sire.run" className="hover:text-[var(--color-text)] transition-colors">
              Sire
            </a>
            <a href="https://github.com/sirerun/mint" className="hover:text-[var(--color-text)] transition-colors">
              GitHub
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
