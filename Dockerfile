# syntax=docker/dockerfile:1.6
#
# Build de producción: un solo contenedor sirve /api/v1, /webhook, y /admin
# (SPA React embebido en el binario Go vía //go:embed con build tag `prod`).
# Para dev local seguir usando Vite (admin/Dockerfile separado o `npm run dev`).

# ─── Stage 1: build del SPA admin ─────────────────────────────────────────────
FROM node:22-alpine AS admin-builder
WORKDIR /admin
# Cache de deps separado del source para builds más rápidos.
COPY admin/package.json admin/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY admin/ ./
RUN npm run build
# Output: /admin/dist (estáticos del SPA).

# ─── Stage 2: build del binario Go con SPA embebido ───────────────────────────
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copiamos el dist del SPA para que el //go:embed lo recoja.
COPY --from=admin-builder /admin/dist ./admin/dist
# `-tags prod` activa api/admin_embed_prod.go (registra /admin); sin el tag,
# se usa api/admin_embed_dev.go (no-op) — preserva el flujo dev local.
RUN CGO_ENABLED=0 GOOS=linux go build -tags prod -ldflags='-s -w' -o /fidel-quick .

# ─── Stage 3: runtime mínimo ──────────────────────────────────────────────────
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=go-builder /fidel-quick /fidel-quick
EXPOSE 8080
CMD ["/fidel-quick"]
