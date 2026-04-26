module github.com/yohnnn/public-survey-platform/back/services/user-service

go 1.25.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/jackc/pgx/v5 v5.9.0
	github.com/yohnnn/public-survey-platform/back v0.0.0
	go.uber.org/mock v0.6.0
	golang.org/x/crypto v0.49.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

replace github.com/yohnnn/public-survey-platform/back => ../..

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
)
