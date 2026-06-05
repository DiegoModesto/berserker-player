import { useEffect, useState } from "react";
import { coverURL } from "../api/client";

export function Cover({ coverArtId, size = 200, className = "" }: {
  coverArtId?: string;
  size?: number;
  className?: string;
}) {
  const [url, setUrl] = useState<string>();

  useEffect(() => {
    let active = true;
    if (coverArtId) {
      coverURL(coverArtId, size).then((u) => active && setUrl(u));
    } else {
      setUrl(undefined);
    }
    return () => {
      active = false;
    };
  }, [coverArtId, size]);

  return (
    <div
      className={`bg-neutral-800 rounded overflow-hidden flex items-center justify-center ${className}`}
      style={{ aspectRatio: "1 / 1" }}
    >
      {url ? (
        <img src={url} alt="" className="w-full h-full object-cover" loading="lazy" />
      ) : (
        <span className="text-neutral-500 text-2xl">♪</span>
      )}
    </div>
  );
}
