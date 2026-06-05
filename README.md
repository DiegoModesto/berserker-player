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

## Instalação local

### Pré‑requisitos

| Ferramenta | Para quê | Versão |
|---|---|---|
| **Go** | servidor | 1.22+ |
| **ffmpeg** / **ffprobe** | scan (duração/bitrate) e transcodificação | qualquer recente |
| **Node** + **npm** | webapp | Node 20+ |
| **Xcode** | app iOS | 15+ |
| **XcodeGen** | gerar o projeto iOS a partir de `project.yml` | 2.4+ |

No macOS, via [Homebrew](https://brew.sh):

```bash
brew install go ffmpeg node xcodegen
# Xcode pela App Store; depois aceite a licença e instale a plataforma de simulador iOS:
sudo xcodebuild -license accept
xcodebuild -downloadPlatform iOS
```

Clone o repositório:

```bash
git clone git@github.com:DiegoModesto/berserker-player.git
cd berserker-player
```

### Servidor (Go + SQLite + ffmpeg)

```bash
cd server
go mod download
go run ./cmd/berserker --music /caminho/da/sua/musica --data ./data --admin-password trocar123
# Servindo em http://localhost:4533  ·  healthcheck: GET /healthz
```

Na 1ª execução cria o usuário `admin` (use `--admin-password`; se omitir, uma senha é gerada e
**logada uma única vez**). Outras opções: `--config berserker.example.toml`, `--port`, `BERSERKER_WATCH=true`
(rescan em tempo real). Para servir o WebApp na mesma origem: `--webapp-dir ../apps/webapp/dist`.

### WebApp (React + Vite)

```bash
cd apps/webapp
npm install
npm run dev        # http://localhost:5173  (faz proxy de /api → http://localhost:4533)
npm run build      # build de produção em dist/ (gera também o service worker do PWA)
```

### App iOS (SwiftUI)

```bash
cd apps/mobile
xcodegen generate            # gera BerserkerPlayer.xcodeproj a partir de project.yml
open BerserkerPlayer.xcodeproj
# Build/teste por linha de comando (escolha um simulador instalado):
xcodebuild -scheme BerserkerPlayer -destination 'platform=iOS Simulator,name=iPhone 15' build
```

No app, informe a URL do servidor (ex.: `http://SEU_IP:4533`), usuário e senha.

## Testes

```bash
# Servidor (Go) — inclui integração com ffmpeg e detector de corrida:
cd server && go test -race ./...
# Cobertura (cruzando pacotes):
go test -coverpkg=./... -coverprofile=cover.out ./... && go tool cover -func=cover.out | tail -1

# WebApp (Vitest):
cd apps/webapp && npm test
npm run test:cov          # com relatório de cobertura

# iOS (XCTest no simulador):
cd apps/mobile && xcodegen generate
xcodebuild -scheme BerserkerPlayer \
  -destination 'platform=iOS Simulator,name=iPhone 15' \
  -enableCodeCoverage YES test
```

Cobertura atual: **servidor ~80%**, **webapp ~96%** (linhas). No iOS, a camada de lógica
(fila, downloads offline, persistência) é coberta ~90%+; as views SwiftUI não são cobertas
por testes unitários (ver [`apps/mobile/README.md`](apps/mobile/README.md)).

> Os testes do servidor e do iOS geram fixtures de áudio com `ffmpeg`; se o `ffmpeg` não estiver
> instalado, esses testes são **pulados** automaticamente em vez de falhar.

Cada componente tem instruções detalhadas no seu próprio README:
[server](server/README.md) · [webapp](apps/webapp/README.md) · [mobile](apps/mobile/README.md).

## Branches por fase

O desenvolvimento avança na `main`; cada fase concluída recebe uma branch-snapshot
`phase/<n>-<nome>` (ex.: `phase/server-0-foundations`).
