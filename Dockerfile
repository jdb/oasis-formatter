# Stage 1: Build Vue 3 + Vite SPA
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build static Go binary (Huma v2 + SSE API)
FROM golang:1.22-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /oasis-formatter ./cmd/server

# Stage 3: Minimal production runtime image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /oasis-formatter /app/oasis-formatter
COPY --from=web-builder /app/web/dist /app/web/dist

ENV PORT=8080
ENV STATIC_DIR=/app/web/dist
EXPOSE 8080

USER nobody
ENTRYPOINT ["/app/oasis-formatter"]
