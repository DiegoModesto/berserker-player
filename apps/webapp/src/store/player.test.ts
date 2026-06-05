import { beforeEach, describe, expect, it } from "vitest";
import { usePlayer } from "./player";
import type { Song } from "../api/types";

const songs: Song[] = [
  { id: "a", title: "A" },
  { id: "b", title: "B" },
  { id: "c", title: "C" },
];

function reset() {
  usePlayer.setState({ queue: [], index: 0, repeat: "off", shuffle: false, isPlaying: false });
}

describe("playerStore", () => {
  beforeEach(reset);

  it("setQueue define fila e índice", () => {
    usePlayer.getState().setQueue(songs, 1);
    const s = usePlayer.getState();
    expect(s.queue.length).toBe(3);
    expect(s.index).toBe(1);
    expect(s.current()?.id).toBe("b");
  });

  it("nextIndex avança e respeita o fim sem repeat", () => {
    usePlayer.getState().setQueue(songs, 0);
    expect(usePlayer.getState().nextIndex()).toBe(1);
    usePlayer.getState().setIndex(2);
    expect(usePlayer.getState().nextIndex()).toBeNull();
  });

  it("repeat=all volta ao início no fim", () => {
    usePlayer.getState().setQueue(songs, 2);
    usePlayer.setState({ repeat: "all" });
    expect(usePlayer.getState().nextIndex()).toBe(0);
  });

  it("shuffle retorna índice dentro do intervalo", () => {
    usePlayer.getState().setQueue(songs, 0);
    usePlayer.setState({ shuffle: true });
    const n = usePlayer.getState().nextIndex();
    expect(n).toBeGreaterThanOrEqual(0);
    expect(n).toBeLessThan(3);
  });

  it("prevIndex não passa de zero", () => {
    usePlayer.getState().setQueue(songs, 0);
    expect(usePlayer.getState().prevIndex()).toBe(0);
    usePlayer.getState().setIndex(2);
    expect(usePlayer.getState().prevIndex()).toBe(1);
  });

  it("cycleRepeat percorre off→all→one→off", () => {
    const { cycleRepeat } = usePlayer.getState();
    cycleRepeat();
    expect(usePlayer.getState().repeat).toBe("all");
    cycleRepeat();
    expect(usePlayer.getState().repeat).toBe("one");
    cycleRepeat();
    expect(usePlayer.getState().repeat).toBe("off");
  });

  it("toggleShuffle e setVolume/setTime/setDuration", () => {
    usePlayer.getState().toggleShuffle();
    expect(usePlayer.getState().shuffle).toBe(true);
    usePlayer.getState().setVolume(0.5);
    usePlayer.getState().setTime(10);
    usePlayer.getState().setDuration(200);
    const s = usePlayer.getState();
    expect(s.volume).toBe(0.5);
    expect(s.currentTime).toBe(10);
    expect(s.duration).toBe(200);
  });

  it("nextIndex com fila vazia retorna null", () => {
    expect(usePlayer.getState().nextIndex()).toBeNull();
  });
});
