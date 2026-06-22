FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Télécharger les dépendances (layer caché si go.mod ne change pas)
COPY go.mod ./
RUN go mod download
RUN go get github.com/canonical/lxd/client@latest
RUN go get github.com/gorilla/websocket@latest
RUN go get github.com/gofiber/contrib/websocket@latest

COPY . .
RUN go mod tidy

# Cache BuildKit pour le build Go : très rapide après le premier build
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o zadia-backend .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/zadia-backend .

EXPOSE 8080

ENTRYPOINT ["./zadia-backend"]
