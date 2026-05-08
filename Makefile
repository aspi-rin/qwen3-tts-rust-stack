SHELL := /usr/bin/env bash

-include .env
export

# Upstream v0.1.6 only publishes a Linux Vulkan release asset.
# BACKEND=cpu uses that same binary without GPU device passthrough, relying on
# llama.cpp/ggml CPU fallback. BACKEND=vulkan passes /dev/dri into the container.
BACKEND ?= $(or $(QWEN3_TTS_BACKEND),vulkan)
COMPOSE_CMD ?= docker compose

SUPPORTED_BACKENDS := cpu vulkan
ifeq ($(filter $(BACKEND),$(SUPPORTED_BACKENDS)),)
$(error Unsupported BACKEND=$(BACKEND). Use one of: $(SUPPORTED_BACKENDS))
endif

QWEN3_TTS_IMAGE_cpu ?= qwen3-tts-rust-stack:v0.1.6-cpu
QWEN3_TTS_IMAGE_vulkan ?= qwen3-tts-rust-stack:v0.1.6-vulkan
IMAGE ?= $(or $(QWEN3_TTS_IMAGE),$(QWEN3_TTS_IMAGE_$(BACKEND)))

COMPOSE_FILES_cpu := -f docker-compose.yml -f docker-compose.cpu.yml
COMPOSE_FILES_vulkan := -f docker-compose.yml -f docker-compose.vulkan.yml
COMPOSE_FILES := $(COMPOSE_FILES_$(BACKEND))
COMPOSE := QWEN3_TTS_IMAGE=$(IMAGE) $(COMPOSE_CMD) $(COMPOSE_FILES)

.PHONY: check build-image run down clean

check:
	@$(COMPOSE_CMD) version >/dev/null 2>&1 || (echo "Compose command failed: $(COMPOSE_CMD). Install Docker Compose plugin or run with COMPOSE_CMD=docker-compose" >&2; exit 2)
	@case "$(BACKEND)" in \
		cpu) echo "Backend cpu: using Linux Vulkan release without GPU passthrough; expects CPU fallback" ;; \
		vulkan) [ -e /dev/dri ] || (echo "Missing /dev/dri for Vulkan backend" >&2; exit 2); echo "Backend vulkan: /dev/dri found" ;; \
	esac

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
