import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/client", () => ({
  Endpoints: {
    star: vi.fn().mockResolvedValue(undefined),
    rating: vi.fn().mockResolvedValue(undefined),
  },
  coverURL: vi.fn().mockResolvedValue("blob:cover"),
}));

import { Endpoints, coverURL } from "../api/client";
import { StarButton } from "./StarButton";
import { StarRating } from "./StarRating";
import { Cover } from "./Cover";
import { PlayerBar } from "./PlayerBar";
import { usePlayer } from "../store/player";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("StarButton", () => {
  it("alterna e chama Endpoints.star", async () => {
    render(<StarButton id="s1" type="song" initial={false} />);
    const btn = screen.getByRole("button");
    expect(btn.textContent).toBe("☆");
    fireEvent.click(btn);
    await waitFor(() => expect(btn.textContent).toBe("★"));
    expect(Endpoints.star).toHaveBeenCalledWith("s1", "song", true);
  });
});

describe("StarRating", () => {
  it("define rating ao clicar numa estrela", async () => {
    render(<StarRating id="s1" type="song" initial={0} />);
    const stars = screen.getAllByRole("button");
    expect(stars).toHaveLength(5);
    fireEvent.click(stars[2]); // 3ª estrela
    await waitFor(() => expect(Endpoints.rating).toHaveBeenCalledWith("s1", "song", 3));
  });

  it("clicar na estrela atual limpa (0)", async () => {
    render(<StarRating id="s1" type="song" initial={3} />);
    const stars = screen.getAllByRole("button");
    fireEvent.click(stars[2]);
    await waitFor(() => expect(Endpoints.rating).toHaveBeenCalledWith("s1", "song", 0));
  });
});

describe("Cover", () => {
  it("mostra placeholder e depois a imagem", async () => {
    render(<Cover coverArtId="alb1" size={100} />);
    await waitFor(() => expect(coverURL).toHaveBeenCalledWith("alb1", 100));
  });

  it("sem coverArtId não chama coverURL", () => {
    render(<Cover size={100} />);
    expect(coverURL).not.toHaveBeenCalled();
  });
});

describe("PlayerBar", () => {
  it("não renderiza sem faixa atual", () => {
    usePlayer.setState({ queue: [], index: 0 });
    const { container } = render(<PlayerBar />);
    expect(container.firstChild).toBeNull();
  });

  it("renderiza e opera todos os controles", () => {
    usePlayer.setState({
      queue: [
        { id: "a", title: "Faixa A", artistName: "Artista" },
        { id: "b", title: "Faixa B" },
      ],
      index: 0,
      isPlaying: false,
      duration: 100,
      currentTime: 0,
      repeat: "off",
      shuffle: false,
    });
    render(<PlayerBar />);
    expect(screen.getByText("Faixa A")).toBeInTheDocument();

    fireEvent.click(screen.getByTitle("Tocar"));
    expect(usePlayer.getState().isPlaying).toBe(true);

    fireEvent.click(screen.getByTitle("Próxima"));
    expect(usePlayer.getState().index).toBe(1);
    fireEvent.click(screen.getByTitle("Anterior"));
    expect(usePlayer.getState().index).toBe(0);

    fireEvent.click(screen.getByTitle("Aleatório"));
    expect(usePlayer.getState().shuffle).toBe(true);
    fireEvent.click(screen.getByTitle("Repetir"));
    expect(usePlayer.getState().repeat).toBe("all");

    // Seek pela barra.
    const slider = screen.getByRole("slider");
    fireEvent.change(slider, { target: { value: "30" } });
    expect(usePlayer.getState().currentTime).toBe(30);
  });
});
