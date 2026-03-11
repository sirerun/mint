import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import PublishPage from "./page";

describe("PublishPage", () => {
  it("renders the publish form heading", () => {
    render(<PublishPage />);
    expect(screen.getByText("Publish an MCP Server")).toBeInTheDocument();
  });

  it("renders required form fields", () => {
    render(<PublishPage />);
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Description")).toBeInTheDocument();
    expect(screen.getByLabelText("Version")).toBeInTheDocument();
    expect(screen.getByLabelText("Artifact (.tar.gz)")).toBeInTheDocument();
  });

  it("renders optional fields", () => {
    render(<PublishPage />);
    expect(screen.getByLabelText("Category")).toBeInTheDocument();
    expect(
      screen.getByLabelText("OpenAPI Spec URL (optional)")
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Changelog (optional)")).toBeInTheDocument();
  });

  it("renders the preview button", () => {
    render(<PublishPage />);
    expect(
      screen.getByRole("button", { name: "Preview" })
    ).toBeInTheDocument();
  });

  it("mentions CLI alternative", () => {
    render(<PublishPage />);
    expect(screen.getByText("mint publish")).toBeInTheDocument();
  });
});
