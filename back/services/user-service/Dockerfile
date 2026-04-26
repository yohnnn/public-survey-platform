FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/
RUN cd services/user-service && go mod download

COPY . .
RUN cd services/user-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/user-service ./cmd/user-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/user-service /usr/local/bin/user-service
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/user-service"]
