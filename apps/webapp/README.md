# BerserkerPlayer — WebApp (cliente secundário)

SPA em React + TypeScript + Vite. Consome a API nativa (`/api/v1`). Servido em produção
pelo próprio binário do servidor (origem única → sem CORS, refresh por cookie httpOnly).

## Stack
- React 18 + TypeScript + Vite
- React Router · TanStack Query (estado do servidor) · Zustand (estado do player)
- Tailwind CSS
- Áudio: elemento `<audio>` + Media Session API

## Desenvolvimento
```bash
npm install
npm run dev        # http://localhost:5173 (proxy /api → http://localhost:4533)
npm run build      # type-check (tsc) + build de produção em dist/ (+ PWA: sw.js, manifest)
npm run test       # Vitest
npm run test:cov   # Vitest com cobertura (v8)
npm run gen:api    # (opcional) gera tipos a partir de ../../openapi.yaml
```

Cobertura de testes (Vitest): **Statements/Lines ~96%**, Branches ~93%, Functions ~80%.

Servir o build pelo servidor (origem única):
```bash
# na pasta server:
go run ./cmd/berserker --webapp-dir ../apps/webapp/dist --music /sua/musica
```

## Estrutura
```
src/
├── api/         client (fetch + refresh + media token), types
├── auth/        SessionProvider (contexto de sessão)
├── store/       player (Zustand)
├── player/      AudioController (<audio> + Media Session), helpers
├── components/  Layout, PlayerBar, Cover
└── features/    login, library, album, search
```

## Decisões
- **accessToken em memória**; **refreshToken em cookie httpOnly** (definido pelo servidor
  quando o header `X-Client: webapp` está presente). Sem token em localStorage.
- **Mídia:** `<audio>`/`<img>` usam `?token=` (token de mídia assinado), cacheado no client.
- **Seek:** atribui `currentTime` (direct play → Range/206 no servidor).

## Entregue
Login · grade de álbuns com filtros · detalhe do álbum (tocar/fila) · playlists
(lista + detalhe + **drag-and-drop**) · **favoritar** + **rating (estrelas)** · busca (debounce) ·
PlayerBar (play/pause, prev/next, seek, shuffle/repeat) · Media Session ·
**tema claro/escuro** · **PWA** (instalável, service worker, cache de app shell).

Fases futuras: i18n PT/EN, painel admin, visualizações/EQ (Web Audio).
