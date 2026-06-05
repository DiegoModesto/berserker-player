import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";

vi.mock("./api/client", () => ({
  login: vi.fn().mockResolvedValue(undefined),
  logout: vi.fn(),
  tryRestore: vi.fn(),
  isAuthenticated: vi.fn().mockReturnValue(false),
  coverURL: vi.fn().mockResolvedValue("blob:c"),
  streamURL: vi.fn().mockResolvedValue("blob:s"),
  Endpoints: {
    me: vi.fn(),
    albums: vi.fn().mockResolvedValue({ items: [], total: 0, offset: 0, limit: 40 }),
    scrobble: vi.fn().mockResolvedValue(undefined),
  },
}));

import * as client from "./api/client";
import { App } from "./App";
import { SessionProvider } from "./auth/session";
import { usePlayer } from "./store/player";

function renderRoot() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <SessionProvider>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </SessionProvider>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  usePlayer.setState({ queue: [], index: 0, isPlaying: false });
});

const albunsHeading = () => screen.getByRole("heading", { name: "Álbuns" });

describe("App + Session", () => {
  it("anônimo mostra login; após login mostra a biblioteca", async () => {
    (client.tryRestore as any).mockResolvedValue(false);
    (client.Endpoints.me as any).mockResolvedValue({ id: "u", username: "admin", isAdmin: true });
    renderRoot();

    // Tela de login.
    await waitFor(() => expect(screen.getByText("Entrar")).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText("Usuário"), { target: { value: "admin" } });
    fireEvent.change(screen.getByPlaceholderText("Senha"), { target: { value: "pw" } });
    fireEvent.click(screen.getByText("Entrar"));

    // Autenticado → layout com navegação.
    await waitFor(() => expect(albunsHeading()).toBeInTheDocument());
    expect(client.login).toHaveBeenCalledWith("admin", "pw");
  });

  it("sessão restaurada vai direto para a biblioteca e faz logout", async () => {
    (client.tryRestore as any).mockResolvedValue(true);
    (client.Endpoints.me as any).mockResolvedValue({ id: "u", username: "admin", isAdmin: true });
    renderRoot();

    await waitFor(() => expect(albunsHeading()).toBeInTheDocument());
    // Toggle de tema presente.
    fireEvent.click(screen.getByLabelText("Alternar tema"));
    // Logout volta para o login.
    fireEvent.click(screen.getByText("Sair"));
    await waitFor(() => expect(screen.getByText("Entrar")).toBeInTheDocument());
    expect(client.logout).toHaveBeenCalled();
  });

  it("login com erro mostra mensagem", async () => {
    (client.tryRestore as any).mockResolvedValue(false);
    (client.login as any).mockRejectedValueOnce(new Error("x"));
    renderRoot();
    await waitFor(() => expect(screen.getByText("Entrar")).toBeInTheDocument());
    fireEvent.change(screen.getByPlaceholderText("Usuário"), { target: { value: "a" } });
    fireEvent.change(screen.getByPlaceholderText("Senha"), { target: { value: "b" } });
    fireEvent.click(screen.getByText("Entrar"));
    await waitFor(() => expect(screen.getByText(/Falha no login/)).toBeInTheDocument());
  });
});
