-- Smart playlists: playlists dinâmicas definidas por regras (avaliadas na leitura).
ALTER TABLE playlists ADD COLUMN is_smart INTEGER NOT NULL DEFAULT 0;
ALTER TABLE playlists ADD COLUMN rules TEXT NOT NULL DEFAULT '';
