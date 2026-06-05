import { useState } from "react";
import { Endpoints } from "../api/client";

export function StarButton({ id, type, initial = false }: { id: string; type: string; initial?: boolean }) {
  const [starred, setStarred] = useState(initial);
  const [busy, setBusy] = useState(false);

  async function toggle(e: React.MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    const next = !starred;
    setStarred(next);
    try {
      await Endpoints.star(id, type, next);
    } catch {
      setStarred(!next); // reverte em erro
    } finally {
      setBusy(false);
    }
  }

  return (
    <button onClick={toggle} title={starred ? "Desfavoritar" : "Favoritar"} className="px-1">
      <span className={starred ? "text-berserker" : "text-neutral-500"}>{starred ? "★" : "☆"}</span>
    </button>
  );
}
