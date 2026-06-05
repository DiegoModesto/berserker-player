import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { Endpoints, login as apiLogin, logout as apiLogout, tryRestore } from "../api/client";
import type { User } from "../api/types";

interface SessionContextValue {
  user: User | null;
  status: "loading" | "authenticated" | "anonymous";
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [status, setStatus] = useState<SessionContextValue["status"]>("loading");

  useEffect(() => {
    (async () => {
      if (await tryRestore()) {
        try {
          setUser(await Endpoints.me());
          setStatus("authenticated");
          return;
        } catch {
          /* cai para anônimo */
        }
      }
      setStatus("anonymous");
    })();
  }, []);

  async function login(username: string, password: string) {
    await apiLogin(username, password);
    setUser(await Endpoints.me());
    setStatus("authenticated");
  }

  function logout() {
    apiLogout();
    setUser(null);
    setStatus("anonymous");
  }

  return (
    <SessionContext.Provider value={{ user, status, login, logout }}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession fora de SessionProvider");
  return ctx;
}
