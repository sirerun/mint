"use client";

import { useState, useEffect, useRef } from "react";
import type { Server } from "@/lib/api";

export function SearchBar({ size = "lg" }: { size?: "sm" | "lg" }) {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<Server[]>([]);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (query.length < 2) {
      setSuggestions([]);
      return;
    }
    const controller = new AbortController();
    const apiBase =
      process.env.NEXT_PUBLIC_REGISTRY_API_URL || "http://localhost:8080";
    fetch(
      `${apiBase}/api/v1/servers?q=${encodeURIComponent(query)}&page_size=5`,
      { signal: controller.signal }
    )
      .then((r) => r.json())
      .then((data) => {
        setSuggestions(data.servers || []);
        setOpen(true);
      })
      .catch(() => {});
    return () => controller.abort();
  }, [query]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const inputClass =
    size === "lg"
      ? "w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-5 py-4 text-lg placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-accent)] focus:outline-none transition-colors"
      : "w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-3 py-2 text-sm placeholder:text-[var(--color-text-secondary)] focus:border-[var(--color-accent)] focus:outline-none transition-colors";

  return (
    <div ref={ref} className="relative w-full">
      <form action="/servers" method="get">
        <input
          type="text"
          name="q"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search MCP servers..."
          className={inputClass}
          autoComplete="off"
          aria-label="Search MCP servers"
        />
      </form>
      {open && suggestions.length > 0 && (
        <ul className="absolute z-40 mt-2 w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] shadow-2xl overflow-hidden">
          {suggestions.map((s) => (
            <li key={s.id}>
              <a
                href={`/servers/${s.name}`}
                className="flex items-center justify-between px-5 py-3 hover:bg-[var(--color-bg-tertiary)] transition-colors"
              >
                <div>
                  <span className="font-medium">{s.name}</span>
                  <p className="text-sm text-[var(--color-text-secondary)] line-clamp-1">
                    {s.description}
                  </p>
                </div>
                <span className="text-xs text-[var(--color-text-secondary)]">
                  {s.downloads.toLocaleString()} downloads
                </span>
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
