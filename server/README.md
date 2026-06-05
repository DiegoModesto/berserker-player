# BerserkerPlayer — Servidor

Servidor em Go: core, API nativa (`/api/v1`), scanner de biblioteca, streaming e artwork.
Índice em SQLite (modo WAL), SQLite puro em Go (`modernc.org/sqlite`, sem CGO).

## Requisitos
- Go 1.22+
- `ffmpeg` / `ffprobe` no PATH (extração de duração/bitrate; transcodificação na fase avançada)

## Rodar

```bash
go run ./cmd/berserker --music /caminho/da/musica --data ./data --admin-password trocar123
# ou via arquivo:
go run ./cmd/berserker --config berserker.example.toml
```

Na primeira execução um usuário `admin` é criado. Se `--admin-password` não for
informado, uma senha é gerada e **logada uma única vez** (troque-a).

## Endpoints (Fase 0)
- `GET  /healthz`
- `POST /api/v1/auth/login` → `{ accessToken, refreshToken?, expiresAt }`
- `POST /api/v1/auth/refresh` (rotação; cookie httpOnly p/ webapp, corpo p/ nativo)
- `POST /api/v1/auth/media-token` → token assinado p/ mídia
- `GET  /api/v1/me`
- `GET  /api/v1/stream/{id}?token=` — direct play com Range/206
- `GET  /api/v1/cover/{id}?token=&size=` — artwork (com resize/cache)
- `POST /api/v1/admin/scan` · `GET /api/v1/admin/scan/status` (admin)

As rotas de biblioteca/playlists/anotações entram na Fase 1.

## Testes
```bash
go test ./...    # inclui integração do scanner (gera áudio via ffmpeg; pula se ausente)
```

## Estrutura
```
cmd/berserker/        entrypoint
internal/config/      configuração (toml+env+flags)
internal/db/          conexão SQLite + migrações embarcadas
internal/model/       entidades de domínio
internal/core/        repositórios + regras (store, users, tokens, library)
internal/auth/        argon2id + JWT (access/media) + refresh tokens
internal/scanner/     walk + tags (dhowden/tag) + ffprobe
internal/stream/      direct play (Range)
internal/artwork/     capas + resize/cache
internal/api/nativeapi/  HTTP /api/v1 (chi)
```
