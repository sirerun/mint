import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { SireBanner } from "./SireBanner";

describe("SireBanner", () => {
  it("renders the promotional message", () => {
    render(<SireBanner />);
    expect(
      screen.getByText(/Build MCP servers with Mint/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Orchestrate them in production with/i)
    ).toBeInTheDocument();
  });

  it("links to sire.run with utm params", () => {
    render(<SireBanner />);
    const sireLink = screen.getByRole("link", { name: "Sire" });
    expect(sireLink.getAttribute("href")).toContain("sire.run");
    expect(sireLink.getAttribute("href")).toContain("utm_source=mint-registry");
  });

  it("renders Get started CTA linking to signup", () => {
    render(<SireBanner />);
    const cta = screen.getByRole("link", { name: "Get started with Sire" });
    expect(cta.getAttribute("href")).toContain("sire.run/signup");
    expect(cta.getAttribute("href")).toContain("source=mint-registry");
  });

  it("does not describe internal architecture", () => {
    const { container } = render(<SireBanner />);
    const text = container.textContent || "";
    expect(text).not.toMatch(/token/i);
    expect(text).not.toMatch(/architecture/i);
    expect(text).not.toMatch(/infrastructure/i);
    expect(text).not.toMatch(/pricing/i);
  });
});
