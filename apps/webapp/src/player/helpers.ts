import { Endpoints, coverURL, streamURL } from "../api/client";

export { coverURL, streamURL };

/** Scrobble que nunca lança (telemetria não deve quebrar a reprodução). */
export async function scrobbleSafe(songId: string, event: string): Promise<void> {
  try {
    await Endpoints.scrobble(songId, event);
  } catch {
    /* ignora */
  }
}
