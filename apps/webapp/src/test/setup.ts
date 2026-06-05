import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// jsdom não implementa a API de mídia; stub global para os testes.
HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
HTMLMediaElement.prototype.pause = vi.fn();
