import { useState } from "react";
import { useSession } from "../../auth/session";

export function LoginPage() {
  const { login } = useSession();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(undefined);
    setBusy(true);
    try {
      await login(username, password);
    } catch {
      setError("Falha no login. Verifique usuário e senha.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="h-screen flex items-center justify-center bg-neutral-950 text-neutral-100">
      <form onSubmit={submit} className="w-80 flex flex-col gap-3">
        <h1 className="text-3xl font-bold text-center text-berserker">Berserker</h1>
        <input
          className="px-3 py-2 rounded bg-neutral-800 border border-neutral-700"
          placeholder="Usuário"
          value={username}
          autoComplete="username"
          onChange={(e) => setUsername(e.target.value)}
        />
        <input
          className="px-3 py-2 rounded bg-neutral-800 border border-neutral-700"
          type="password"
          placeholder="Senha"
          value={password}
          autoComplete="current-password"
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <div className="text-red-400 text-sm">{error}</div>}
        <button
          disabled={busy || !username || !password}
          className="px-3 py-2 rounded bg-berserker text-white font-medium disabled:opacity-50"
        >
          {busy ? "Entrando…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
