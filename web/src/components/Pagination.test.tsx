import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Pagination } from "./Pagination";

describe("Pagination", () => {
  it("renders nothing when totalPages is 1", () => {
    const { container } = render(
      <Pagination page={1} totalPages={1} baseUrl="/servers" />
    );
    expect(container.querySelector("nav")).toBeNull();
  });

  it("renders page links", () => {
    render(<Pagination page={2} totalPages={5} baseUrl="/servers" />);
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
  });

  it("renders Previous and Next buttons", () => {
    render(<Pagination page={3} totalPages={5} baseUrl="/servers" />);
    expect(screen.getByText("Previous")).toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
  });

  it("hides Previous on first page", () => {
    render(<Pagination page={1} totalPages={5} baseUrl="/servers" />);
    expect(screen.queryByText("Previous")).not.toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
  });

  it("hides Next on last page", () => {
    render(<Pagination page={5} totalPages={5} baseUrl="/servers" />);
    expect(screen.getByText("Previous")).toBeInTheDocument();
    expect(screen.queryByText("Next")).not.toBeInTheDocument();
  });

  it("highlights current page", () => {
    render(<Pagination page={2} totalPages={3} baseUrl="/servers" />);
    const current = screen.getByText("2");
    expect(current.className).toContain("bg-[var(--color-accent)]");
  });
});
