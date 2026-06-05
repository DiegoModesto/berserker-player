import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { renderApp } from "../test/utils";

vi.mock("../api/client", () => ({
  Endpoints: {
    albums: vi.fn(),
    album: vi.fn(),
    playlist: vi.fn(),
    playlists: vi.fn(),
    search: vi.fn(),
    updatePlaylist: vi.fn().mockResolvedValue({}),
  },
  coverURL: vi.fn().mockResolvedValue("blob:cover"),
}));

import { Endpoints } from "../api/client";
import { LibraryPage } from "./library/LibraryPage";
import { AlbumPage } from "./album/AlbumPage";
import { PlaylistsPage } from "./playlists/PlaylistsPage";
import { PlaylistPage, reorder } from "./playlists/PlaylistPage";
import { SearchPage } from "./search/SearchPage";

beforeEach(() => vi.clearAllMocks());

describe("reorder", () => {
  it("move item de uma posição para outra", () => {
    expect(reorder([1, 2, 3], 0, 2)).toEqual([2, 3, 1]);
    expect(reorder(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
  });
});

describe("LibraryPage", () => {
  it("lista álbuns e troca de filtro", async () => {
    (Endpoints.albums as any).mockResolvedValue({
      items: [{ id: "al1", name: "Álbum Um", artistName: "Art" }],
      total: 1,
      offset: 0,
      limit: 40,
    });
    renderApp(<LibraryPage />);
    await waitFor(() => expect(screen.getByText("Álbum Um")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Recentes"));
    await waitFor(() => expect(Endpoints.albums).toHaveBeenCalledWith("recent", 0, 60));
  });
});

describe("AlbumPage", () => {
  it("mostra faixas e toca", async () => {
    (Endpoints.album as any).mockResolvedValue({
      id: "al1",
      name: "Álbum",
      artistName: "Art",
      songs: [{ id: "s1", title: "Faixa 1", duration: 100 }],
    });
    renderApp(
      <Routes>
        <Route path="/album/:id" element={<AlbumPage />} />
      </Routes>,
      { route: "/album/al1" },
    );
    await waitFor(() => expect(screen.getByText("Faixa 1")).toBeInTheDocument());
  });
});

describe("PlaylistsPage", () => {
  it("lista playlists", async () => {
    (Endpoints.playlists as any).mockResolvedValue([{ id: "p1", name: "Minha", songCount: 3 }]);
    renderApp(<PlaylistsPage />);
    await waitFor(() => expect(screen.getByText("Minha")).toBeInTheDocument());
  });

  it("estado vazio", async () => {
    (Endpoints.playlists as any).mockResolvedValue([]);
    renderApp(<PlaylistsPage />);
    await waitFor(() => expect(screen.getByText(/Nenhuma playlist/)).toBeInTheDocument());
  });
});

describe("PlaylistPage", () => {
  it("renderiza faixas e reordena por drag-and-drop", async () => {
    (Endpoints.playlist as any).mockResolvedValue({
      id: "p1",
      name: "Mix",
      songs: [
        { id: "s1", title: "Um", artistName: "A" },
        { id: "s2", title: "Dois", artistName: "B" },
      ],
    });
    renderApp(
      <Routes>
        <Route path="/playlist/:id" element={<PlaylistPage />} />
      </Routes>,
      { route: "/playlist/p1" },
    );
    await waitFor(() => expect(screen.getByText(/Um/)).toBeInTheDocument());

    const items = screen.getAllByRole("listitem");
    fireEvent.dragStart(items[0]);
    fireEvent.dragOver(items[1]);
    fireEvent.drop(items[1]);
    await waitFor(() =>
      expect(Endpoints.updatePlaylist).toHaveBeenCalledWith("p1", ["s2", "s1"]),
    );
  });
});

describe("SearchPage", () => {
  it("busca com debounce e mostra resultados", async () => {
    (Endpoints.search as any).mockResolvedValue({
      artists: [],
      albums: [{ id: "al1", name: "Achado", artistName: "Art" }],
      songs: [],
    });
    renderApp(<SearchPage />);
    fireEvent.change(screen.getByPlaceholderText(/Artistas/), { target: { value: "ach" } });
    await waitFor(() => expect(screen.getByText("Achado")).toBeInTheDocument(), { timeout: 2000 });
  });
});
