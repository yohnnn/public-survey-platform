FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/
RUN cd services/auth-service && go mod download

COPY . .
RUN cd services/auth-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth-service ./cmd/auth-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/auth-service /usr/local/bin/auth-service
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/auth-service"]
