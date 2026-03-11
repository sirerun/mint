import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DeployToSireCTA } from "./DeployToSireCTA";

vi.mock("@/lib/analytics", () => ({
  buildDeployToSireUrl: (name: string) =>
    `https://sire.run/signup?mcp=${name}&source=mint-registry&utm_source=mint-registry&utm_medium=cta&utm_campaign=deploy-to-sire`,
  trackEvent: vi.fn(),
}));

describe("DeployToSireCTA", () => {
  it("renders a link to sire.run signup", () => {
    render(<DeployToSireCTA serverName="stripe-mcp" />);
    const link = screen.getByTestId("deploy-to-sire-cta");
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining("sire.run/signup")
    );
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining("mcp=stripe-mcp")
    );
  });

  it("includes utm params in the URL", () => {
    render(<DeployToSireCTA serverName="github-mcp" />);
    const link = screen.getByTestId("deploy-to-sire-cta");
    const href = link.getAttribute("href") || "";
    expect(href).toContain("utm_source=mint-registry");
    expect(href).toContain("utm_medium=cta");
    expect(href).toContain("utm_campaign=deploy-to-sire");
    expect(href).toContain("source=mint-registry");
  });

  it("renders Deploy to Sire text", () => {
    render(<DeployToSireCTA serverName="test" />);
    expect(screen.getByText("Deploy to Sire")).toBeInTheDocument();
  });

  it("tracks click event", async () => {
    const { trackEvent } = await import("@/lib/analytics");
    render(<DeployToSireCTA serverName="my-server" />);
    fireEvent.click(screen.getByTestId("deploy-to-sire-cta"));
    expect(trackEvent).toHaveBeenCalledWith({
      event: "deploy_to_sire_click",
      server: "my-server",
    });
  });
});
