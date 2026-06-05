// Tipos espelhando openapi.yaml. (Geração automática disponível via `npm run gen:api`.)

export interface User {
  id: string;
  username: string;
  isAdmin: boolean;
}

export interface Artist {
  id: string;
  name: string;
  albumCount?: number;
  songCount?: number;
  starred?: boolean;
  coverArtId?: string;
}

export interface Album {
  id: string;
  name: string;
  artistId?: string;
  artistName?: string;
  year?: number;
  genre?: string;
  songCount?: number;
  duration?: number;
  coverArtId?: string;
  starred?: boolean;
  playCount?: number;
}

export interface Song {
  id: string;
  title: string;
  albumId?: string;
  albumName?: string;
  artistId?: string;
  artistName?: string;
  track?: number;
  disc?: number;
  duration?: number;
  suffix?: string;
  coverArtId?: string;
  starred?: boolean;
  rating?: number;
  playCount?: number;
}

export interface Playlist {
  id: string;
  name: string;
  songCount?: number;
  duration?: number;
}

export interface Page<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
}

export interface AlbumDetail extends Album {
  songs: Song[];
}

export interface PlaylistDetail extends Playlist {
  songs: Song[];
}

export interface SearchResult {
  artists: Artist[];
  albums: Album[];
  songs: Song[];
}

export interface TokenPair {
  accessToken: string;
  refreshToken?: string;
  expiresAt?: string;
}
