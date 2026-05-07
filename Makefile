SHELL := /usr/bin/env bash

-include .env
export

BACKEND ?= $(or $(QWEN3_TTS_BACKEND),vulkan)
COMPOSE_CMD ?= docker compose
IMAGE ?= $(or $(QWEN3_TTS_IMAGE),qwen3-tts-rust-stack:v0.1.6-vulkan)

COMPOSE_FILES_none := -f docker-compose.yml
COMPOSE_FILES_vulkan := -f docker-compose.yml -f docker-compose.vulkan.yml
SUPPORTED_BACKENDS := none vulkan
ifeq ($(filter $(BACKEND),$(SUPPORTED_BACKENDS)),)
$(error Unsupported BACKEND=$(BACKEND). Use one of: $(SUPPORTED_BACKENDS))
endif

COMPOSE_FILES := $(COMPOSE_FILES_$(BACKEND))
COMPOSE := QWEN3_TTS_IMAGE=$(IMAGE) $(COMPOSE_CMD) $(COMPOSE_FILES)

.PHONY: check build-image run down check-vulkan clean

check:
	@$(COMPOSE_CMD) version >/dev/null 2>&1 || (echo "Compose command failed: $(COMPOSE_CMD). Install Docker Compose plugin or run with COMPOSE_CMD=docker-compose" >&2; exit 2)
	@case "$(BACKEND)" in \
		none) echo "Backend none: no GPU device is passed through" ;; \
		vulkan) [ -e /dev/dri ] || (echo "Missing /dev/dri for Vulkan backend" >&2; exit 2); echo "Backend vulkan: /dev/dri found" ;; \
	esac

build-image:
	docker build -f docker/Dockerfile -t $(IMAGE) .

run: check
	$(COMPOSE) run --rm qwen3-tts

down:
	$(COMPOSE) down

check-vulkan:
	./scripts/verify_vulkan.sh

clean:
	rm -rf outputs/*
