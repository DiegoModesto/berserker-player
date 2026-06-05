import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { Endpoints } from "../../api/client";
import type { Song } from "../../api/types";
import { usePlayer } from "../../store/player";

function fmt(t?: number): string {
  if (!t) return "0:00";
  const m = Math.floor(t / 60);
  const s = Math.floor(t % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

/** Move item de `from` para `to` numa cópia do array. */
export function reorder<T>(list: T[], from: number, to: number): T[] {
  const copy = list.slice();
  const [moved] = copy.splice(from, 1);
  copy.splice(to, 0, moved);
  return copy;
}

export function PlaylistPage() {
  const { id = "" } = useParams();
  const { data } = useQuery({ queryKey: ["playlist", id], queryFn: () => Endpoints.playlist(id) });
  const [songs, setSongs] = useState<Song[]>([]);
  const dragIndex = useRef<number | null>(null);
  const setQueue = usePlayer((s) => s.setQueue);
  const setPlaying = usePlayer((s) => s.setPlaying);

  useEffect(() => {
    if (data) setSongs(data.songs);
  }, [data]);

  if (!data) return <div className="text-neutral-500">Carregando…</div>;

  const playFrom = (index: number) => {
    setQueue(songs, index);
    setPlaying(true);
  };

  const onDrop = async (to: number) => {
    const from = dragIndex.current;
    dragIndex.current = null;
    if (from === null || from === to) return;
    const next = reorder(songs, from, to);
    setSongs(next);
    try {
      await Endpoints.updatePlaylist(id, next.map((s) => s.id));
    } catch {
      setSongs(songs); // reverte
    }
  };

  return (
    <div>
      <div className="flex items-end justify-between mb-4">
        <div>
          <h1 className="text-3xl font-bold">{data.name}</h1>
          <div className="text-neutral-500 text-sm">{songs.length} faixas · arraste para reordenar</div>
        </div>
        {songs.length > 0 && (
          <button onClick={() => playFrom(0)} className="px-4 py-2 rounded bg-berserker text-white font-medium">
            ▶ Tocar
          </button>
        )}
      </div>
      <ol className="divide-y divide-neutral-200 dark:divide-neutral-800">
        {songs.map((song, i) => (
          <li
            key={`${song.id}-${i}`}
            draggable
            onDragStart={() => (dragIndex.current = i)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={() => onDrop(i)}
            className="flex items-center gap-4 py-2 px-2 hover:bg-neutral-100 dark:hover:bg-neutral-900"
          >
            <span className="cursor-grab text-neutral-400" title="Arraste">⠿</span>
            <button onClick={() => playFrom(i)} className="flex-1 truncate text-left">
              {song.title}
              <span className="text-neutral-500 text-sm"> · {song.artistName}</span>
            </button>
            <span className="text-neutral-500 text-sm tabular-nums">{fmt(song.duration)}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
