# BerserkerPlayer — App iOS (cliente primário)

App nativo SwiftUI (iOS 17+) que consome a API nativa (`/api/v1`). Reprodução com
`AVQueuePlayer`, áudio em background, Now Playing (tela de bloqueio) e controles remotos.

## Gerar o projeto e compilar

O `.xcodeproj` é gerado por [XcodeGen](https://github.com/yonaskolb/XcodeGen) a partir de
`project.yml` (o projeto não é versionado; `project.yml` é a fonte da verdade).

```bash
brew install xcodegen
cd apps/mobile
xcodegen generate
open BerserkerPlayer.xcodeproj
# ou via CLI (requer plataforma de simulador iOS instalada):
xcodebuild -scheme BerserkerPlayer -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Validação de compilação sem simulador (type-check):
```bash
SDK=$(xcrun --sdk iphonesimulator --show-sdk-path)
xcrun swiftc -sdk "$SDK" -target arm64-apple-ios17.0-simulator -typecheck Sources/**/*.swift
```

## Estrutura
```
Sources/
├── App/                BerserkerApp (@main), RootView, MainTabView
├── Core/
│   ├── Networking/     APIModels (Codable), APIClient (async, refresh, media token)
│   ├── Auth/           KeychainStore, Session (@Observable)
│   └── Playback/       PlaybackEngine (AVQueuePlayer, Now Playing, comandos remotos)
├── Features/           Login, Library, AlbumDetail, Search, NowPlaying, Settings
└── DesignSystem/       Theme, CoverImage
```

## Decisões
- **iOS 17+** para usar a macro `@Observable` (Observation) e SwiftData (fases futuras).
- **Auth de mídia:** `/stream` e `/cover` recebem `?token=` (token de mídia assinado), pois o
  `AVPlayer` não envia header `Authorization`. O `APIClient` cacheia e renova o token.
- **Background audio:** `UIBackgroundModes: [audio]` no Info.plist + `AVAudioSession(.playback)`.
- **Scrobble:** `nowplaying` ao iniciar; `submission` ao passar 50% (dedup no servidor).
- ATS: `NSAllowsArbitraryLoads` habilitado para servidores self-hosted em HTTP (dev).

## Entregue
Login (Keychain) · grade de álbuns com filtros · detalhe do álbum · busca · playlists
(aba + detalhe) · favoritar no Now Playing · player (fila, seek, prev/next, shuffle/repeat) ·
mini-player · Now Playing · áudio em background · **downloads offline (SwiftData)** com
reprodução local, gestão de armazenamento e remoção.

Fases futuras: CarPlay, widgets/Live Activities, i18n.

## Testes e cobertura
```bash
xcodegen generate
xcodebuild -scheme BerserkerPlayer -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -enableCodeCoverage YES test
```
20 testes unitários (XCTest) cobrindo a **camada de lógica** (sem UI):
- `PlaybackQueue` 94% · `DownloadStore` 95% · `DownloadedTrack` 100% · `DownloadManager` 60%
- decodificação de modelos da API

> Nota: a cobertura *do app inteiro* é baixa (~10%) porque a maior parte é **views SwiftUI**,
> que não são exercitadas por testes unitários. Atingir ≥80% global no iOS exigiria testes de
> UI (XCUITest) com um backend ativo — planejado como trabalho futuro. A lógica testável
> (offline, fila, persistência, rede) está coberta em ~90%+.
