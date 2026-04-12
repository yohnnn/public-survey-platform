.PHONY: proto-lint proto-breaking proto-deps-update proto-generate

BUF ?= /home/yohn/go/bin/buf

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