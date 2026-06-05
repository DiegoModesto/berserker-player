# BerserkerPlayer — Plano do App iOS

> Cliente nativo iOS para o servidor BerserkerPlayer. Consome a API nativa (`/api/v1`).
> Foco em reprodução fluida, áudio em background, controles de sistema (tela de bloqueio /
> CarPlay) e download offline.
>
> **Prioridade:** este é o **cliente primário do MVP** (decisão do projeto). O WebApp segue como
> cliente secundário. O contrato é a fonte da verdade em [`../openapi.yaml`](../openapi.yaml).

Referência cruzada: [`00-initial-plan.md`](00-initial-plan.md) · [`01-server-plan.md`](01-server-plan.md)

---

## 1. Stack e Justificativa

| Item | Escolha | Por quê |
|---|---|---|
| Linguagem | **Swift 5.9+** | Padrão moderno iOS. |
| UI | **SwiftUI** (UIKit pontual) | Produtividade, declarativo; UIKit onde SwiftUI ainda limita. |
| Áudio | **AVFoundation** (`AVPlayer`/`AVQueuePlayer`) | Streaming HTTP, seek, gapless básico, background. |
| Now Playing | `MPNowPlayingInfoCenter` + `MPRemoteCommandCenter` | Tela de bloqueio, fones, CarPlay. |
| Rede | `URLSession` + `async/await` | Sem dependências; concorrência estruturada. |
| Persistência | **SwiftData** | Cache de metadados, fila, downloads offline. |
| Imagens | `Kingfisher` ou `AsyncImage` + cache | Capas com cache em disco. |
| Arquitetura | **MVVM** + camada de serviços | Testável, separação clara. |
| Min. target | **iOS 17+** | Resolve a contradição "iOS 16 + SwiftData": SwiftData exige iOS 17. Adotar iOS 17+ p/ usar SwiftData direto (sem fallback Core Data). |

---

## 2. Arquitetura

```
App
├── Core
│   ├── Networking/        # APIClient (async), endpoints, modelos Codable, auth/refresh
│   ├── Auth/              # KeychainStore (tokens), sessão
│   ├── Persistence/       # SwiftData: cache, fila, downloads
│   └── Playback/          # PlaybackEngine (AVQueuePlayer), NowPlaying, RemoteCommands
├── Features
│   ├── Login/
│   ├── Library/           # Artists, Albums, Songs, Genres
│   ├── AlbumDetail/
│   ├── Search/
│   ├── Playlists/
│   ├── NowPlaying/        # player full-screen + mini-player
│   └── Settings/
└── DesignSystem/          # cores, tipografia, componentes reutilizáveis
```

**Camadas:** View (SwiftUI) → ViewModel (`@Observable`/`ObservableObject`) → Service → APIClient.
`PlaybackEngine` é um singleton observável compartilhado pela UI (mini-player + tela cheia).

---

## 3. Funcionalidades por Tela

- **Onboarding/Login:** URL do servidor + usuário/senha → login → guarda tokens no **Keychain**.
  Suporte a múltiplos servidores (opcional).
- **Biblioteca:** abas Artistas / Álbuns / Playlists; listas com capas, busca, ordenação,
  filtros (recentes, mais tocados, aleatórios).
- **Detalhe do Álbum/Artista:** faixas, durações, ações (tocar, adicionar à fila, favoritar).
- **Busca:** unificada (artistas + álbuns + faixas), debounce, resultados ao vivo.
- **Now Playing:** capa grande, scrubber (seek), play/pause, prev/next, shuffle, repeat,
  fila editável (arrastar p/ reordenar), favoritar, AirPlay.
- **Mini-player:** persistente acima da tab bar; expande para a tela cheia.
- **Configurações:** qualidade de streaming (transcode), Wi-Fi-only para download, conta, logout.

---

## 4. Engine de Reprodução

