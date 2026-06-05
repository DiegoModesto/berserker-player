import { useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Endpoints } from "../../api/client";
import { Cover } from "../../components/Cover";
import { usePlayer } from "../../store/player";

export function SearchPage() {
  const [q, setQ] = useState("");
  const setQueue = usePlayer((s) => s.setQueue);
  const setPlaying = usePlayer((s) => s.setPlaying);

  const { data } = useQuery({
    queryKey: ["search", q],
    queryFn: () => Endpoints.search(q),
    enabled: q.trim().length >= 2,
    placeholderData: keepPreviousData,
  });

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Buscar</h1>
      <input
        autoFocus
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Artistas, álbuns, faixas"
        className="w-full max-w-lg px-3 py-2 rounded bg-neutral-800 border border-neutral-700 mb-6"
      />

      {data && (
        <div className="space-y-6">
          {data.albums.length > 0 && (
            <section>
              <h2 className="text-sm uppercase text-neutral-500 mb-2">Álbuns</h2>
              <div className="grid gap-4" style={{ gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))" }}>
                {data.albums.map((a) => (
                  <Link key={a.id} to={`/album/${a.id}`}>
                    <Cover coverArtId={a.coverArtId ?? a.id} size={280} className="mb-1" />
                    <div className="text-sm truncate">{a.name}</div>
                    <div className="text-xs text-neutral-400 truncate">{a.artistName}</div>
                  </Link>
                ))}
              </div>
            </section>
          )}
          {data.songs.length > 0 && (
            <section>
              <h2 className="text-sm uppercase text-neutral-500 mb-2">Faixas</h2>
              <ul className="divide-y divide-neutral-800">
                {data.songs.map((s) => (
                  <li key={s.id}>
                    <button
                      onClick={() => {
                        setQueue([s], 0);
                        setPlaying(true);
                      }}
                      className="w-full flex items-center gap-3 py-2 px-2 hover:bg-neutral-900 text-left"
                    >
                      <Cover coverArtId={s.coverArtId} size={40} className="w-10 h-10 shrink-0" />
                      <div className="min-w-0">
                        <div className="truncate">{s.title}</div>
                        <div className="text-xs text-neutral-400 truncate">{s.artistName}</div>
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>
      )}
    </div>
  );
}
