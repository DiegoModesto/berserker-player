import { usePlayer } from "../store/player";
import { Cover } from "./Cover";

function fmt(t: number): string {
  if (!isFinite(t)) return "0:00";
  const m = Math.floor(t / 60);
  const s = Math.floor(t % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export function PlayerBar() {
  const { queue, index, isPlaying, currentTime, duration, repeat, shuffle } = usePlayer();
  const setPlaying = usePlayer((s) => s.setPlaying);
  const setIndex = usePlayer((s) => s.setIndex);
  const setTime = usePlayer((s) => s.setTime);
  const nextIndex = usePlayer((s) => s.nextIndex);
  const prevIndex = usePlayer((s) => s.prevIndex);
  const cycleRepeat = usePlayer((s) => s.cycleRepeat);
  const toggleShuffle = usePlayer((s) => s.toggleShuffle);

  const current = queue[index];
  if (!current) return null;

  return (
    <div className="fixed bottom-0 left-0 right-0 h-20 bg-neutral-100 dark:bg-neutral-900 border-t border-neutral-200 dark:border-neutral-800 flex items-center px-4 gap-4">
      <div className="flex items-center gap-3 w-64 min-w-0">
        <Cover coverArtId={current.coverArtId} size={56} className="w-14 h-14 shrink-0" />
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{current.title}</div>
          <div className="truncate text-xs text-neutral-400">{current.artistName}</div>
        </div>
      </div>

      <div className="flex-1 flex flex-col items-center gap-1">
        <div className="flex items-center gap-4">
          <button
            className={shuffle ? "text-berserker" : "text-neutral-400"}
            onClick={toggleShuffle}
            title="Aleatório"
          >
            ⇄
          </button>
          <button
            onClick={() => {
              const pi = prevIndex();
              if (pi !== null) setIndex(pi);
            }}
            title="Anterior"
          >
            ⏮
          </button>
          <button
            className="w-9 h-9 rounded-full bg-white text-black flex items-center justify-center"
            onClick={() => setPlaying(!isPlaying)}
            title={isPlaying ? "Pausar" : "Tocar"}
          >
            {isPlaying ? "⏸" : "▶"}
          </button>
          <button
            onClick={() => {
              const ni = nextIndex();
              if (ni !== null) setIndex(ni);
            }}
            title="Próxima"
          >
            ⏭
          </button>
          <button
            className={repeat !== "off" ? "text-berserker" : "text-neutral-400"}
            onClick={cycleRepeat}
            title="Repetir"
          >
            {repeat === "one" ? "🔂" : "🔁"}
          </button>
        </div>
        <div className="flex items-center gap-2 w-full max-w-xl text-xs text-neutral-400">
          <span className="tabular-nums w-10 text-right">{fmt(currentTime)}</span>
          <input
            type="range"
            min={0}
            max={duration || 0}
            value={currentTime}
            onChange={(e) => {
              const v = Number(e.target.value);
              setTime(v);
              const audio = document.querySelector("audio");
              if (audio) audio.currentTime = v;
            }}
            className="flex-1 accent-berserker"
          />
          <span className="tabular-nums w-10">{fmt(duration)}</span>
        </div>
      </div>

      <div className="w-64" />
    </div>
  );
}
