import { NavLink, Outlet } from "react-router-dom";
import { useSession } from "../auth/session";
import { AudioController } from "../player/AudioController";
import { PlayerBar } from "./PlayerBar";

export function Layout() {
  const { user, logout } = useSession();
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `block px-4 py-2 rounded ${isActive ? "bg-neutral-800 text-white" : "text-neutral-400 hover:text-white"}`;

  return (
    <div className="h-screen flex flex-col bg-neutral-950 text-neutral-100">
      <div className="flex flex-1 min-h-0">
        <aside className="w-56 border-r border-neutral-800 p-4 flex flex-col gap-1">
          <div className="text-berserker font-bold text-lg px-4 mb-4">Berserker</div>
          <NavLink to="/" className={linkClass} end>
            Álbuns
          </NavLink>
          <NavLink to="/search" className={linkClass}>
            Buscar
          </NavLink>
          <div className="mt-auto text-xs text-neutral-500 px-4">
            <div className="truncate">{user?.username}</div>
            <button onClick={logout} className="mt-1 hover:text-white">
              Sair
            </button>
          </div>
        </aside>
        <main className="flex-1 overflow-y-auto p-6 pb-28">
          <Outlet />
        </main>
      </div>
      <PlayerBar />
      <AudioController />
    </div>
  );
}
