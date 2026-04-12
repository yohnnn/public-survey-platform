FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/analytics-service/go.mod ./services/analytics-service/go.mod
RUN cd services/analytics-service && go mod download

COPY . .
RUN cd services/analytics-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/analytics-service ./cmd/analytics-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/analytics-service /usr/local/bin/analytics-service
EXPOSE 50055
ENTRYPOINT ["/usr/local/bin/analytics-service"]
