import { create } from "zustand";

export type ThemeMode = "light" | "dark";

const STORAGE_KEY = "theme";

export function initialTheme(): ThemeMode {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") return saved;
  return "dark"; // padrão
}

/** Aplica o tema ao <html> (classe 'dark' usada pelo Tailwind). */
export function applyTheme(mode: ThemeMode) {
  const root = document.documentElement;
  root.classList.toggle("dark", mode === "dark");
}

interface ThemeState {
  mode: ThemeMode;
  toggle: () => void;
  set: (mode: ThemeMode) => void;
}

export const useTheme = create<ThemeState>((set, get) => ({
  mode: initialTheme(),
  toggle: () => get().set(get().mode === "dark" ? "light" : "dark"),
  set: (mode) => {
    localStorage.setItem(STORAGE_KEY, mode);
    applyTheme(mode);
    set({ mode });
  },
}));
