"use client";

import { useState } from "react";
import type { FormEvent } from "react";

export default function PublishPage() {
  const [step, setStep] = useState<"form" | "preview" | "done">("form");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [version, setVersion] = useState("");
  const [category, setCategory] = useState("");
  const [specUrl, setSpecUrl] = useState("");
  const [changelog, setChangelog] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState("");
  const [publishing, setPublishing] = useState(false);

  function handlePreview(e: FormEvent) {
    e.preventDefault();
    if (!name || !description || !version || !file) {
      setError("Name, description, version, and artifact file are required.");
      return;
    }
    setError("");
    setStep("preview");
  }

  async function handlePublish() {
    setPublishing(true);
    setError("");

    const apiBase =
      process.env.NEXT_PUBLIC_REGISTRY_API_URL || "http://localhost:8080";
    const token = localStorage.getItem("mint_api_key");
    if (!token) {
      setError("You must be logged in. Set your API key in account settings.");
      setPublishing(false);
      return;
    }

    const metadata = JSON.stringify({
      name,
      description,
      version,
      category: category || undefined,
      openapi_spec_url: specUrl || undefined,
      changelog: changelog || undefined,
    });

    const form = new FormData();
    form.set("metadata", metadata);
    if (file) form.set("artifact", file);

    try {
      const res = await fetch(`${apiBase}/api/v1/publish`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: form,
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || `Publish failed: ${res.status}`);
      }
      setStep("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Publish failed");
    } finally {
      setPublishing(false);
    }
  }

  if (step === "done") {
    return (
      <div className="mx-auto max-w-2xl py-20 text-center">
        <h1 className="text-3xl font-bold text-[var(--color-success)]">
          Published!
        </h1>
        <p className="mt-4 text-[var(--color-text-secondary)]">
          <strong>{name}</strong> v{version} is now live on the registry.
        </p>
        <div className="mt-6 flex justify-center gap-4">
          <a
            href={`/servers/${name}`}
            className="rounded-lg bg-[var(--color-accent)] px-5 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
          >
            View server
          </a>
          <button
            onClick={() => {
              setStep("form");
              setName("");
              setDescription("");
              setVersion("");
              setFile(null);
            }}
            className="rounded-lg border border-[var(--color-border)] px-5 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] transition-colors"
          >
            Publish another
          </button>
        </div>
      </div>
    );
  }

  if (step === "preview") {
    return (
      <div className="mx-auto max-w-2xl py-10">
        <h1 className="text-2xl font-bold">Preview</h1>
        <div className="mt-6 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-6 space-y-3">
          <p>
            <span className="text-[var(--color-text-secondary)]">Name:</span>{" "}
            {name}
          </p>
          <p>
            <span className="text-[var(--color-text-secondary)]">
              Version:
            </span>{" "}
            {version}
          </p>
          <p>
            <span className="text-[var(--color-text-secondary)]">
              Description:
            </span>{" "}
            {description}
          </p>
          {category && (
            <p>
              <span className="text-[var(--color-text-secondary)]">
                Category:
              </span>{" "}
              {category}
            </p>
          )}
          {specUrl && (
            <p>
              <span className="text-[var(--color-text-secondary)]">
                OpenAPI Spec:
              </span>{" "}
              {specUrl}
            </p>
          )}
          {changelog && (
            <p>
              <span className="text-[var(--color-text-secondary)]">
                Changelog:
              </span>{" "}
              {changelog}
            </p>
          )}
          <p>
            <span className="text-[var(--color-text-secondary)]">File:</span>{" "}
            {file?.name} ({((file?.size || 0) / 1024).toFixed(1)} KB)
          </p>
        </div>
        {error && (
          <p className="mt-4 text-sm text-red-400">{error}</p>
        )}
        <div className="mt-6 flex gap-4">
          <button
            onClick={() => setStep("form")}
            className="rounded-lg border border-[var(--color-border)] px-5 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] transition-colors"
          >
            Back
          </button>
          <button
            onClick={handlePublish}
            disabled={publishing}
            className="rounded-lg bg-[var(--color-accent)] px-5 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors disabled:opacity-50"
          >
            {publishing ? "Publishing..." : "Publish"}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl py-10">
      <h1 className="text-2xl font-bold">Publish an MCP Server</h1>
      <p className="mt-2 text-[var(--color-text-secondary)]">
        Share your MCP server with the community. You can also publish via the
        CLI:{" "}
        <code className="rounded bg-[var(--color-bg-tertiary)] px-2 py-0.5 text-sm">
          mint publish
        </code>
      </p>

      {error && (
        <p className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 px-4 py-2 text-sm text-red-400">
          {error}
        </p>
      )}

      <form onSubmit={handlePreview} className="mt-8 space-y-5">
        <div>
          <label htmlFor="name" className="block text-sm font-medium mb-1">
            Name
          </label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-mcp-server"
            pattern="^[a-z][a-z0-9\-]{1,62}[a-z0-9]$"
            required
            className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
          />
        </div>
        <div>
          <label
            htmlFor="description"
            className="block text-sm font-medium mb-1"
          >
            Description
          </label>
          <textarea
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            rows={3}
            className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
          />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="version"
              className="block text-sm font-medium mb-1"
            >
              Version
            </label>
            <input
              id="version"
              type="text"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="1.0.0"
              required
              className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
            />
          </div>
          <div>
            <label
              htmlFor="category"
              className="block text-sm font-medium mb-1"
            >
              Category
            </label>
            <select
              id="category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
            >
              <option value="">None</option>
              <option value="payments">Payments</option>
              <option value="crm">CRM</option>
              <option value="communication">Communication</option>
              <option value="devops">DevOps</option>
              <option value="analytics">Analytics</option>
              <option value="storage">Storage</option>
              <option value="ai">AI / ML</option>
              <option value="database">Database</option>
              <option value="monitoring">Monitoring</option>
              <option value="security">Security</option>
            </select>
          </div>
        </div>
        <div>
          <label htmlFor="specUrl" className="block text-sm font-medium mb-1">
            OpenAPI Spec URL (optional)
          </label>
          <input
            id="specUrl"
            type="url"
            value={specUrl}
            onChange={(e) => setSpecUrl(e.target.value)}
            placeholder="https://example.com/openapi.json"
            className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
          />
        </div>
        <div>
          <label
            htmlFor="changelog"
            className="block text-sm font-medium mb-1"
          >
            Changelog (optional)
          </label>
          <textarea
            id="changelog"
            value={changelog}
            onChange={(e) => setChangelog(e.target.value)}
            rows={3}
            className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-4 py-2 text-sm focus:border-[var(--color-accent)] focus:outline-none"
          />
        </div>
        <div>
          <label htmlFor="artifact" className="block text-sm font-medium mb-1">
            Artifact (.tar.gz)
          </label>
          <input
            id="artifact"
            type="file"
            accept=".tar.gz,.tgz"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            required
            className="w-full text-sm file:mr-4 file:rounded-lg file:border-0 file:bg-[var(--color-bg-tertiary)] file:px-4 file:py-2 file:text-sm file:text-[var(--color-text)]"
          />
        </div>
        <button
          type="submit"
          className="rounded-lg bg-[var(--color-accent)] px-6 py-2 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
        >
          Preview
        </button>
      </form>
    </div>
  );
}
