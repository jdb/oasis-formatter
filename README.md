# Oasis Formatter

**Oasis Formatter** is a bilingual book/document formatter designed specifically for the **7" Kindle Oasis (9th/10th Gen, 300 PPI)**.

Drop or paste any text, choose your destination language, and watch in real time as:
1. The **source language** is automatically detected.
2. The text is split into logical paragraphs and **translated paragraph-by-paragraph over Server-Sent Events (SSE)**.
3. Each `(source, translation)` paragraph pair is dynamically laid out onto **exact Kindle Oasis-sized pages (`107.02 mm × 142.24 mm` / `303.36 pt × 403.20 pt`)** where the first paragraph is the original text and the second paragraph is its translation.
4. Export a **1:1 vector PDF** with self-hosted embedded e-ink fonts (`Literata`) ready to transfer to your Kindle Oasis with zero pinch-zooming or wasted margin space.

## Architecture (Mirroring `mespousses.fr`)

- **Backend:** Go + [Huma v2](https://huma.rocks/) (`OpenAPI 3.1` at `/openapi.json`, `/api/health`, RFC 9457 `application/problem+json`, `X-Request-Id` middleware, and SSE streaming at `POST /api/translate/stream`).
- **Frontend:** Vue 3 + Vite + TypeScript SPA with self-hosted `Literata` e-ink fonts (`web/public/fonts/`), incremental canvas-measured Kindle Oasis paginator (`303.36 pt × 403.20 pt`), and `pdf-lib` vector PDF exporter.
- **Deployment:** Single multi-stage Docker container deployed on **Fly.io (`fra` / Frankfurt)** behind **Cloudflare**.

## Local Development

```bash
# Build frontend
cd web && npm install && npm run build && cd ..

# Run Go server (serves both API and compiled SPA on :8080)
go run ./cmd/server
```
