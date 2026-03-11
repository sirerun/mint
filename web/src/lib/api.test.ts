import { describe, it, expect, vi, beforeEach } from "vitest";
import { listServers, getServer, CATEGORIES } from "./api";

const mockFetch = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch);
  mockFetch.mockReset();
});

describe("listServers", () => {
  it("fetches servers with default params", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve({ servers: [], total: 0, page: 1, page_size: 20 }),
    });
    const result = await listServers();
    expect(result.servers).toEqual([]);
    expect(mockFetch).toHaveBeenCalledOnce();
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("/api/v1/servers");
  });

  it("includes query params", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve({ servers: [], total: 0, page: 2, page_size: 10 }),
    });
    await listServers({ page: 2, category: "devops", q: "test" });
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("page=2");
    expect(url).toContain("category=devops");
    expect(url).toContain("q=test");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 500 });
    await expect(listServers()).rejects.toThrow("Failed to fetch servers: 500");
  });
});

describe("getServer", () => {
  it("fetches a server by name", async () => {
    const server = { id: "1", name: "test", description: "desc" };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(server),
    });
    const result = await getServer("test");
    expect(result.name).toBe("test");
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("/api/v1/servers/test");
  });

  it("encodes special characters in name", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: "1", name: "a%b" }),
    });
    await getServer("a%b");
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("/api/v1/servers/a%25b");
  });

  it("throws on not found", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 404 });
    await expect(getServer("nope")).rejects.toThrow(
      "Failed to fetch server: 404"
    );
  });
});

describe("CATEGORIES", () => {
  it("has at least 5 categories", () => {
    expect(CATEGORIES.length).toBeGreaterThanOrEqual(5);
  });

  it("each category has slug, label, and icon", () => {
    for (const cat of CATEGORIES) {
      expect(cat.slug).toBeTruthy();
      expect(cat.label).toBeTruthy();
      expect(cat.icon).toBeTruthy();
    }
  });
});
