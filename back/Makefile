.PHONY: proto-lint proto-breaking proto-generate

BUF ?= /home/yohn/go/bin/buf

proto-lint:
	cd api && $(BUF) dep update && $(BUF) lint

proto-breaking:
	cd api && $(BUF) breaking --against '.git#branch=main'

proto-generate:
	cd api && $(BUF) dep update && $(BUF) generate