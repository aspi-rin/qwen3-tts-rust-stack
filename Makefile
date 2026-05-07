.PHONY: bootstrap build smoke bench docker-build docker-smoke docker-vulkan-test verify-vulkan clean

bootstrap:
	./scripts/bootstrap_upstream.sh

build:
	./scripts/build_cli.sh

smoke:
	./scripts/run_cli.sh "你好，我是本地语音合成服务。" data/outputs/smoke.wav

bench:
	./scripts/benchmark_cli.py --rounds 3

docker-build:
	docker build -f docker/Dockerfile -t qwen3-tts-rust-stack:local .

docker-smoke:
	docker compose run --rm qwen3-tts-cli

docker-vulkan-test:
	docker compose --profile gpu run --rm qwen3-tts-vulkan

verify-vulkan:
	./scripts/verify_vulkan.sh

clean:
	rm -rf upstream target data/outputs/*
