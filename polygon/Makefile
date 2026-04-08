SHELL := /bin/sh

BRIDGE_STATE_FILE ?= /tmp/gscale-zebra/bridge_state.json
HTTP_ADDR ?= 127.0.0.1:18000
PRINT_MODE ?= success
PRINT_DELAY ?= 1100ms

.PHONY: help run build test

help:
	@echo "Targets:"
	@echo "  make run   - polygon simulatorni ishga tushiradi"
	@echo "  make build - polygon binary build qiladi"
	@echo "  make test  - polygon testlarini yugurtiradi"
	@echo ""
	@echo "Override:"
	@echo "  make run HTTP_ADDR=0.0.0.0:18000"
	@echo "  make run PRINT_MODE=alternate"

run:
	go run . --http-addr "$(HTTP_ADDR)" --bridge-state-file "$(BRIDGE_STATE_FILE)" --print-mode "$(PRINT_MODE)" --print-delay "$(PRINT_DELAY)"

build:
	go build .

test:
	go test ./...
