# BerserkerPlayer — Plano do Servidor

> Núcleo do sistema. Inspirado na arquitetura do [Navidrome](https://github.com/navidrome/navidrome):
> servidor em Go, índice em SQLite, scanner de biblioteca, streaming com transcodificação e
> API REST nativa + camada de compatibilidade Subsonic/OpenSubsonic.

Referência cruzada: [`00-initial-plan.md`](00-initial-plan.md)

---

## 1. Stack e Justificativa

| Item | Escolha | Por quê |
|---|---|---|
| Linguagem | **Go** | Binário único, concorrência nativa (scan/stream), baixo footprint — mesma escolha do Navidrome. |
| Banco | **SQLite** (modo WAL) | Zero-config, perfeito para single-host; índice reconstruível. Migração futura p/ Postgres possível. |
| HTTP router | `chi` ou `gin` | Middlewares, roteamento limpo. |
| Migrações | `goose` / `golang-migrate` | Versionamento de schema. |
| Tags de áudio | `dhowden/tag` + `ffprobe` (fallback) | Cobrir MP3/FLAC/OGG/M4A/etc. |
| Transcodificação | **ffmpeg** (processo externo) | Padrão de fato; pipe de saída por HTTP. |
| Auth | **JWT** (`golang-jwt`) + bcrypt/argon2 | Stateless, refresh tokens. |
| Config | `viper` + flags + env | Config em arquivo/env/flags. |
| Logs | `slog` (stdlib) ou `zerolog` | Estruturado, níveis. |

---

## 2. Estrutura de Pastas (proposta)

```
server/
├── cmd/berserker/main.go        # entrypoint
├── internal/
│   ├── config/                  # carregamento de config
│   ├── db/                      # conexão, migrações, queries
│   │   └── migrations/
│   ├── model/                   # entidades de domínio
│   ├── scanner/                 # walk de filesystem + extração de metadados
│   ├── core/                    # regras de negócio (library, playlists, users)
│   ├── api/
│   │   ├── nativeapi/           # REST /api/v1 (JSON moderna)
│   │   └── subsonic/            # /rest/* compatível Subsonic/OpenSubsonic
│   ├── auth/                    # JWT, hashing, middleware
│   ├── stream/                  # streaming + transcoder (ffmpeg)
│   └── artwork/                 # extração/cache de capas
├── go.mod
└── Dockerfile
```

---

## 3. Modelo de Dados (SQLite)

Entidades centrais (espelhando o Navidrome):

- **media_file** — uma faixa: path, title, album_id, artist_id, track/disc, duration, bitrate,
  sample_rate, suffix (ext), size, mtime, has_cover_art, mbz_*ids.
- **album** — agregado de faixas: name, album_artist_id, year, genre, song_count, duration,
  cover path, compilation flag.
- **artist** — name, album_count, song_count, mbz_id, biografia (opcional).
- **genre** — name (n:n com faixas/álbuns).
- **playlist** + **playlist_track** — ordenadas, com owner_id; suporte a smart playlists (regras).
- **user** — name, password_hash, is_admin, created_at, settings.
- **annotation** — (user_id, item_id, item_type) → starred, rating, play_count, last_played.
  Separa preferências por usuário dos metadados objetivos.
- **library / folder** — raízes monitoradas.
- **share** — links públicos temporários (fase avançada).
- **scrobble_buffer** — fila p/ Last.fm/ListenBrainz (fase avançada).

**Índices importantes:** `media_file(album_id)`, `media_file(path)`, `album(artist_id)`,
`annotation(user_id, item_type, item_id)`, full-text (FTS5) para busca em title/album/artist.

---

## 4. Scanner de Biblioteca

Fluxo (espelha o Navidrome):

1. **Walk** recursivo das pastas-raiz configuradas.
2. Para cada arquivo de áudio suportado, comparar `mtime`/`size` com o índice → **scan incremental**
   (só processa o que mudou).
3. **Extrair metadados** (tags) → normalizar (trim, capitalização de artistas, multi-valor de genre).
4. Resolver/derivar **álbuns e artistas** (incl. AlbumArtist x Artist, compilations).
5. Detectar **capa** (tag embutida ou `cover.jpg`/`folder.jpg` na pasta).
6. Persistir em transação; remover órfãos (arquivos deletados).
7. Atualizar contadores agregados (song_count, duration por álbum/artista).

**Modos:**
- Scan completo (boot inicial / `--full`).
- Scan incremental agendado (intervalo configurável).
- **Watcher** de filesystem (fsnotify) para rescan quase em tempo real (Fase 4).

**Concorrência:** pool de workers para parsing de tags; escrita serializada no SQLite (WAL).

---

## 5. API Nativa (`/api/v1`)

JSON moderno, autenticado por JWT (header `Authorization: Bearer`). Esboço de endpoints:

```
POST   /api/v1/auth/login            # { username, password } -> { accessToken, refreshToken? }
POST   /api/v1/auth/refresh          # rotação de refresh (corpo p/ iOS; cookie p/ WebApp)
POST   /api/v1/auth/media-token      # -> { token, expiresAt } p/ /stream e /cover
GET    /api/v1/me

GET    /api/v1/artists               # offset/limit + sort/order
GET    /api/v1/artists/{id}
GET    /api/v1/albums                # offset/limit, ?filter=recent|frequent|random|starred&genre=&artistId=
GET    /api/v1/albums/{id}           # inclui faixas
GET    /api/v1/songs/{id}
GET    /api/v1/search?q=             # FTS: artists+albums+songs

GET    /api/v1/playlists
POST   /api/v1/playlists
PUT    /api/v1/playlists/{id}        # reordenar/editar
DELETE /api/v1/playlists/{id}

POST   /api/v1/star                  # { id, type }  (favoritar)
DELETE /api/v1/star
POST   /api/v1/rating                # { id, type, rating }
POST   /api/v1/scrobble              # { songId, event: nowplaying|submission, playedAt }

GET    /api/v1/stream/{songId}?token=  # áudio; Range (direct play) | chunked (transcode)
GET    /api/v1/cover/{id}?token=       # artwork, ?size=
```

> **Paginação:** decidido **offset/limit + total** (envelope `Page`), não cursor. Cursor com
> sorts arbitrários (nome, ano, recentes, mais tocados, aleatório) é caro em SQLite; offset/limit
> casa com `infinite query` (TanStack Query) e permite jump-to-page. Limite máx. 200.
>
> **Auth de mídia:** `/stream` e `/cover` **não** usam header Bearer — um `<audio>` (browser) e o
> `AVPlayer` (iOS) não enviam headers customizados de forma confiável. Eles autenticam por
> `?token=` (token assinado e de curta duração emitido por `/auth/media-token`, vinculado ao
> usuário, escopo somente-leitura de mídia). Ver §8.
>
> **Erros:** `application/problem+json` (RFC 7807) com `{ type, title, status, detail }`.

**Contrato:** a fonte da verdade é [`../openapi.yaml`](../openapi.yaml) (OpenAPI 3.1) — gera tipos
para o WebApp (TS) e referência para o iOS. Testes de contrato no CI validam respostas contra ela.

---

## 6. Camada de Compatibilidade Subsonic / OpenSubsonic (`/rest/*`)

Implementar o subconjunto essencial do protocolo para reaproveitar clientes existentes e validar
o servidor:

- `ping`, `getLicense`
- `getArtists`, `getArtist`, `getAlbum`, `getAlbumList2`, `getSong`
- `search3`
- `getPlaylists`, `getPlaylist`, `createPlaylist`, `updatePlaylist`
- `stream`, `download`, `getCoverArt`
- `star`, `unstar`, `setRating`, `scrobble`
- Extensões **OpenSubsonic** (campos extra, `getOpenSubsonicExtensions`).

Auth Subsonic: parâmetros `u`, `t` (token = md5(password+salt)), `s` (salt), `c`, `v`, `f=json`.

> Esta camada compartilha o mesmo `core/` — apenas serializa para o formato Subsonic.

---

## 7. Streaming e Transcodificação

- **Direct play:** servir o arquivo com `Accept-Ranges: bytes`, respondendo a `Range` (HTTP 206)
  para seek e buffering parcial. É o caminho padrão (`format=raw`).
- **Transcodificação sob demanda:** quando o cliente pede `format`/`maxBitRate`:
  spawn de `ffmpeg` → stdout → resposta HTTP `200` **chunked** (`Transfer-Encoding: chunked`).
  Perfis configuráveis: `mp3-320`, `opus-128`, `aac-256`, etc.
  > ⚠️ **Range e transcode são incompatíveis:** a saída do ffmpeg tem tamanho desconhecido e não é
  > seekável por `Range`. Decisão de contrato: **seek em conteúdo transcodificado** é feito pelo
  > cliente **reabrindo** a stream com `?timeOffset=<segundos>`; o servidor inicia o ffmpeg a partir
  > daquele ponto (`-ss`). Um header `Range` em requisição transcodificada é ignorado.
- **Cache de transcode** (opcional): chave = (songId, perfil); LRU em disco. Coalescer requisições
  concorrentes do mesmo (songId, perfil) para um único processo ffmpeg.
- Encerrar processo ffmpeg ao desconectar o cliente (evitar zumbis); usar `context` cancelável.

---

## 8. Autenticação e Usuários

- Login → JWT (access curto, ~15 min + refresh longo). Refresh **rotativo** (cada refresh invalida
  o anterior; detectar reuso = possível roubo).
- **Entrega do refresh:** clientes nativos (iOS) recebem no corpo e guardam no Keychain; o WebApp
  servido pela mesma origem recebe via **cookie httpOnly `Secure SameSite=Strict`** (ver §9). Isso
  unifica a divergência entre os planos de cliente.
- **Token de mídia** (`/auth/media-token`): JWT/HMAC assinado de vida curta (poucos minutos),
  escopo `media:read`, vinculado ao `userID`. Aceito **apenas** em `/stream` e `/cover` via
  `?token=`, contornando a impossibilidade de header Bearer em `<audio>`/`AVPlayer`. Renovável.
- Middleware injeta `userID` no contexto; favoritos/playcount sempre por usuário.
- Admin: criar/editar/remover usuários, disparar rescan, ver status.
- Hash de senha com **argon2id** (texto puro nunca armazenado).
  > **Subsonic adiado (Fase 3):** o protocolo Subsonic exige `token = md5(password+salt)`, ou seja,
  > senha **recuperável** — incompatível com argon2id-only. Decisão: o MVP **não** suporta Subsonic;
  > quando/se ativado, usar um cofre de senha **criptografado reversível** separado, opt-in por
  > usuário, e deixar claro o trade-off de segurança.

---

## 9. Configuração e Operação

- Config via arquivo (`berserker.toml`), env (`BERSERKER_*`) e flags.
  Principais: `MusicFolder`, `DataFolder`, `Port`, `ScanInterval`, `TranscodingEnabled`,
  `LogLevel`, `BaseURL`.
- **Docker:** imagem multiarch (amd64/arm64); volume p/ música (ro) e p/ dados.
- **Servir o WebApp pelo próprio binário (origem única):** o servidor entrega o build estático do
  WebApp (ex.: `/` → SPA) na **mesma origem** da API. Isso **elimina CORS**, viabiliza o cookie
  httpOnly de refresh e dispensa configurar BaseURL no WebApp. Modelo do Navidrome. (O app iOS é
  cross-origin por natureza e usa refresh no corpo — não precisa de cookie.)
- **CORS:** desabilitado por padrão (origem única). Se um WebApp externo for usado, expor allowlist
  de origens configurável (`AllowedOrigins`) com credenciais.
- Healthcheck endpoint `/healthz`.
- Métricas (opcional): `/metrics` Prometheus.

---

## 10. Roadmap do Servidor

**Fase 0 — Fundações**
- [ ] Bootstrap do projeto Go, config, logging, `/healthz`.
- [ ] Schema + migrações; conexão SQLite (WAL).
- [ ] Scanner (full scan) com extração de tags e capa.
- [ ] Auth JWT (access+refresh rotativo) + token de mídia assinado + usuário admin via seed/config.
- [ ] Endpoint `/api/v1/stream` com suporte a Range (direct play) e auth por `?token=`.

**Fase 1 — API nativa completa (MVP)**
- [ ] Endpoints de biblioteca (artists/albums/songs/search) + paginação.
- [ ] Playlists CRUD.
- [ ] Artwork + thumbnails (resize/cache).
- [ ] OpenAPI publicado.

**Fase 2 — Suporte aos clientes**
- [ ] Favoritos, rating, playcount, scrobble interno.
- [ ] Ajustes a partir do feedback do WebApp/iOS.

**Fase 3 — Avançado**
- [ ] Transcodificação + perfis + cache.
- [ ] Camada Subsonic/OpenSubsonic.
- [ ] Multiusuário completo, scrobbling externo (Last.fm/ListenBrainz).
- [ ] Smart playlists.

**Fase 4 — Produção**
- [ ] Watcher fsnotify (rescan incremental em tempo real).
- [ ] Docker + CI/CD + releases.
- [ ] Testes (unit, integração, contrato), observabilidade.

---

## 11. Testes
- **Unit:** parsing de tags, normalização, regras de playlist.
- **Integração:** scan de uma fixture de biblioteca → asserts no índice.
- **Contrato:** validar respostas contra o OpenAPI e contra um cliente Subsonic real.
- **Streaming:** testes de `Range`/seek e de encerramento de ffmpeg.
