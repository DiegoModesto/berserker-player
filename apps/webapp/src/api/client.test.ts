import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as client from "./client";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("api client", () => {
  beforeEach(() => {
    client.logout();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("login guarda o access token", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ accessToken: "tok" }));
    vi.stubGlobal("fetch", fetchMock);
    await client.login("admin", "pw");
    expect(client.isAuthenticated()).toBe(true);
    const [, init] = fetchMock.mock.calls[0];
    expect(init.headers["X-Client"]).toBe("webapp");
  });

  it("login falho lança e não autentica", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({}, 401)));
    await expect(client.login("admin", "x")).rejects.toThrow();
    expect(client.isAuthenticated()).toBe(false);
  });

  it("refresh automático em 401 e retry", async () => {
    // login
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ accessToken: "old" })));
    await client.login("admin", "pw");

    const fetchMock = vi
      .fn()
      // 1ª chamada ao endpoint protegido → 401
      .mockResolvedValueOnce(jsonResponse({}, 401))
      // refresh → novo token
      .mockResolvedValueOnce(jsonResponse({ accessToken: "new" }))
      // retry → sucesso
      .mockResolvedValueOnce(jsonResponse({ id: "u", username: "admin", isAdmin: true }));
    vi.stubGlobal("fetch", fetchMock);

    const me = await client.Endpoints.me();
    expect(me.username).toBe("admin");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("ensureMediaToken cacheia e monta URLs de stream/cover", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ accessToken: "tok" })));
    await client.login("admin", "pw");

    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ token: "MT", expiresAt: new Date(Date.now() + 3600_000).toISOString() }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const sURL = await client.streamURL("song1");
    expect(sURL).toContain("/api/v1/stream/song1?token=MT");
    const cURL = await client.coverURL("alb1", 128);
    expect(cURL).toContain("/api/v1/cover/alb1?size=128&token=MT");
    // Segundo uso reaproveita o token (apenas 1 POST media-token).
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("tryRestore usa o refresh (cookie) e devolve bool", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ accessToken: "tok" })));
    expect(await client.tryRestore()).toBe(true);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({}, 401)));
    expect(await client.tryRestore()).toBe(false);
  });
});
