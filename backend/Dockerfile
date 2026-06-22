FROM golang:1.22-alpine AS builder

WORKDIR /app

# git requis pour que go mod puisse cloner les dépôts VCS
RUN apk add --no-cache git

# Copier les sources en premier pour que go mod tidy puisse résoudre
# les imports et trouver les vraies versions publiées
COPY . .
RUN go get github.com/gofiber/fiber/v2@v2.52.5
RUN go get github.com/lib/pq@v1.10.9
RUN go get github.com/canonical/lxd/client@latest
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zadia-backend .

# Image finale légère
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/zadia-backend .

EXPOSE 8080

ENTRYPOINT ["./zadia-backend"]
