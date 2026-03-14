import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DeployManagedCTA } from "./DeployManagedCTA";

vi.mock("@/lib/analytics", () => ({
  buildDeployUrl: (name: string) =>
    `https://mintmcp.com/deploy?mcp=${name}&source=mint-registry&utm_source=mint-registry&utm_medium=cta&utm_campaign=deploy-managed`,
  trackEvent: vi.fn(),
}));

describe("DeployManagedCTA", () => {
  it("renders a deployment link", () => {
    render(<DeployManagedCTA serverName="stripe-mcp" />);
    const link = screen.getByTestId("deploy-managed-cta");
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining("mintmcp.com/deploy")
    );
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining("mcp=stripe-mcp")
    );
  });

  it("includes utm params in the URL", () => {
    render(<DeployManagedCTA serverName="github-mcp" />);
    const link = screen.getByTestId("deploy-managed-cta");
    const href = link.getAttribute("href") || "";
    expect(href).toContain("utm_source=mint-registry");
    expect(href).toContain("utm_medium=cta");
    expect(href).toContain("utm_campaign=deploy-managed");
    expect(href).toContain("source=mint-registry");
  });

  it("renders Deploy Managed text", () => {
    render(<DeployManagedCTA serverName="test" />);
    expect(screen.getByText("Deploy Managed")).toBeInTheDocument();
  });

  it("tracks click event", async () => {
    const { trackEvent } = await import("@/lib/analytics");
    render(<DeployManagedCTA serverName="my-server" />);
    fireEvent.click(screen.getByTestId("deploy-managed-cta"));
    expect(trackEvent).toHaveBeenCalledWith({
      event: "deploy_managed_click",
      server: "my-server",
    });
  });
});
