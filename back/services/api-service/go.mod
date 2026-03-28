module github.com/yohnnn/public-survey-platform/back/services/api-service

go 1.25.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0
	github.com/yohnnn/public-survey-platform/back v0.0.0
	google.golang.org/grpc v1.79.3
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/yohnnn/public-survey-platform/back => ../..
