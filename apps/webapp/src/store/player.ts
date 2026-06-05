import { create } from "zustand";
import type { Song } from "../api/types";

export type RepeatMode = "off" | "all" | "one";

interface PlayerState {
  queue: Song[];
  index: number;
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  repeat: RepeatMode;
  shuffle: boolean;
  current: () => Song | undefined;
  setQueue: (songs: Song[], index: number) => void;
  setIndex: (index: number) => void;
  setPlaying: (playing: boolean) => void;
  setTime: (t: number) => void;
  setDuration: (d: number) => void;
  setVolume: (v: number) => void;
  cycleRepeat: () => void;
  toggleShuffle: () => void;
  nextIndex: () => number | null;
  prevIndex: () => number | null;
}

export const usePlayer = create<PlayerState>((set, get) => ({
  queue: [],
  index: 0,
  isPlaying: false,
  currentTime: 0,
  duration: 0,
  volume: 1,
  repeat: "off",
  shuffle: false,
  current: () => {
    const { queue, index } = get();
    return queue[index];
  },
  setQueue: (songs, index) => set({ queue: songs, index, currentTime: 0 }),
  setIndex: (index) => set({ index, currentTime: 0 }),
  setPlaying: (isPlaying) => set({ isPlaying }),
  setTime: (currentTime) => set({ currentTime }),
  setDuration: (duration) => set({ duration }),
  setVolume: (volume) => set({ volume }),
  cycleRepeat: () =>
    set((s) => ({ repeat: s.repeat === "off" ? "all" : s.repeat === "all" ? "one" : "off" })),
  toggleShuffle: () => set((s) => ({ shuffle: !s.shuffle })),
  nextIndex: () => {
    const { queue, index, repeat, shuffle } = get();
    if (queue.length === 0) return null;
    if (shuffle) return Math.floor(Math.random() * queue.length);
    if (index < queue.length - 1) return index + 1;
    return repeat === "all" ? 0 : null;
  },
  prevIndex: () => {
    const { index } = get();
    return index > 0 ? index - 1 : 0;
  },
}));
