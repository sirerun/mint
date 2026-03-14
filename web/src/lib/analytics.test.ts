import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { buildDeployUrl, trackEvent } from "./analytics";

describe("buildDeployUrl", () => {
  it("returns a deployment URL with correct params", () => {
    const url = buildDeployUrl("stripe-mcp");
    expect(url).toContain("https://mint.sire.run/deploy?");
    expect(url).toContain("mcp=stripe-mcp");
    expect(url).toContain("source=mint-registry");
    expect(url).toContain("utm_source=mint-registry");
    expect(url).toContain("utm_medium=cta");
    expect(url).toContain("utm_campaign=deploy-managed");
  });

  it("encodes special characters in server name", () => {
    const url = buildDeployUrl("my server+name");
    expect(url).toContain("mcp=my+server%2Bname");
  });
});

describe("trackEvent", () => {
  const originalEnv = process.env.NEXT_PUBLIC_ANALYTICS_URL;

  beforeEach(() => {
    vi.stubGlobal("navigator", { sendBeacon: vi.fn() });
  });

  afterEach(() => {
    process.env.NEXT_PUBLIC_ANALYTICS_URL = originalEnv;
    vi.unstubAllGlobals();
  });

  it("does nothing when analytics endpoint is not set", () => {
    delete process.env.NEXT_PUBLIC_ANALYTICS_URL;
    const beaconSpy = vi.fn();
    vi.stubGlobal("navigator", { sendBeacon: beaconSpy });
    trackEvent({ event: "registry_visit", page: "/" });
    expect(beaconSpy).not.toHaveBeenCalled();
  });

  it("sends beacon when endpoint is set", () => {
    process.env.NEXT_PUBLIC_ANALYTICS_URL = "https://analytics.test/track";
    const beaconSpy = vi.fn();
    vi.stubGlobal("navigator", { sendBeacon: beaconSpy });
    trackEvent({ event: "server_view", server: "test-server", category: "devops" });
    expect(beaconSpy).toHaveBeenCalledWith(
      "https://analytics.test/track",
      expect.stringContaining('"event":"server_view"')
    );
  });

  it("includes timestamp in payload", () => {
    process.env.NEXT_PUBLIC_ANALYTICS_URL = "https://analytics.test/track";
    const beaconSpy = vi.fn();
    vi.stubGlobal("navigator", { sendBeacon: beaconSpy });
    trackEvent({ event: "search", query: "stripe" });
    const payload = beaconSpy.mock.calls[0][1] as string;
    const parsed = JSON.parse(payload);
    expect(parsed.timestamp).toBeTruthy();
    expect(parsed.event).toBe("search");
    expect(parsed.query).toBe("stripe");
  });

  it("falls back to fetch when sendBeacon is unavailable", () => {
    process.env.NEXT_PUBLIC_ANALYTICS_URL = "https://analytics.test/track";
    const fetchSpy = vi.fn().mockResolvedValue({});
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("fetch", fetchSpy);
    trackEvent({ event: "download_click", server: "test" });
    expect(fetchSpy).toHaveBeenCalledWith(
      "https://analytics.test/track",
      expect.objectContaining({
        method: "POST",
        keepalive: true,
      })
    );
  });
});
