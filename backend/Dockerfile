FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod ./
# Télécharger fiber et lib/pq (versions fixes et valides)
RUN go mod download

# Ajouter canonical/lxd à la vraie version disponible
RUN go get github.com/canonical/lxd/client@latest

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zadia-backend .

# Image finale légère
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/zadia-backend .

EXPOSE 8080

ENTRYPOINT ["./zadia-backend"]
