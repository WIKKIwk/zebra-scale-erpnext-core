SHELL := /bin/sh

FLUTTER ?= flutter
FLUTTER_DEVICE ?=
FLUTTER_WEB_DEVICE ?= chrome
FLUTTER_RUN_ARGS ?=
ANDROID_EMULATOR_ID ?= gscale_atd35
ANDROID_SDK_ROOT ?= $(HOME)/Android/Sdk
ANDROID_EMULATOR_GPU ?= host
ANDROID_FLUTTER_MODE ?= --profile
ANDROID_REVERSE_PORTS ?= 8081 18000
RUN_DEV_PLATFORM ?= auto
RUN_DEVICE_ARG := $(if $(strip $(FLUTTER_DEVICE)),-d $(FLUTTER_DEVICE),)

.PHONY: help pub-get devices emulators run run-auto run-linux run-android run-ios run-web analyze test build-linux clean

help:
	@echo "Targets:"
	@echo "  make run         - Flutter app'ni ishga tushiradi"
	@echo "  make run-auto    - mos device'ni tanlab ishga tushiradi"
	@echo "  make run-linux   - Linux desktop'da ishga tushiradi"
	@echo "  make run-android - Android device/emulator'da ishga tushiradi"
	@echo "  make run-ios     - iOS device/simulator'da ishga tushiradi"
	@echo "  make run-web     - Web (Chrome) da ishga tushiradi"
	@echo "  make devices     - ulangan device'lar ro'yxati"
	@echo "  make emulators   - emulatorlar ro'yxati"
	@echo "  make analyze     - static analiz"
	@echo "  make test        - widget testlar"
	@echo "  make pub-get     - dependencies yuklash"
	@echo "  make clean       - Flutter build cache tozalash"
	@echo ""
	@echo "Override:"
	@echo "  make run FLUTTER_DEVICE=android"
	@echo "  make run FLUTTER_DEVICE=chrome"
	@echo "  make run FLUTTER_DEVICE=web-server"
	@echo "  make run FLUTTER_DEVICE=linux"
	@echo "  make run-android ANDROID_EMULATOR_ID=gscale_api35"
	@echo "  make run FLUTTER_RUN_ARGS='--dart-define=API_BASE_URL=http://127.0.0.1:8081'"
	@echo ""
	@echo "Eslatma:"
	@echo "  iOS build/run odatda faqat macOS'da ishlaydi."

pub-get:
	$(FLUTTER) pub get

devices:
	$(FLUTTER) devices

emulators:
	$(FLUTTER) emulators

run: pub-get
	$(FLUTTER) run $(RUN_DEVICE_ARG) $(FLUTTER_RUN_ARGS)

run-auto:
	@set -e; \
	if [ "$(RUN_DEV_PLATFORM)" = "android" ]; then \
		$(MAKE) run-android FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
	elif [ "$(RUN_DEV_PLATFORM)" = "linux" ]; then \
		$(MAKE) run-linux FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
	elif [ "$(RUN_DEV_PLATFORM)" = "web" ]; then \
		$(MAKE) run-web FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
	elif adb devices | awk '/^[^[:space:]]+[[:space:]]+device$$/ {print $$1; exit}' | grep -q .; then \
		$(MAKE) run-android FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
	else \
		$(FLUTTER) devices >/tmp/gscale-flutter-devices.txt; \
		if grep -q 'Linux (desktop)' /tmp/gscale-flutter-devices.txt; then \
		echo "Auto device: linux desktop"; \
		$(MAKE) run-linux FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
		else \
		echo "Auto device: android emulator"; \
		$(MAKE) run-android FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"; \
		fi; \
	fi

run-linux:
	$(MAKE) run FLUTTER_DEVICE=linux FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"

run-android:
	@set -e; \
	$(FLUTTER) pub get; \
	SERIAL="$$(adb devices | awk '/^emulator-/{print $$1; exit}')"; \
	if [ -z "$$SERIAL" ]; then \
		echo "Launching Android emulator: $(ANDROID_EMULATOR_ID)"; \
		nohup "$(ANDROID_SDK_ROOT)/emulator/emulator" \
			-avd "$(ANDROID_EMULATOR_ID)" \
			-no-snapshot-save \
			-no-boot-anim \
			-no-audio \
			-gpu "$(ANDROID_EMULATOR_GPU)" \
			>/tmp/gscale-android-emulator.log 2>&1 & \
		echo $$! >/tmp/gscale-android-emulator.pid; \
		for i in $$(seq 1 60); do \
			SERIAL="$$(adb devices | awk '/^emulator-/{print $$1; exit}')"; \
			if [ -n "$$SERIAL" ]; then \
				break; \
			fi; \
			sleep 2; \
		done; \
	fi; \
	if [ -z "$$SERIAL" ]; then \
		echo "Android emulator topilmadi."; \
		exit 1; \
	fi; \
	echo "Waiting for emulator boot: $$SERIAL"; \
	adb -s "$$SERIAL" wait-for-device >/dev/null 2>&1; \
	for i in $$(seq 1 90); do \
		BOOTED="$$(adb -s "$$SERIAL" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')"; \
		if [ "$$BOOTED" = "1" ]; then \
			break; \
		fi; \
		sleep 2; \
	done; \
	BOOTED="$$(adb -s "$$SERIAL" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')"; \
	if [ "$$BOOTED" != "1" ]; then \
		echo "Android emulator boot bo'lmadi: $$SERIAL"; \
		exit 1; \
	fi; \
	for port in $(ANDROID_REVERSE_PORTS); do \
		adb -s "$$SERIAL" reverse "tcp:$$port" "tcp:$$port" >/dev/null 2>&1 || true; \
	done; \
	echo "Running on $$SERIAL"; \
	$(FLUTTER) run $(ANDROID_FLUTTER_MODE) -d "$$SERIAL" $(FLUTTER_RUN_ARGS)

run-ios:
	$(MAKE) run FLUTTER_DEVICE=ios FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"

run-web:
	$(MAKE) run FLUTTER_DEVICE=$(FLUTTER_WEB_DEVICE) FLUTTER_RUN_ARGS="$(FLUTTER_RUN_ARGS)"

analyze: pub-get
	$(FLUTTER) analyze

test: pub-get
	$(FLUTTER) test

build-linux: pub-get
	$(FLUTTER) build linux $(FLUTTER_RUN_ARGS)

clean:
	$(FLUTTER) clean
