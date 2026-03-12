FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOGC=50 CGO_ENABLED=0 GOOS=linux go build \
    -p 1 \
    -ldflags="-s -w" \
    -o bin/api \
    ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api .

EXPOSE 8080

ENTRYPOINT ["./api"]