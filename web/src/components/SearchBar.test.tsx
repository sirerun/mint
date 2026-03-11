import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SearchBar } from "./SearchBar";

describe("SearchBar", () => {
  it("renders search input", () => {
    render(<SearchBar />);
    expect(
      screen.getByPlaceholderText("Search MCP servers...")
    ).toBeInTheDocument();
  });

  it("has an accessible label", () => {
    render(<SearchBar />);
    expect(screen.getByLabelText("Search MCP servers")).toBeInTheDocument();
  });

  it("renders with small size", () => {
    render(<SearchBar size="sm" />);
    const input = screen.getByPlaceholderText("Search MCP servers...");
    expect(input.className).toContain("text-sm");
  });

  it("renders with large size by default", () => {
    render(<SearchBar />);
    const input = screen.getByPlaceholderText("Search MCP servers...");
    expect(input.className).toContain("text-lg");
  });

  it("updates value on typing", () => {
    render(<SearchBar />);
    const input = screen.getByPlaceholderText(
      "Search MCP servers..."
    ) as HTMLInputElement;
    fireEvent.change(input, { target: { value: "stripe" } });
    expect(input.value).toBe("stripe");
  });

  it("is wrapped in a form submitting to /servers", () => {
    render(<SearchBar />);
    const input = screen.getByPlaceholderText("Search MCP servers...");
    const form = input.closest("form");
    expect(form).toHaveAttribute("action", "/servers");
    expect(form).toHaveAttribute("method", "get");
  });

  it("does not show suggestions initially", () => {
    render(<SearchBar />);
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });
});
