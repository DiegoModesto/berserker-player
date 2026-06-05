import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { Endpoints } from "../../api/client";
import { Cover } from "../../components/Cover";
import { usePlayer } from "../../store/player";

function fmt(t?: number): string {
  if (!t) return "0:00";
  const m = Math.floor(t / 60);
  const s = Math.floor(t % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export function AlbumPage() {
  const { id = "" } = useParams();
  const { data } = useQuery({ queryKey: ["album", id], queryFn: () => Endpoints.album(id) });
  const setQueue = usePlayer((s) => s.setQueue);
  const setPlaying = usePlayer((s) => s.setPlaying);

  if (!data) return <div className="text-neutral-500">Carregando…</div>;

  const playFrom = (index: number) => {
    setQueue(data.songs, index);
    setPlaying(true);
  };

  return (
    <div>
      <div className="flex gap-6 mb-6">
        <Cover coverArtId={data.coverArtId ?? data.id} size={400} className="w-48 h-48 shrink-0" />
        <div className="flex flex-col justify-end">
          <h1 className="text-3xl font-bold">{data.name}</h1>
          <div className="text-neutral-400">{data.artistName}</div>
          {data.year ? <div className="text-neutral-500 text-sm">{data.year}</div> : null}
          <button
            onClick={() => playFrom(0)}
            className="mt-3 w-32 px-4 py-2 rounded bg-berserker text-white font-medium"
          >
            ▶ Tocar
          </button>
        </div>
      </div>

      <ol className="divide-y divide-neutral-800">
        {data.songs.map((song, i) => (
          <li key={song.id}>
            <button
              onClick={() => playFrom(i)}
              className="w-full flex items-center gap-4 py-2 px-2 hover:bg-neutral-900 text-left"
            >
              <span className="w-6 text-right text-neutral-500 tabular-nums">{song.track ?? i + 1}</span>
              <span className="flex-1 truncate">{song.title}</span>
              <span className="text-neutral-500 text-sm tabular-nums">{fmt(song.duration)}</span>
            </button>
          </li>
        ))}
      </ol>
    </div>
  );
}
