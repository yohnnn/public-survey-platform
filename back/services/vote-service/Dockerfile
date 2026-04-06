FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/vote-service/go.mod ./services/vote-service/go.mod
RUN cd services/vote-service && go mod download

COPY . .
RUN cd services/vote-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/vote-service ./cmd/vote-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/vote-service /usr/local/bin/vote-service
EXPOSE 50053
ENTRYPOINT ["/usr/local/bin/vote-service"]
