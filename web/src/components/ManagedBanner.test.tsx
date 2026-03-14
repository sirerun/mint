import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ManagedBanner } from "./ManagedBanner";

describe("ManagedBanner", () => {
  it("renders the promotional message", () => {
    render(<ManagedBanner />);
    expect(
      screen.getByText(/Build MCP servers with Mint/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Orchestrate them in production with/i)
    ).toBeInTheDocument();
  });

  it("links to managed hosting with utm params", () => {
    render(<ManagedBanner />);
    const managedLink = screen.getByRole("link", { name: /Managed Hosting/i });
    expect(managedLink.getAttribute("href")).toContain("mint.sire.run/managed");
    expect(managedLink.getAttribute("href")).toContain("utm_source=mint-registry");
  });

  it("renders Get started CTA linking to signup", () => {
    render(<ManagedBanner />);
    const cta = screen.getByRole("link", { name: /Get started for free/i });
    expect(cta.getAttribute("href")).toContain("mint.sire.run/signup");
    expect(cta.getAttribute("href")).toContain("source=mint-registry");
  });

  it("does not describe internal architecture", () => {
    const { container } = render(<ManagedBanner />);
    const text = container.textContent || "";
    expect(text).not.toMatch(/token/i);
    expect(text).not.toMatch(/architecture/i);
    expect(text).not.toMatch(/infrastructure/i);
    expect(text).not.toMatch(/pricing/i);
  });
});
