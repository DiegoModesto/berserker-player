import type {
  Album,
  AlbumDetail,
  Artist,
  Page,
  Playlist,
  SearchResult,
  Song,
  TokenPair,
  User,
} from "./types";

const BASE = "/api/v1";

// accessToken em memória; refreshToken vive em cookie httpOnly (origem única).
let accessToken: string | null = null;
let mediaToken: string | null = null;
let mediaTokenExpiry = 0;

export function isAuthenticated(): boolean {
  return accessToken !== null;
}

async function rawFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("X-Client", "webapp");
  if (init.body) headers.set("Content-Type", "application/json");
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  return fetch(BASE + path, { ...init, headers, credentials: "include" });
}

async function refresh(): Promise<boolean> {
  const res = await fetch(`${BASE}/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "X-Client": "webapp" },
  });
  if (!res.ok) {
    accessToken = null;
    return false;
  }
  const tp: TokenPair = await res.json();
  accessToken = tp.accessToken;
  return true;
}

async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res = await rawFetch(path, init);
  if (res.status === 401) {
    if (await refresh()) res = await rawFetch(path, init);
  }
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export async function login(username: string, password: string): Promise<void> {
  const res = await fetch(`${BASE}/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json", "X-Client": "webapp" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) throw new Error("login_failed");
  const tp: TokenPair = await res.json();
  accessToken = tp.accessToken;
}

export async function tryRestore(): Promise<boolean> {
  return refresh();
}

export function logout() {
  accessToken = null;
  mediaToken = null;
  mediaTokenExpiry = 0;
}

async function ensureMediaToken(): Promise<string> {
  if (mediaToken && mediaTokenExpiry - Date.now() > 60_000) return mediaToken;
  const r = await api<{ token: string; expiresAt: string }>("/auth/media-token", { method: "POST" });
  mediaToken = r.token;
  mediaTokenExpiry = new Date(r.expiresAt).getTime();
  return r.token;
}

export async function streamURL(songId: string): Promise<string> {
  const token = await ensureMediaToken();
  return `${BASE}/stream/${songId}?token=${encodeURIComponent(token)}`;
}

export async function coverURL(coverArtId: string, size = 300): Promise<string> {
  const token = await ensureMediaToken();
  return `${BASE}/cover/${coverArtId}?size=${size}&token=${encodeURIComponent(token)}`;
}

// ---- Endpoints ----

export const Endpoints = {
  me: () => api<User>("/me"),
  albums: (filter = "all", offset = 0, limit = 40) =>
    api<Page<Album>>(`/albums?filter=${filter}&offset=${offset}&limit=${limit}`),
  album: (id: string) => api<AlbumDetail>(`/albums/${id}`),
  artists: () => api<Page<Artist>>("/artists?limit=200"),
  search: (q: string) => api<SearchResult>(`/search?q=${encodeURIComponent(q)}`),
  playlists: () => api<Playlist[]>("/playlists"),
  star: (id: string, type: string, on: boolean) =>
    api<void>("/star", { method: on ? "POST" : "DELETE", body: JSON.stringify({ id, type }) }),
  scrobble: (songId: string, event: string) =>
    api<void>("/scrobble", { method: "POST", body: JSON.stringify({ songId, event }) }),
};

export type { Album, AlbumDetail, Artist, Page, Playlist, SearchResult, Song, User };
