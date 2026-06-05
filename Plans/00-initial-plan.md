# BerserkerPlayer — Plano Inicial (Visão Geral)

> Player de música self-hosted inspirado no [Navidrome](https://github.com/navidrome/navidrome).
> Objetivo: indexar uma biblioteca de música local, servir streaming sob demanda e oferecer
> clientes nativos (iOS) e web para reprodução, navegação e gerenciamento.

---

## 1. Visão e Princípios

**Visão:** um servidor de música pessoal que roda em hardware próprio (NAS, VPS, Raspberry Pi),
escaneia arquivos de áudio do disco, expõe uma API e permite tocar a coleção de qualquer lugar,
por meio de um app iOS nativo e de um WebApp.

**Princípios norteadores (herdados do Navidrome):**
- **Leve e self-hosted:** binário único, baixo consumo de RAM/CPU, sem dependências pesadas.
- **Compatibilidade com Subsonic API:** falar o "dialeto" Subsonic/OpenSubsonic garante que
  dezenas de clientes existentes (DSub, play:Sub, Symfonium, Feishin etc.) já funcionem.
- **Biblioteca como fonte da verdade:** o sistema de arquivos é a fonte; o banco é um índice
  reconstruível a partir do scan.
- **Streaming com transcodificação sob demanda:** servir o formato adequado à banda/cliente.
- **Multiusuário:** cada usuário tem favoritos, playlists, histórico e preferências próprias.
- **Privacidade primeiro:** dados ficam no servidor do usuário; nada de telemetria obrigatória.

---

## 2. Componentes do Sistema

| Componente | Stack proposta | Plano |
|---|---|---|
| **Servidor** (core, API, scanner, streaming, DB) | Go + SQLite | [`01-server-plan.md`](01-server-plan.md) |
| **App iOS** (cliente nativo) | Swift + SwiftUI + AVFoundation | [`02-ios-app-plan.md`](02-ios-app-plan.md) |
| **WebApp** (cliente no navegador) | React + TypeScript + Web Audio | [`03-webapp-plan.md`](03-webapp-plan.md) |

```
                         ┌──────────────────────────┐
                         │     Biblioteca de Música  │
                         │   (arquivos no disco)     │
                         └────────────┬─────────────┘
                                      │ scan
                         ┌────────────▼─────────────┐
                         │      SERVIDOR (Go)        │
                         │  ┌─────────────────────┐  │
                         │  │ Scanner / Watcher   │  │
                         │  │ Metadata extractor  │  │
                         │  │ SQLite (índice)     │  │
                         │  │ Transcoder (ffmpeg) │  │
                         │  └─────────────────────┘  │
                         │   REST + Subsonic API     │
                         └──────┬───────────┬────────┘
                                │           │
                  ┌─────────────▼──┐    ┌───▼─────────────┐
                  │   App iOS       │    │   WebApp        │
                  │ (SwiftUI)       │    │ (React/TS)      │
                  └─────────────────┘    └─────────────────┘
```

---

## 3. Decisões de Arquitetura

### 3.1 Contrato de API
Adotaremos **duas camadas de API** sobre o mesmo core:
1. **API nativa do BerserkerPlayer** (REST/JSON moderna, versionada em `/api/v1`) — usada pelo
   WebApp e pelo app iOS oficial. Mais limpa, paginação por cursor, JSON consistente.
2. **API compatível Subsonic / OpenSubsonic** (`/rest/*`) — para reaproveitar o ecossistema de
   clientes terceiros e validar o servidor contra clientes maduros.

> Decisão de design: o app iOS e o WebApp oficiais consomem a API nativa, mas o servidor mantém
> a camada Subsonic como "modo de compatibilidade". Isso evita amarrar a UX moderna às limitações
> históricas do Subsonic, sem perder o ecossistema.

### 3.2 Autenticação
- Login por usuário/senha → emissão de **JWT** (access curto + refresh longo, rotativo).
- **Entrega do refresh:** cookie httpOnly para o WebApp (origem única); corpo + Keychain para o iOS.
- **Mídia (`/stream`, `/cover`):** `<audio>` e `AVPlayer` não enviam header Bearer; por isso esses
  endpoints autenticam por **token de mídia assinado e de curta duração** via `?token=`
  (emitido por `/api/v1/auth/media-token`).
- Senhas armazenadas com **argon2id** (nunca texto puro).
- **Subsonic adiado p/ Fase 3:** exige senha recuperável (`md5(password+salt)`), incompatível com
  argon2id-only; quando ativado, usar cofre criptografado reversível opt-in.

### 3.3 Modelo de Dados (entidades centrais)
`Artist`, `Album`, `Song/MediaFile`, `Genre`, `Playlist`, `User`, `Annotation`
(rating/starred/playcount por usuário), `Library/Folder`, `Share`.

### 3.4 Streaming
- **Direct play:** servir o arquivo original com `Range`/206 (seek e buffering parcial).
- **Transcodificar sob demanda** (ffmpeg) com perfis por qualidade (ex.: opus 128k para 4G). A saída
  é chunked e **não-seekável por Range**; seek em transcode = reabrir a stream com `?timeOffset=`.
- Cache de transcodificação opcional (coalescer requisições idênticas).

### 3.5 Origem única (servir o WebApp pelo servidor)
O binário do servidor entrega o build estático do WebApp na **mesma origem** da API — elimina CORS,
viabiliza o cookie httpOnly de refresh e dispensa configurar BaseURL no WebApp. O contrato vive em
[`../openapi.yaml`](../openapi.yaml) (OpenAPI 3.1), fonte única para tipos do WebApp e referência iOS.

---

## 4. Roadmap em Fases

### Fase 0 — Fundações (servidor mínimo)
- [ ] Estrutura do projeto Go, config, logging.
- [ ] Schema SQLite + migrações.
- [ ] Scanner que lê metadados (tags ID3/Vorbis) e popula o índice.
- [ ] Endpoint de streaming com suporte a `Range`.
- [ ] Auth básica (JWT) + 1 usuário.

### Fase 1 — API nativa MVP
- [ ] API nativa: artistas, álbuns, faixas, busca, playlists + `/auth/media-token`.
- [ ] Capas de álbum (artwork) + placeholder.
- [ ] OpenAPI publicado e validado por testes de contrato.

### Fase 2 — App iOS MVP (cliente primário)
- [ ] Cliente iOS: login, navegação, player com AVFoundation (URL de stream com `?token=`).
- [ ] Controles na tela de bloqueio / Now Playing (MPNowPlayingInfoCenter).
- [ ] Reprodução em background.

### Fase 2b — WebApp MVP (cliente secundário)
- [ ] WebApp servido pelo binário (origem única): login, navegar biblioteca, player com fila.

### Fase 3 — Recursos avançados
- [ ] Transcodificação sob demanda + perfis.
- [ ] Multiusuário, favoritos, playcount, scrobbling (Last.fm / ListenBrainz).
- [ ] Compatibilidade Subsonic/OpenSubsonic.
- [ ] Download offline (iOS), smart playlists, rádio/streams.

### Fase 4 — Produção e qualidade
- [ ] Watcher de filesystem (rescan incremental).
- [ ] Empacotamento Docker, releases multiplataforma.
- [ ] Testes E2E, CI/CD, documentação.

---

## 5. Critérios de Sucesso do MVP
1. Servidor escaneia uma pasta de música e indexa corretamente artistas/álbuns/faixas.
2. WebApp permite logar, navegar e tocar uma música do começo ao fim com seek.
3. App iOS toca música em background com controles na tela de bloqueio.
4. Tudo roda em um único host (ex.: Docker compose) com setup documentado.

---

## 6. Riscos e Mitigações
| Risco | Mitigação |
|---|---|
| Variedade de formatos/tags inconsistentes | Usar libs maduras de parsing; normalizar no scan; logar arquivos problemáticos. |
| Performance de scan em bibliotecas grandes (100k+ faixas) | Scan incremental por mtime; processamento concorrente; índices no SQLite. |
| Latência/seek em streaming via celular | Suporte robusto a `Range` + transcodificação adaptativa. |
| Manter 2 clientes + servidor sincronizados com a API | Contrato versionado + testes de contrato; [`openapi.yaml`](../openapi.yaml) como fonte. |
| Background audio / políticas da App Store | Seguir guidelines de áudio em background do iOS desde cedo. |
| Auth de mídia em `<audio>`/`AVPlayer` (sem header Bearer) | Token de mídia assinado de curta duração via `?token=` (`/auth/media-token`). |
| Seek em conteúdo transcodificado (stream não-seekável) | Direct play usa Range/206; transcode usa reabertura com `?timeOffset=`. |

---

## 7. Próximos Passos
1. ✅ Contrato da API nativa definido em [`../openapi.yaml`](../openapi.yaml) (fonte da verdade).
2. Implementar o servidor Fase 0 (ver [`01-server-plan.md`](01-server-plan.md)) — exige instalar
   Go + ffmpeg no ambiente (ainda ausentes).
3. Iniciar o **app iOS** (cliente primário) assim que o servidor expuser a Fase 1.
4. WebApp em seguida, servido pelo próprio binário (origem única).
