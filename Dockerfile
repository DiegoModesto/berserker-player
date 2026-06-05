# syntax=docker/dockerfile:1

# ---- Stage 1: build do WebApp ----
FROM node:20-alpine AS webapp
WORKDIR /webapp
COPY apps/webapp/package.json apps/webapp/package-lock.json* ./
RUN npm ci
COPY apps/webapp/ ./
RUN npm run build

# ---- Stage 2: build do servidor (binário estático, CGO off) ----
FROM golang:1.26-alpine AS server
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /berserker ./cmd/berserker

# ---- Stage 3: imagem final ----
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates && adduser -D -u 1000 berserker
WORKDIR /app
COPY --from=server /berserker /usr/local/bin/berserker
COPY --from=webapp /webapp/dist /app/webapp
COPY openapi.yaml /app/openapi.yaml

ENV BERSERKER_MUSIC_FOLDER=/music \
    BERSERKER_DATA_FOLDER=/data \
    BERSERKER_PORT=4533 \
    BERSERKER_WEBAPP_DIR=/app/webapp

# Cria os pontos de montagem com dono uid 1000 para que os volumes anônimos/bind
# herdem permissão de escrita (o servidor roda como usuário não-root).
RUN mkdir -p /music /data && chown -R berserker:berserker /music /data /app
VOLUME ["/music", "/data"]
EXPOSE 4533
USER berserker
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://localhost:4533/healthz || exit 1
ENTRYPOINT ["berserker"]
