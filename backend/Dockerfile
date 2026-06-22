FROM golang:1.22-alpine AS builder

WORKDIR /app

# git requis par go mod download pour certains modules VCS
RUN apk add --no-cache git

COPY go.mod ./
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
