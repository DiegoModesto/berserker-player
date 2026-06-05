# BerserkerPlayer — Plano do WebApp

> Cliente web (SPA) para o servidor BerserkerPlayer. Inspirado na UI web do
> [Navidrome](https://github.com/navidrome/navidrome) (que usa React + react-admin), mas com
> stack moderna e foco em player responsivo. Consome a API nativa (`/api/v1`).
>
> **Prioridade:** cliente **secundário** do MVP (o iOS é o primário — ver
> [`02-ios-app-plan.md`](02-ios-app-plan.md)). É **servido pelo próprio binário do servidor**
> (origem única). Tipos gerados de [`../openapi.yaml`](../openapi.yaml).

Referência cruzada: [`00-initial-plan.md`](00-initial-plan.md) · [`01-server-plan.md`](01-server-plan.md)

---

## 1. Stack e Justificativa

| Item | Escolha | Por quê |
|---|---|---|
| Framework | **React 18 + TypeScript** | Ecossistema maduro; mesma família do Navidrome. |
| Build | **Vite** | Dev rápido, build otimizado. |
| Roteamento | **React Router** | SPA com rotas aninhadas. |
| Estado servidor | **TanStack Query** | Cache, paginação infinita, revalidação — ideal p/ API. |
| Estado cliente | **Zustand** | Estado do player (fila, faixa atual) leve e global. |
| UI | **Tailwind CSS** + **shadcn/ui** (ou MUI) | Produtividade e consistência visual. |
| Áudio | **HTML5 `<audio>`** + (opcional) **Web Audio API** | Streaming, seek; Web Audio p/ visualizações/EQ. |
| Tipos da API | gerados do **OpenAPI** do servidor | Contrato único, type-safety. |
| i18n | `react-i18next` | PT/EN. |
| PWA | service worker (Workbox) | Instalável; cache de assets; mídia offline (avançado). |

> O Navidrome usa `react-admin` para acelerar telas CRUD. Para o BerserkerPlayer optamos por uma
> stack mais "à mão" (Query + Zustand + componentes) para liberdade total no player e na UX.
> A camada de admin (usuários, status do scan) pode reutilizar componentes simples de tabela.

---

## 2. Arquitetura

```
webapp/
├── src/
│   ├── api/            # cliente fetch, tipos gerados (OpenAPI), hooks (useAlbums, useSong...)
│   ├── auth/           # contexto de sessão, refresh de token, guarda de rotas
│   ├── store/          # Zustand: playerStore (fila, índice, status, volume, modos)
│   ├── player/         # AudioController (wraps <audio>), Now Playing, Media Session API
│   ├── features/
│   │   ├── library/    # Albums, Artists, Genres
│   │   ├── album/      # detalhe
│   │   ├── search/
│   │   ├── playlists/
│   │   └── admin/      # usuários, status do servidor (admin)
│   ├── components/     # PlayerBar, TrackList, AlbumCard, Queue, etc.
│   ├── routes.tsx
│   └── main.tsx
├── index.html
└── vite.config.ts
```

**Fluxo de dados:** componentes usam hooks do TanStack Query (dados do servidor) e o `playerStore`
(Zustand) para o estado do player. O `AudioController` é um singleton que controla o elemento
`<audio>` e sincroniza com o store.

---

## 3. Layout / UX

```
┌───────────────────────────────────────────────────────────┐
│  Sidebar    │           Conteúdo principal                 │
│  - Início   │   (grade de álbuns / detalhe / busca / ...)   │
│  - Álbuns   │                                               │
│  - Artistas │                                               │
│  - Playlists│                                               │
│  - Admin    │                                               │
├───────────────────────────────────────────────────────────┤
│  PlayerBar (fixa): capa | título/artista | ◀ ▶ ⏯ | scrubber│
│                    | volume | fila | repeat/shuffle         │
└───────────────────────────────────────────────────────────┘
```

- **Responsivo:** sidebar colapsa em drawer no mobile; PlayerBar vira mini-player.
- **Tema claro/escuro.**

---

## 4. Funcionalidades

- **Login:** usuário/senha (origem única — sem campo de URL do servidor) → `accessToken` em
  **memória** + `refreshToken` em **cookie httpOnly** definido pelo servidor (ver §6). Sem token em
  `localStorage`.
- **Navegação:** álbuns (grade com capas), artistas, gêneros; filtros (recentes, mais tocados,
  aleatórios); **paginação offset/limit** consumida como infinite query (TanStack Query) usando o
  `total` do envelope `Page`.
- **Detalhe de álbum/artista:** lista de faixas, tocar tudo, adicionar à fila, favoritar.
- **Busca:** unificada com debounce.
- **Player:** play/pause, próxima/anterior, **seek** (scrubber), volume, **shuffle**, **repeat**
  (off/all/one), fila visível e reordenável (drag-and-drop).
- **Playlists:** criar, editar, reordenar, remover; arrastar faixas.
- **Favoritos / rating / playcount.**
- **Media Session API:** integra com controles de mídia do SO/teclado (play/pause/next nas
  teclas de mídia e na central do sistema operacional).
- **Admin (se admin):** gestão de usuários, status/disparo de scan, info do servidor.

---

## 5. Player (detalhes técnicos)

- Elemento **`<audio>`** único controlado pelo `AudioController`.
- `src` = `/api/v1/stream/{songId}?token=<mediaToken>&maxBitRate=&format=`.
  > **Auth de mídia:** o `<audio>` não envia header `Authorization`. Usar o **token de mídia
  > assinado** (`POST /api/v1/auth/media-token`) na query string; idem para `<img src>` de capas
  > (`/cover/{id}?token=`). Cachear o token até `expiresAt` e renovar antes de expirar.
- **Seek:**
  - *Direct play* (`format=raw`): atribuir `currentTime`; o servidor responde a `Range` (HTTP 206).
  - *Transcodificado*: stream chunked não é seekável — ao buscar, **trocar o `src`** para
    `?timeOffset=<segundos>` e dar play (servidor reinicia o ffmpeg no offset).
- **Pré-carregamento** da próxima faixa para transição suave.
- **`navigator.mediaSession`:** metadata (título/artista/capa) + action handlers
  (play, pause, previoustrack, nexttrack, seekto).
- **Scrobble:** chamar `/api/v1/scrobble` ao começar e ao passar ~50% / ao concluir.
- (Opcional) **Web Audio API:** ganho/normalização e visualizador (analyser node).

---

## 6. Autenticação e Sessão
- JWT: `accessToken` em memória; `refreshToken` em **cookie httpOnly `Secure SameSite=Strict`**
  (viável por ser **origem única** — o WebApp é servido pelo binário do servidor, sem CORS).
  Refresh automático via interceptor no cliente fetch (retry transparente em 401).
- Para mídia (`<audio>`/`<img>`), obter token assinado em `/auth/media-token` (ver §5).
- Guarda de rotas: redireciona para login se sem sessão.
- Logout limpa tokens (e cookie via endpoint) e estado do player.

---

## 7. PWA / Offline (Fase avançada)
- Service worker (Workbox): cache de app shell e de capas.
- "Instalar app" (manifest).
- Cache de mídia offline é complexo no navegador — tratar como avançado/opcional.

---

## 8. Roadmap WebApp

**Cliente MVP (secundário — após o iOS; depende de servidor Fase 1)**
- [ ] Setup Vite + TS + Tailwind + Router; tipos gerados do OpenAPI.
- [ ] Login + guarda de rotas + refresh de token.
- [ ] Grade de álbuns + detalhe de álbum.
- [ ] PlayerBar funcional: tocar faixa, seek, play/pause, próxima/anterior, fila básica.
- [ ] Capas com cache.

**Fase 2 — Recursos**
- [ ] Artistas, gêneros, busca unificada.
- [ ] Playlists CRUD + drag-and-drop.
- [ ] Favoritos/rating, shuffle/repeat, fila reordenável.
- [ ] Media Session API.
- [ ] Tema claro/escuro, i18n PT/EN.

**Fase 3 — Avançado/Admin**
- [ ] Painel admin (usuários, scan, status).
- [ ] PWA (instalável + cache de assets).
- [ ] Visualizações/EQ (Web Audio) — opcional.

---

## 9. Testes
- **Unit:** lógica do playerStore (fila, shuffle, repeat), hooks de API (mock).
- **Componentes:** React Testing Library (PlayerBar, TrackList).
- **E2E:** Playwright — login → tocar faixa → seek → próxima. (ver skill `webapp-testing`).
- **Contrato:** tipos gerados do OpenAPI garantem alinhamento com o servidor.

---

## 10. Riscos Específicos do WebApp
| Risco | Mitigação |
|---|---|
| Autoplay bloqueado pelos navegadores | Iniciar reprodução só após interação do usuário. |
| Seek/streaming inconsistente entre navegadores | Garantir `Range` no servidor; testar Chrome/Safari/Firefox. |
| Segurança de tokens no browser | Preferir refresh via cookie httpOnly; minimizar exposição em JS. |
| Offline de mídia limitado no browser | Marcar como opcional; focar PWA em app shell + capas. |