- **`AVQueuePlayer`** para fila com pré-carregamento da próxima faixa.
- URL de stream: `/api/v1/stream/{songId}?token=<mediaToken>&maxBitRate=&format=`.
  > **Auth de mídia:** o `AVPlayer` não envia `Authorization: Bearer` de forma confiável. Usar o
  > **token de mídia assinado** (`POST /api/v1/auth/media-token`) na query string. Obter/renovar o
  > token antes de montar a `AVURLAsset`; cachear até `expiresAt` e renovar proativamente.
- **Seek:**
  - *Direct play* (`format=raw`): seek nativo via `Range` (servidor responde 206).
  - *Transcodificado*: a saída é chunked e não-seekável; ao buscar, **recriar** o item com
    `?timeOffset=<segundos>` e dar `replaceCurrentItem` (o servidor reinicia o ffmpeg no offset).
- **Background audio:** ativar `Background Modes → Audio` no entitlements; configurar
  `AVAudioSession` (`.playback`).
- **Controles remotos:** `MPRemoteCommandCenter` (play/pause/next/prev/seek) e
  `MPNowPlayingInfoCenter` (título, artista, capa, duração, posição) → tela de bloqueio + CarPlay.
- **Interrupções:** tratar chamadas, fones desconectados (`AVAudioSession.interruptionNotification`,
  `routeChangeNotification`).
- **Scrobble:** `event=nowplaying` ao iniciar e `event=submission` ao passar ~50%/concluir
  (`POST /api/v1/scrobble`). O servidor deduplica submissões — o cliente não precisa se preocupar
  com recargas/duplicatas.

---

## 5. Offline / Downloads (Fase avançada)

- Baixar faixas/álbuns/playlists selecionados para o dispositivo (arquivo + metadados no SwiftData).
- Indicadores de "baixado" na UI; gestão de armazenamento (limpar, ver uso).
- Reprodução offline: o `PlaybackEngine` resolve URL local quando disponível.
- Opção "somente Wi-Fi" para downloads.

---

## 6. Segurança
- Tokens no **Keychain** (nunca em UserDefaults).
- Refresh automático de access token (interceptor no APIClient; retry transparente em 401).
- Suporte a HTTPS/self-signed (alerta claro ao usuário se aceitar cert custom).

---

## 7. Roadmap iOS

**Cliente MVP (primário, depende de servidor Fase 1)**
- [ ] Projeto Xcode, DesignSystem base, APIClient + auth/Keychain.
- [ ] Login + listagem de álbuns/artistas + detalhe.
- [ ] PlaybackEngine (AVQueuePlayer) + mini-player + Now Playing.
- [ ] Background audio + controles na tela de bloqueio.
- [ ] Busca.

**Fase 3 — Recursos**
- [ ] Playlists (criar/editar/reordenar).
- [ ] Favoritos, rating, shuffle/repeat, fila editável.
- [ ] Configurações de qualidade/transcode.
- [ ] Scrobble.

**Fase 4 — Polimento**
- [ ] Download offline + gestão de armazenamento.
- [ ] CarPlay.
- [ ] Widgets / Live Activities (Now Playing).
- [ ] Acessibilidade (VoiceOver, Dynamic Type), localização PT/EN.

---

## 8. Testes e Distribuição
- **Unit:** ViewModels, mapeamento de modelos, lógica de fila.
- **UI tests:** fluxo de login → tocar faixa.
- **Manual:** background, interrupções, AirPlay, CarPlay.
- **Distribuição:** TestFlight para beta; depois App Store (atentar a guidelines de áudio em
  background e à exigência de funcionalidade real do app cliente).

---

## 9. Riscos Específicos do iOS
| Risco | Mitigação |
|---|---|
| Gapless/crossfade limitado no AVQueuePlayer | Aceitar gapless básico no MVP; avaliar engine custom (AVAudioEngine) depois. |
| Políticas da App Store p/ apps "cliente de servidor" | Garantir UX completa e não depender só de config manual de URL. |
| Servidores self-signed / HTTP | UX clara de confiança de certificado; recomendar HTTPS. |
| Consumo de bateria em background | Otimizar pré-carregamento e rede; respeitar `maxBitRate`. |
