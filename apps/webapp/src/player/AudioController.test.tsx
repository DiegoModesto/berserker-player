import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, waitFor } from "@testing-library/react";

vi.mock("./helpers", () => ({
  streamURL: vi.fn().mockResolvedValue("blob:stream"),
  coverURL: vi.fn().mockResolvedValue("blob:cover"),
  scrobbleSafe: vi.fn().mockResolvedValue(undefined),
}));

import { streamURL, scrobbleSafe } from "./helpers";
import { AudioController } from "./AudioController";
import { usePlayer } from "../store/player";

beforeEach(() => {
  vi.clearAllMocks();
  // jsdom não implementa play/pause.
  HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
  HTMLMediaElement.prototype.pause = vi.fn();
  // Media Session API simulada.
  (navigator as any).mediaSession = { setActionHandler: vi.fn(), metadata: null };
  (globalThis as any).MediaMetadata = class {
    constructor(init: unknown) {
      Object.assign(this, init);
    }
  };
  usePlayer.setState({
    queue: [
      { id: "a", title: "A", artistName: "Art", coverArtId: "c1" },
      { id: "b", title: "B" },
    ],
    index: 0,
    isPlaying: true,
    repeat: "off",
    volume: 1,
  });
});

function getAudio(container: HTMLElement): HTMLAudioElement {
  return container.querySelector("audio") as HTMLAudioElement;
}

describe("AudioController", () => {
  it("carrega a faixa atual e registra nowplaying", async () => {
    const { container } = render(<AudioController />);
    await waitFor(() => expect(streamURL).toHaveBeenCalledWith("a"));
    await waitFor(() => expect(scrobbleSafe).toHaveBeenCalledWith("a", "nowplaying"));
    expect(navigator.mediaSession.setActionHandler).toHaveBeenCalled();
    const audio = getAudio(container);
    expect(audio).toBeTruthy();
    // Eventos do elemento <audio> sincronizam o store.
    Object.defineProperty(audio, "duration", { value: 120, configurable: true });
    audio.dispatchEvent(new Event("loadedmetadata"));
    await waitFor(() => expect(usePlayer.getState().duration).toBe(120));
    audio.dispatchEvent(new Event("play"));
    expect(usePlayer.getState().isPlaying).toBe(true);
    audio.dispatchEvent(new Event("pause"));
    expect(usePlayer.getState().isPlaying).toBe(false);
  });

  it("scrobbla submission ao passar de 50%", async () => {
    const { container } = render(<AudioController />);
    await waitFor(() => expect(streamURL).toHaveBeenCalled());
    const audio = getAudio(container);
    Object.defineProperty(audio, "duration", { value: 100, configurable: true });
    audio.currentTime = 60;
    audio.dispatchEvent(new Event("timeupdate"));
    await waitFor(() => expect(scrobbleSafe).toHaveBeenCalledWith("a", "submission"));
  });

  it("ao terminar avança para a próxima faixa", async () => {
    const { container } = render(<AudioController />);
    await waitFor(() => expect(streamURL).toHaveBeenCalled());
    getAudio(container).dispatchEvent(new Event("ended"));
    await waitFor(() => expect(usePlayer.getState().index).toBe(1));
  });

  it("os action handlers da Media Session controlam o player", async () => {
    const handlers: Record<string, () => void> = {};
    (navigator as any).mediaSession.setActionHandler = vi.fn((action: string, cb: () => void) => {
      handlers[action] = cb;
    });
    render(<AudioController />);
    await waitFor(() => expect(streamURL).toHaveBeenCalled());

    usePlayer.setState({ isPlaying: false, index: 0 });
    handlers["play"]?.();
    expect(usePlayer.getState().isPlaying).toBe(true);
    handlers["pause"]?.();
    expect(usePlayer.getState().isPlaying).toBe(false);
    handlers["nexttrack"]?.();
    expect(usePlayer.getState().index).toBe(1);
    handlers["previoustrack"]?.();
    expect(usePlayer.getState().index).toBe(0);
  });
});
