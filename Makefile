SHELL := /bin/sh

SCALE_DEVICE ?= /dev/ttyUSB0
ZEBRA_DEVICE ?= /dev/usb/lp0
BRIDGE_STATE_FILE ?= /tmp/gscale-zebra/bridge_state.json
POLYGON_HTTP_ADDR ?= 127.0.0.1:18000
APP_USER ?= $(shell id -un)
APP_GROUP ?= $(shell id -gn)

.PHONY: help check-env build build-bot build-scale build-zebra build-polygon build-mobileapi run run-scale run-bot run-polygon run-test run-mobileapi test test-polygon test-mobileapi clean release release-all autostart-install autostart-status autostart-restart autostart-stop

help:
	@echo "Targets:"
	@echo "  make run        - scale TUI ni ishga tushiradi (bot auto-start bilan)"
	@echo "  make run-scale  - faqat scale TUI (bot auto-startsiz)"
	@echo "  make run-bot    - faqat telegram bot"
	@echo "  make run-polygon - real qurilmasiz polygon simulator"
	@echo "  make run-test   - polygon + scale TUI (qurilmasiz core test)"
	@echo "  make run-mobileapi - mobile API backend"
	@echo "  make build      - bot + scale + zebra binary build (./bin)"
	@echo "  make build-polygon - polygon binary build (./bin)"
	@echo "  make build-mobileapi - mobile API binary build (./bin)"
	@echo "  make test       - barcha modullarda test"
	@echo "  make test-polygon - polygon modul testlari"
	@echo "  make test-mobileapi - mobile API testlari"
	@echo "  make autostart-install - systemd service'larni o'rnatadi va start qiladi"
	@echo "  make autostart-status  - service holatini ko'rsatadi"
	@echo "  make autostart-restart - service'larni restart qiladi"
	@echo "  make autostart-stop    - service'larni to'xtatadi"
	@echo "  make release    - linux/amd64 tar release"
	@echo "  make release-all - linux/amd64 + linux/arm64 tar release"
	@echo "  make clean      - local build papkalarini tozalash"
	@echo ""
	@echo "Override:"
	@echo "  make run SCALE_DEVICE=/dev/ttyUSB1 ZEBRA_DEVICE=/dev/usb/lp0"

check-env:
	@test -f bot/.env || (echo "xato: bot/.env topilmadi (bot/.env.example dan nusxa oling)"; exit 1)

build: build-bot build-scale build-zebra

build-bot:
	@mkdir -p bin
	go build -o ./bin/bot ./bot/cmd/bot

build-scale:
	@mkdir -p bin
	go build -o ./bin/scale ./scale

build-zebra:
	@mkdir -p bin
	go build -o ./bin/zebra ./zebra

build-polygon:
	@mkdir -p bin
	go build -o ./bin/polygon ./polygon

build-mobileapi:
	@mkdir -p bin
	go build -o ./bin/mobileapi ./cmd/mobileapi

run: check-env
	cd scale && go run . --no-bridge --device "$(SCALE_DEVICE)" --zebra-device "$(ZEBRA_DEVICE)" --bridge-state-file "$(BRIDGE_STATE_FILE)"

run-scale:
	cd scale && go run . --no-bot --no-bridge --device "$(SCALE_DEVICE)" --zebra-device "$(ZEBRA_DEVICE)" --bridge-state-file "$(BRIDGE_STATE_FILE)"

run-bot: check-env
	cd bot && go run ./cmd/bot

run-polygon:
	cd polygon && go run .

run-test:
	@mkdir -p /tmp/gscale-zebra
	@POLY_PID=""; \
	trap 'if [ -n "$$POLY_PID" ]; then kill $$POLY_PID 2>/dev/null || true; fi' EXIT INT TERM; \
	(cd polygon && go run . --http-addr "$(POLYGON_HTTP_ADDR)" --bridge-state-file "$(BRIDGE_STATE_FILE)" >/tmp/gscale-zebra/polygon.log 2>&1) & \
	POLY_PID=$$!; \
	sleep 1; \
	cd scale && go run . --no-bot --no-zebra --bridge-url "http://$(POLYGON_HTTP_ADDR)/api/v1/scale" --bridge-state-file "$(BRIDGE_STATE_FILE)"

run-mobileapi:
	go run ./cmd/mobileapi

test:
	cd bot && go test ./...
	cd bridge && go test ./...
	cd scale && go test ./...
	cd core && GOWORK=off go test ./...

test-polygon:
	cd polygon && go test ./...

test-mobileapi:
	go test ./internal/mobileapi ./cmd/mobileapi

clean:
	@if [ -d ./bin ]; then find ./bin -type f -delete; find ./bin -type d -empty -delete; fi
	@if [ -d ./dist ]; then find ./dist -type f -delete; find ./dist -type d -empty -delete; fi

autostart-install: check-env build
	sudo ./deploy/install.sh --user "$(APP_USER)" --group "$(APP_GROUP)" --start

autostart-status:
	sudo systemctl --no-pager --full status gscale-scale.service gscale-bot.service

autostart-restart:
	sudo systemctl restart gscale-scale.service gscale-bot.service

autostart-stop:
	sudo systemctl stop gscale-scale.service gscale-bot.service

release:
	./scripts/release.sh --arch amd64

release-all:
	./scripts/release.sh --arch amd64 --arch arm64
