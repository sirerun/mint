"use client";

import { buildDeployUrl, trackEvent } from "@/lib/analytics";

export function DeployManagedCTA({ serverName }: { serverName: string }) {
  const url = buildDeployUrl(serverName);

  function handleClick() {
    trackEvent({ event: "deploy_managed_click", server: serverName });
  }

  return (
    <a
      href={url}
      onClick={handleClick}
      className="mt-4 flex w-full items-center justify-center gap-2 rounded-xl bg-[var(--color-accent)] px-5 py-3 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
      data-testid="deploy-managed-cta"
    >
      Deploy Managed
    </a>
  );
}
