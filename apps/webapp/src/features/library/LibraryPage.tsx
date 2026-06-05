import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Endpoints } from "../../api/client";
import { Cover } from "../../components/Cover";

const FILTERS = [
  { key: "all", label: "Todos" },
  { key: "recent", label: "Recentes" },
  { key: "frequent", label: "Mais tocados" },
  { key: "random", label: "Aleatório" },
];

export function LibraryPage() {
  const [filter, setFilter] = useState("all");
  const { data, isLoading } = useQuery({
    queryKey: ["albums", filter],
    queryFn: () => Endpoints.albums(filter, 0, 60),
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">Álbuns</h1>
        <div className="flex gap-2">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              className={`px-3 py-1 rounded text-sm ${
                filter === f.key ? "bg-berserker text-white" : "bg-neutral-800 text-neutral-300"
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="text-neutral-500">Carregando…</div>
      ) : (
        <div className="grid gap-4" style={{ gridTemplateColumns: "repeat(auto-fill, minmax(150px, 1fr))" }}>
          {data?.items.map((album) => (
            <Link key={album.id} to={`/album/${album.id}`} className="group">
              <Cover coverArtId={album.coverArtId ?? album.id} size={300} className="mb-2 group-hover:opacity-80" />
              <div className="text-sm font-medium truncate">{album.name}</div>
              <div className="text-xs text-neutral-400 truncate">{album.artistName}</div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
