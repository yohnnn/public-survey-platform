FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/api-service/go.mod ./services/api-service/go.mod
RUN cd services/api-service && go mod download

COPY . .
RUN cd services/api-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api-service ./cmd/api-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/api-service /usr/local/bin/api-service
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api-service"]
