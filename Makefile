SHELL := /usr/bin/env bash

-include .env
export

BACKEND ?= $(or $(QWEN3_TTS_BACKEND),vulkan)
QUANT ?= $(or $(QWEN3_TTS_QUANT),q5_k_m)
# Llama.cpp release tag the source build pins to for the cuda backend. Must
# match the GGUF format produced by upstream Qwen3-TTS-Rust (currently b7885);
# bumping this without re-quantizing the model files crashes inside the talker
# graph.
LLAMA_TAG ?= b7885
COMPOSE_CMD ?= docker compose
CARGO ?= cargo
PREPARE_MANIFEST := tools/qwen3-tts-model-download/Cargo.toml

COMPOSE_FILES_cpu := -f docker-compose.yml -f docker-compose.cpu.yml
COMPOSE_FILES_vulkan := -f docker-compose.yml -f docker-compose.vulkan.yml
COMPOSE_FILES_cuda := -f docker-compose.yml -f docker-compose.cuda.yml

IMAGE_cpu := qwen3-tts-rust-stack:v0.1.6-cpu
IMAGE_vulkan := qwen3-tts-rust-stack:v0.1.6-vulkan
IMAGE_cuda := qwen3-tts-rust-stack:v0.1.6-cuda
ADAPTER_IMAGE := qwen3-tts-openai-api:latest
GO_ADAPTER_DIR := services/openai-api

# cuda uses a separate Dockerfile because the runtime base must be
# nvidia/cuda:* and the llama.cpp libs must come from a CUDA-built source
# (llama.cpp upstream does not publish a Linux CUDA tarball that the rust
# binary's auto-download path would find).
DOCKERFILE_cpu := docker/Dockerfile
DOCKERFILE_vulkan := docker/Dockerfile
DOCKERFILE_cuda := docker/Dockerfile.cuda

SUPPORTED_BACKENDS := cpu vulkan cuda
SUPPORTED_QUANTS := none q5_k_m q8_0
ifeq ($(filter $(BACKEND),$(SUPPORTED_BACKENDS)),)
$(error Unsupported BACKEND=$(BACKEND). Use one of: $(SUPPORTED_BACKENDS))
endif
ifeq ($(filter $(QUANT),$(SUPPORTED_QUANTS)),)
$(error Unsupported QUANT=$(QUANT). Use one of: $(SUPPORTED_QUANTS))
endif

COMPOSE_FILES := $(COMPOSE_FILES_$(BACKEND))
IMAGE := $(IMAGE_$(BACKEND))
DOCKERFILE := $(DOCKERFILE_$(BACKEND))
COMPOSE := $(COMPOSE_CMD) $(COMPOSE_FILES)

.PHONY: check fmt fmt-check test up down

check:
	@test -f upstream/src/models/onnx.rs || (echo "Missing upstream submodule contents. Run: git submodule update --init --recursive" >&2; exit 2)
	@$(COMPOSE_CMD) version >/dev/null 2>&1 || (echo "Compose command failed: $(COMPOSE_CMD). Install Docker Compose plugin or run with COMPOSE_CMD=docker-compose" >&2; exit 2)
	@case "$(BACKEND)" in \
		cpu) echo "Backend cpu: no GPU passthrough; using upstream Linux Vulkan runtime bundle with CPU fallback" ;; \
		vulkan) test -e /dev/dri || (echo "Missing /dev/dri for Vulkan backend" >&2; exit 2); echo "Backend vulkan: /dev/dri found" ;; \
		cuda) command -v nvidia-smi >/dev/null 2>&1 || (echo "Missing nvidia-smi; cuda backend needs the NVIDIA driver + NVIDIA Container Toolkit installed on the host" >&2; exit 2); echo "Backend cuda: nvidia-smi found ($$(nvidia-smi --query-gpu=name --format=csv,noheader | head -1))" ;; \
	esac

fmt:
	$(CARGO) fmt --manifest-path $(PREPARE_MANIFEST)
	cd $(GO_ADAPTER_DIR) && gofmt -w .

fmt-check:
	$(CARGO) fmt --manifest-path $(PREPARE_MANIFEST) -- --check
	@test -z "$$(cd $(GO_ADAPTER_DIR) && gofmt -l .)"

test:
	$(CARGO) test --manifest-path $(PREPARE_MANIFEST) --lib --no-default-features
	cd $(GO_ADAPTER_DIR) && go test ./...

up: check
	mkdir -p models
	docker build -f $(DOCKERFILE) \
		--build-arg QWEN3_TTS_RUNTIME_BACKEND=$(BACKEND) \
		--build-arg LLAMA_TAG=$(LLAMA_TAG) \
		-t $(IMAGE) .
	docker build -f $(GO_ADAPTER_DIR)/Dockerfile -t $(ADAPTER_IMAGE) .
	$(COMPOSE) up -d --force-recreate

down:
	$(COMPOSE) down

