.PHONY: help \
	proto-lint proto-breaking proto-deps-update proto-generate \
	test-back test-services test-unit test-e2e test-all \
	vet-services tidy-services \
	compose-up compose-down compose-logs compose-ps compose-restart

BUF ?= /home/yohn/go/bin/buf
COMPOSE ?= docker compose -f ../docker-compose.yml
SERVICE_DIRS := \
	services/api-service \
	services/auth-service \
	services/poll-service \
	services/vote-service \
	services/feed-service \
	services/analytics-service

help:
	@echo "Targets:"
	@echo "  proto-lint         - Update proto deps and run buf lint"
	@echo "  proto-breaking     - Check proto breaking changes against main"
	@echo "  proto-deps-update  - Update proto dependencies"
	@echo "  proto-generate     - Generate stubs and refresh api-service OpenAPI"
	@echo "  test-back          - Run tests in shared back module"
	@echo "  test-services      - Run tests in all service modules"
	@echo "  test-unit          - Run back + service tests"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  test-all           - Run unit and e2e tests"
	@echo "  vet-services       - Run go vet in all service modules"
	@echo "  tidy-services      - Run go mod tidy in all service modules"
	@echo "  compose-up         - Start local stack from docker-compose.yml"
	@echo "  compose-down       - Stop local stack"
	@echo "  compose-logs       - Tail local stack logs"
	@echo "  compose-ps         - Show stack status"
	@echo "  compose-restart    - Restart all stack services"

proto-lint:
	cd api && $(BUF) dep update && $(BUF) lint

proto-breaking:
	cd api && $(BUF) breaking --against '.git#branch=main'

proto-deps-update:
	cd api && $(BUF) dep update

proto-generate:
	cd api && $(BUF) generate
	cp api/gen/openapi/openapi.swagger.yaml services/api-service/api/openapi.yml
	awk 'BEGIN { inserted = 0 } { if (!inserted && $$0 ~ /^paths:$$/) { print "schemes:"; print "  - http"; print "host: localhost:8080"; print "basePath: /"; print "securityDefinitions:"; print "  BearerAuth:"; print "    type: apiKey"; print "    name: Authorization"; print "    in: header"; print "    description: '\''JWT access token. Example: Bearer <token>'\''"; print "security:"; print "  - BearerAuth: []"; inserted = 1 } print }' services/api-service/api/openapi.yml > services/api-service/api/openapi.yml.tmp
	mv services/api-service/api/openapi.yml.tmp services/api-service/api/openapi.yml

test-back:
	go test ./...

test-services:
	@set -e; \
	for dir in $(SERVICE_DIRS); do \
		echo "==> $$dir"; \
		$(MAKE) -C $$dir test; \
	done

test-unit: test-back test-services

test-e2e:
	go test -v ./tests/e2e/...

test-all: test-unit test-e2e

vet-services:
	@set -e; \
	for dir in $(SERVICE_DIRS); do \
		echo "==> $$dir"; \
		$(MAKE) -C $$dir vet; \
	done

tidy-services:
	@set -e; \
	for dir in $(SERVICE_DIRS); do \
		echo "==> $$dir"; \
		$(MAKE) -C $$dir tidy; \
	done

compose-up:
	$(COMPOSE) up -d --build --remove-orphans --wait

compose-down:
	$(COMPOSE) down --remove-orphans

compose-logs:
	$(COMPOSE) logs -f --tail=200

compose-ps:
	$(COMPOSE) ps

compose-restart:
	$(COMPOSE) restart