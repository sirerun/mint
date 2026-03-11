import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CategoryCard } from "./CategoryCard";

describe("CategoryCard", () => {
  it("renders category label", () => {
    render(<CategoryCard slug="payments" label="Payments" icon="$" />);
    expect(screen.getByText("Payments")).toBeInTheDocument();
  });

  it("renders icon", () => {
    render(<CategoryCard slug="payments" label="Payments" icon="$" />);
    expect(screen.getByText("$")).toBeInTheDocument();
  });

  it("links to filtered servers page", () => {
    render(<CategoryCard slug="devops" label="DevOps" icon="G" />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/servers?category=devops");
  });
});
