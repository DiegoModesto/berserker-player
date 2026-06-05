import { useState } from "react";
import { Endpoints } from "../api/client";

/** Rating de 1 a 5 estrelas; 0 limpa. Persiste via /rating. */
export function StarRating({ id, type, initial = 0 }: { id: string; type: string; initial?: number }) {
  const [rating, setRating] = useState(initial);
  const [hover, setHover] = useState(0);

  async function set(value: number, e: React.MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    const next = value === rating ? 0 : value; // clicar na mesma estrela limpa
    const prev = rating;
    setRating(next);
    try {
      await Endpoints.rating(id, type, next);
    } catch {
      setRating(prev);
    }
  }

  return (
    <span className="inline-flex" onMouseLeave={() => setHover(0)}>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          onClick={(e) => set(n, e)}
          onMouseEnter={() => setHover(n)}
          title={`${n} estrela(s)`}
          className={(hover || rating) >= n ? "text-berserker" : "text-neutral-400"}
          aria-label={`Avaliar ${n}`}
        >
          ★
        </button>
      ))}
    </span>
  );
}
