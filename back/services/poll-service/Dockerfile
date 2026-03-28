FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/poll-service/go.mod services/poll-service/go.sum ./services/poll-service/
RUN cd services/poll-service && go mod download

COPY . .
RUN cd services/poll-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/poll-service ./cmd/poll-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/poll-service /usr/local/bin/poll-service
EXPOSE 50052
ENTRYPOINT ["/usr/local/bin/poll-service"]
