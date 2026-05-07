SHELL := /usr/bin/env bash

-include .env
export

BACKEND ?= $(or $(QWEN3_TTS_BACKEND),vulkan)
COMPOSE_CMD ?= docker compose

SUPPORTED_BACKENDS := cpu vulkan cuda
ifeq ($(filter $(BACKEND),$(SUPPORTED_BACKENDS)),)
$(error Unsupported BACKEND=$(BACKEND). Use one of: $(SUPPORTED_BACKENDS))
endif

QWEN3_TTS_IMAGE_cpu ?= qwen3-tts-rust-stack:v0.1.6-cpu
QWEN3_TTS_IMAGE_vulkan ?= qwen3-tts-rust-stack:v0.1.6-vulkan
QWEN3_TTS_IMAGE_cuda ?= qwen3-tts-rust-stack:cuda
IMAGE ?= $(or $(QWEN3_TTS_IMAGE),$(QWEN3_TTS_IMAGE_$(BACKEND)))

COMPOSE_FILES_cpu := -f docker-compose.yml -f docker-compose.cpu.yml
COMPOSE_FILES_vulkan := -f docker-compose.yml -f docker-compose.vulkan.yml
COMPOSE_FILES_cuda := -f docker-compose.yml -f docker-compose.cuda.yml
COMPOSE_FILES := $(COMPOSE_FILES_$(BACKEND))
COMPOSE := QWEN3_TTS_IMAGE=$(IMAGE) $(COMPOSE_CMD) $(COMPOSE_FILES)

.PHONY: check build-image run down check-vulkan clean

check:
	@$(COMPOSE_CMD) version >/dev/null 2>&1 || (echo "Compose command failed: $(COMPOSE_CMD). Install Docker Compose plugin or run with COMPOSE_CMD=docker-compose" >&2; exit 2)
	@case "$(BACKEND)" in \
		cpu) echo "Backend cpu: no GPU device is passed through" ;; \
		vulkan) [ -e /dev/dri ] || (echo "Missing /dev/dri for Vulkan backend" >&2; exit 2); echo "Backend vulkan: /dev/dri found" ;; \
		cuda) command -v nvidia-smi >/dev/null || (echo "nvidia-smi not found; install NVIDIA driver/container toolkit for CUDA backend" >&2; exit 2); nvidia-smi -L ;; \
	esac

build-image:
	@if [ "$(BACKEND)" = "cuda" ]; then \
		echo "Linux CUDA image is not available from upstream Qwen3-TTS-Rust release v0.1.6." >&2; \
		echo "Windows CUDA release exists, but it is not usable for Linux Docker." >&2; \
		echo "Use BACKEND=cpu or BACKEND=vulkan for now, or implement docker/Dockerfile.cuda from source." >&2; \
		exit 2; \
	fi
	docker build -f docker/Dockerfile \
		--build-arg QWEN3_TTS_RUNTIME_BACKEND=$(BACKEND) \
		-t $(IMAGE) .

run: check
	$(COMPOSE) run --rm qwen3-tts

down:
	$(COMPOSE) down

check-vulkan:
	./scripts/verify_vulkan.sh

clean:
	rm -rf outputs/*
