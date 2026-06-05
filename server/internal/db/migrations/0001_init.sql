-- Schema inicial do BerserkerPlayer (índice reconstruível a partir do scan).

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_admin      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artists (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    album_count INTEGER NOT NULL DEFAULT 0,
    song_count  INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_artists_name ON artists(name);

CREATE TABLE IF NOT EXISTS albums (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    artist_id    TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    year         INTEGER NOT NULL DEFAULT 0,
    genre        TEXT NOT NULL DEFAULT '',
    song_count   INTEGER NOT NULL DEFAULT 0,
    duration     INTEGER NOT NULL DEFAULT 0,
    cover_path   TEXT NOT NULL DEFAULT '',
    play_count   INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_albums_artist ON albums(artist_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_albums_artist_name ON albums(artist_id, name);

CREATE TABLE IF NOT EXISTS media_files (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    album_id    TEXT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    artist_id   TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    track       INTEGER NOT NULL DEFAULT 0,
    disc        INTEGER NOT NULL DEFAULT 0,
    duration    INTEGER NOT NULL DEFAULT 0,
    bit_rate    INTEGER NOT NULL DEFAULT 0,
    sample_rate INTEGER NOT NULL DEFAULT 0,
    suffix      TEXT NOT NULL DEFAULT '',
    size        INTEGER NOT NULL DEFAULT 0,
    mtime       INTEGER NOT NULL DEFAULT 0,
    has_embedded_cover INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_media_album ON media_files(album_id);
CREATE INDEX IF NOT EXISTS idx_media_artist ON media_files(artist_id);

CREATE TABLE IF NOT EXISTS playlists (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    owner_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_playlists_owner ON playlists(owner_id);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    media_id    TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    PRIMARY KEY (playlist_id, position)
);

-- Anotações por usuário (separadas dos metadados objetivos).
CREATE TABLE IF NOT EXISTS annotations (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id     TEXT NOT NULL,
    item_type   TEXT NOT NULL,
    starred_at  TEXT,
    rating      INTEGER NOT NULL DEFAULT 0,
    play_count  INTEGER NOT NULL DEFAULT 0,
    last_played TEXT,
    PRIMARY KEY (user_id, item_type, item_id)
);

-- Refresh tokens rotativos (hash do token + detecção de reuso).
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked    INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_tokens(user_id);

-- Busca full-text (FTS5) sobre faixas/álbuns/artistas.
CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
    item_id UNINDEXED,
    item_type UNINDEXED,
    text
);
