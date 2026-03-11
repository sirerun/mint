export type AnalyticsEvent =
  | { event: "registry_visit"; page: string }
  | { event: "search"; query: string }
  | { event: "server_view"; server: string; category?: string }
  | { event: "download_click"; server: string }
  | { event: "deploy_to_sire_click"; server: string }
  | { event: "signup_conversion"; server: string; source: string };

function getAnalyticsEndpoint(): string {
  return process.env.NEXT_PUBLIC_ANALYTICS_URL || "";
}

export function trackEvent(payload: AnalyticsEvent): void {
  const ANALYTICS_ENDPOINT = getAnalyticsEndpoint();
  if (!ANALYTICS_ENDPOINT) return;

  const body = {
    ...payload,
    timestamp: new Date().toISOString(),
    url: typeof window !== "undefined" ? window.location.href : "",
    referrer: typeof document !== "undefined" ? document.referrer : "",
  };

  if (typeof navigator !== "undefined" && "sendBeacon" in navigator) {
    navigator.sendBeacon(
      ANALYTICS_ENDPOINT,
      JSON.stringify(body)
    );
  } else {
    fetch(ANALYTICS_ENDPOINT, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      keepalive: true,
    }).catch(() => {});
  }
}

export function buildDeployToSireUrl(serverName: string): string {
  const params = new URLSearchParams({
    mcp: serverName,
    source: "mint-registry",
    utm_source: "mint-registry",
    utm_medium: "cta",
    utm_campaign: "deploy-to-sire",
  });
  return `https://sire.run/signup?${params}`;
}
