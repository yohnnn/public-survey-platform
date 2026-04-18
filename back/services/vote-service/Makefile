.PHONY: help run build test vet tidy clean

SERVICE_NAME := vote-service
CMD_PATH := ./cmd/$(SERVICE_NAME)
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(SERVICE_NAME)

help:
	@echo "Targets:"
	@echo "  run    - Run service"
	@echo "  build  - Build binary into ./bin"
	@echo "  test   - Run all tests"
	@echo "  vet    - Run go vet"
	@echo "  tidy   - Run go mod tidy"
	@echo "  clean  - Remove build artifacts"

run:
	go run $(CMD_PATH)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) $(CMD_PATH)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
