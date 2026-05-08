SHELL := /usr/bin/env bash

-include .env
export

BACKEND ?= $(or $(QWEN3_TTS_BACKEND),vulkan)
COMPOSE_CMD ?= docker compose

ifeq ($(BACKEND),cpu)
IMAGE := qwen3-tts-rust-stack:v0.1.6-cpu
COMPOSE_FILES := -f docker-compose.yml -f docker-compose.cpu.yml
CHECK_BACKEND := @echo "Backend cpu: no GPU passthrough; using upstream Linux Vulkan binary with CPU fallback"
else ifeq ($(BACKEND),vulkan)
IMAGE := qwen3-tts-rust-stack:v0.1.6-vulkan
COMPOSE_FILES := -f docker-compose.yml -f docker-compose.vulkan.yml
CHECK_BACKEND := @test -e /dev/dri || (echo "Missing /dev/dri for Vulkan backend" >&2; exit 2); echo "Backend vulkan: /dev/dri found"
else
$(error Unsupported BACKEND=$(BACKEND). Use BACKEND=cpu or BACKEND=vulkan)
endif

COMPOSE := $(COMPOSE_CMD) $(COMPOSE_FILES)

.PHONY: check build-image run down clean

check:
	@$(COMPOSE_CMD) version >/dev/null 2>&1 || (echo "Compose command failed: $(COMPOSE_CMD)" >&2; exit 2)
	$(CHECK_BACKEND)

build-image:
	docker build -f docker/Dockerfile \
		--build-arg QWEN3_TTS_RUNTIME_BACKEND=$(BACKEND) \
		-t $(IMAGE) .

run: check
	mkdir -p models outputs
	$(COMPOSE) run --rm qwen3-tts

down:
	$(COMPOSE) down

clean:
	rm -rf outputs
