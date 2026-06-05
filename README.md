# BerserkerPlayer

> Player de música self-hosted inspirado no [Navidrome](https://github.com/navidrome/navidrome).
> Indexa uma biblioteca de música local, serve streaming sob demanda e oferece clientes nativos
> (iOS) e web.

Monorepo com três componentes sobre o mesmo contrato de API ([`openapi.yaml`](openapi.yaml)).

## Estrutura

```
berserker-player/
├── server/         # Servidor (Go + SQLite + ffmpeg) — core, API, scanner, streaming
├── apps/
│   ├── webapp/     # Cliente web (React + TypeScript + Vite)
│   └── mobile/     # App iOS (Swift + SwiftUI + AVFoundation)
├── Plans/          # Planos de arquitetura e roadmap
└── openapi.yaml    # Contrato da API nativa (fonte da verdade)
```

## Componentes

| Componente | Stack | Plano |
|---|---|---|
| [server](server/) | Go + SQLite | [`Plans/01-server-plan.md`](Plans/01-server-plan.md) |
| [apps/mobile](apps/mobile/) | Swift + SwiftUI | [`Plans/02-ios-app-plan.md`](Plans/02-ios-app-plan.md) |
| [apps/webapp](apps/webapp/) | React + TypeScript | [`Plans/03-webapp-plan.md`](Plans/03-webapp-plan.md) |

Visão geral em [`Plans/00-initial-plan.md`](Plans/00-initial-plan.md).

## Quickstart (Docker)

```bash
# Coloque sua música em ./music e suba o stack:
docker compose up --build
# Acesse http://localhost:4533  (admin / valor de BERSERKER_ADMIN_PASSWORD)
```

A imagem é multi-stage: builda o WebApp, compila o servidor (binário estático, sem CGO) e
empacota com `ffmpeg`. O WebApp é servido pelo próprio binário (origem única). `BERSERKER_WATCH=true`
habilita rescan incremental em tempo real (fsnotify).

## Rodar em desenvolvimento

```bash
# Servidor (Go + ffmpeg):
cd server && go run ./cmd/berserker --music /sua/musica --data ./data --admin-password trocar

# WebApp (proxy para :4533):
cd apps/webapp && npm install && npm run dev      # http://localhost:5173

# iOS (Xcode + XcodeGen):
cd apps/mobile && xcodegen generate && open BerserkerPlayer.xcodeproj
```

## Requisitos de desenvolvimento

- **Go** 1.22+ e **ffmpeg** (servidor)
- **Node** 20+ (webapp)
- **Xcode** 15+ e **XcodeGen** (mobile)

## Branches por fase

O desenvolvimento avança na `main`; cada fase concluída recebe uma branch-snapshot
`phase/<n>-<nome>` (ex.: `phase/server-0-foundations`).
