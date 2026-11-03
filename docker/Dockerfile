# auth-service image: make build-image (build context — repo ildizi).
# 1) web: React SPA build (web/dist), Go binary ichiga embed qilinadi
# 2) build: rasmiy golang image'da statik binary (CGO o'chiq)
# 3) runtime: distroless static, nonroot user — faqat binary va runtime fayllar

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/*.go web/
COPY --from=web /src/web/dist web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/auth-service ./auth-service
COPY migrations/ migrations/
COPY docs/openapi.json docs/openapi.json
COPY config/config.yaml config/config.yaml
# Swagger UI image'ga kirmaydi (prod da taqiqlangan). JWT kaliti image'ga kirmaydi:
# JWT_PRIVATE_KEY_PATH bo'yicha volume/secret sifatida ulanadi (standart certs/private.pem).
ENV SWAGGER_UI_ENABLED=false
EXPOSE 5000
USER nonroot:nonroot
ENTRYPOINT ["/app/auth-service"]
