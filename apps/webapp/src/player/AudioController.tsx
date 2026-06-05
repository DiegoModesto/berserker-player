import { useEffect, useRef } from "react";
import { usePlayer } from "../store/player";
import { coverURL, scrobbleSafe, streamURL } from "./helpers";

/**
 * AudioController: elemento <audio> único controlado pelo playerStore.
 * Sincroniza tempo/duração, integra com a Media Session API e dispara scrobble.
 */
export function AudioController() {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const scrobbledRef = useRef(false);

  const { index, queue, isPlaying, volume, repeat } = usePlayer();
  const setTime = usePlayer((s) => s.setTime);
  const setDuration = usePlayer((s) => s.setDuration);
  const setPlaying = usePlayer((s) => s.setPlaying);
  const setIndex = usePlayer((s) => s.setIndex);
  const nextIndex = usePlayer((s) => s.nextIndex);
  const prevIndex = usePlayer((s) => s.prevIndex);

  const current = queue[index];

  // Carrega a faixa atual.
  useEffect(() => {
    const audio = audioRef.current;
    if (!audio || !current) return;
    let cancelled = false;
    scrobbledRef.current = false;
    (async () => {
      const url = await streamURL(current.id);
      if (cancelled) return;
      audio.src = url;
      if (isPlaying) await audio.play().catch(() => setPlaying(false));
      void scrobbleSafe(current.id, "nowplaying");
      void updateMediaSession(current);
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current?.id]);

  // Play/pause reativo.
  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;
    if (isPlaying) audio.play().catch(() => setPlaying(false));
    else audio.pause();
  }, [isPlaying, setPlaying]);

  useEffect(() => {
    if (audioRef.current) audioRef.current.volume = volume;
  }, [volume]);

  function onTimeUpdate() {
    const audio = audioRef.current;
    if (!audio) return;
    setTime(audio.currentTime);
    if (!scrobbledRef.current && audio.duration > 0 && audio.currentTime / audio.duration >= 0.5) {
      scrobbledRef.current = true;
      if (current) void scrobbleSafe(current.id, "submission");
    }
  }

  function onEnded() {
    if (repeat === "one") {
      const a = audioRef.current;
      if (a) {
        a.currentTime = 0;
        void a.play();
      }
      return;
    }
    const ni = nextIndex();
    if (ni !== null) setIndex(ni);
    else setPlaying(false);
  }

  // Media Session: teclas de mídia / central do SO.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    navigator.mediaSession.setActionHandler("play", () => setPlaying(true));
    navigator.mediaSession.setActionHandler("pause", () => setPlaying(false));
    navigator.mediaSession.setActionHandler("nexttrack", () => {
      const ni = nextIndex();
      if (ni !== null) setIndex(ni);
    });
    navigator.mediaSession.setActionHandler("previoustrack", () => {
      const pi = prevIndex();
      if (pi !== null) setIndex(pi);
    });
  }, [nextIndex, prevIndex, setIndex, setPlaying]);

  return (
    <audio
      ref={audioRef}
      onTimeUpdate={onTimeUpdate}
      onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
      onPlay={() => setPlaying(true)}
      onPause={() => setPlaying(false)}
      onEnded={onEnded}
      hidden
    />
  );
}

async function updateMediaSession(song: { title: string; artistName?: string; albumName?: string; coverArtId?: string }) {
  if (!("mediaSession" in navigator)) return;
  const artwork: MediaImage[] = [];
  if (song.coverArtId) {
    const url = await coverURL(song.coverArtId, 512);
    artwork.push({ src: url, sizes: "512x512", type: "image/jpeg" });
  }
  navigator.mediaSession.metadata = new MediaMetadata({
    title: song.title,
    artist: song.artistName ?? "",
    album: song.albumName ?? "",
    artwork,
  });
}
