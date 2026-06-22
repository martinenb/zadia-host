FROM golang:1.22-alpine AS builder

WORKDIR /app

# Installer les dépendances système pour cgo (sqlite3, lib/pq)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zadia-backend .

# Image finale légère
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/zadia-backend .

EXPOSE 8080

ENTRYPOINT ["./zadia-backend"]
