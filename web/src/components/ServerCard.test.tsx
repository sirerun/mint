import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ServerCard } from "./ServerCard";
import type { Server } from "@/lib/api";

const mockServer: Server = {
  id: "s1",
  name: "test-server",
  description: "A test MCP server for testing purposes",
  latest_version: "1.2.3",
  publisher_id: "pub1",
  category: "devops",
  downloads: 1234,
  stars: 42,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-03-01T00:00:00Z",
};

describe("ServerCard", () => {
  it("renders server name", () => {
    render(<ServerCard server={mockServer} />);
    expect(screen.getByText("test-server")).toBeInTheDocument();
  });

  it("renders version badge", () => {
    render(<ServerCard server={mockServer} />);
    expect(screen.getByText("v1.2.3")).toBeInTheDocument();
  });

  it("renders description", () => {
    render(<ServerCard server={mockServer} />);
    expect(
      screen.getByText("A test MCP server for testing purposes")
    ).toBeInTheDocument();
  });

  it("renders category", () => {
    render(<ServerCard server={mockServer} />);
    expect(screen.getByText("devops")).toBeInTheDocument();
  });

  it("renders download count", () => {
    render(<ServerCard server={mockServer} />);
    expect(screen.getByText("1,234 downloads")).toBeInTheDocument();
  });

  it("renders star count", () => {
    render(<ServerCard server={mockServer} />);
    expect(screen.getByText("42 stars")).toBeInTheDocument();
  });

  it("links to server detail page", () => {
    render(<ServerCard server={mockServer} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/servers/test-server");
  });

  it("handles missing category", () => {
    const serverNoCategory = { ...mockServer, category: "" };
    render(<ServerCard server={serverNoCategory} />);
    expect(screen.queryByText("devops")).not.toBeInTheDocument();
  });
});
