import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { AnalyticsTracker } from "./AnalyticsTracker";

vi.mock("next/navigation", () => ({
  usePathname: () => "/servers",
}));

const mockTrackEvent = vi.fn();
vi.mock("@/lib/analytics", () => ({
  trackEvent: (...args: unknown[]) => mockTrackEvent(...args),
}));

describe("AnalyticsTracker", () => {
  it("fires a registry_visit event on mount", () => {
    render(<AnalyticsTracker />);
    expect(mockTrackEvent).toHaveBeenCalledWith({
      event: "registry_visit",
      page: "/servers",
    });
  });

  it("renders nothing visible", () => {
    const { container } = render(<AnalyticsTracker />);
    expect(container.innerHTML).toBe("");
  });
});
