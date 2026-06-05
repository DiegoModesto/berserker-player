import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Endpoints } from "../../api/client";

export function PlaylistsPage() {
  const { data, isLoading } = useQuery({ queryKey: ["playlists"], queryFn: Endpoints.playlists });

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Playlists</h1>
      {isLoading ? (
        <div className="text-neutral-500">Carregando…</div>
      ) : data && data.length > 0 ? (
        <ul className="divide-y divide-neutral-800">
          {data.map((pl) => (
            <li key={pl.id}>
              <Link to={`/playlist/${pl.id}`} className="flex items-center justify-between py-3 hover:text-berserker">
                <span>{pl.name}</span>
                <span className="text-sm text-neutral-500">{pl.songCount ?? 0} faixas</span>
              </Link>
            </li>
          ))}
        </ul>
      ) : (
        <div className="text-neutral-500">Nenhuma playlist ainda.</div>
      )}
    </div>
  );
}
