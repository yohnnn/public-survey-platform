FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/feed-service/go.mod services/feed-service/go.sum ./services/feed-service/
RUN cd services/feed-service && go mod download

COPY . .
RUN cd services/feed-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/feed-service ./cmd/feed-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/feed-service /usr/local/bin/feed-service
EXPOSE 50054
ENTRYPOINT ["/usr/local/bin/feed-service"]
