FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/wpp-gateway ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/wpp-gateway /bin/wpp-gateway
COPY migrations/ /app/migrations/
COPY web/ /app/web/

EXPOSE 8080

ENTRYPOINT ["/bin/wpp-gateway"]
