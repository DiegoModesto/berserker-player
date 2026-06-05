import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("../api/client", () => ({
  Endpoints: { scrobble: vi.fn() },
  coverURL: vi.fn(),
  streamURL: vi.fn(),
}));

import { Endpoints } from "../api/client";
import { scrobbleSafe } from "./helpers";

beforeEach(() => vi.clearAllMocks());

describe("scrobbleSafe", () => {
  it("chama o endpoint", async () => {
    (Endpoints.scrobble as any).mockResolvedValue(undefined);
    await scrobbleSafe("s1", "submission");
    expect(Endpoints.scrobble).toHaveBeenCalledWith("s1", "submission");
  });

  it("engole erros (não lança)", async () => {
    (Endpoints.scrobble as any).mockRejectedValue(new Error("falha"));
    await expect(scrobbleSafe("s1", "nowplaying")).resolves.toBeUndefined();
  });
});
