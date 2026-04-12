FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/realtime-service/go.mod ./services/realtime-service/go.mod
RUN cd services/realtime-service && go mod download

COPY . .
RUN cd services/realtime-service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/realtime-service ./cmd/realtime-service

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/realtime-service /usr/local/bin/realtime-service
EXPOSE 50056
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/realtime-service"]
