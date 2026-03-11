"use client";

import { buildDeployToSireUrl, trackEvent } from "@/lib/analytics";

export function DeployToSireCTA({ serverName }: { serverName: string }) {
  const url = buildDeployToSireUrl(serverName);

  function handleClick() {
    trackEvent({ event: "deploy_to_sire_click", server: serverName });
  }

  return (
    <a
      href={url}
      onClick={handleClick}
      className="mt-4 flex w-full items-center justify-center gap-2 rounded-xl bg-[var(--color-accent)] px-5 py-3 font-medium text-white hover:bg-[var(--color-accent-hover)] transition-colors"
      data-testid="deploy-to-sire-cta"
    >
      Deploy to Sire
    </a>
  );
}
