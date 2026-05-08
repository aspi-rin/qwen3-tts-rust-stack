package config

import (
	"os"
	"strings"
)

type Config struct {
	ListenAddr      string
	UpstreamBaseURL string
	DefaultSpeaker  string
}

func Load() Config {
	return Config{
		ListenAddr:      env("OPENAI_API_LISTEN_ADDR", ":8080"),
		UpstreamBaseURL: strings.TrimRight(env("QWEN3_TTS_UPSTREAM_URL", "http://qwen3-tts-server:3000"), "/"),
		DefaultSpeaker:  env("QWEN3_TTS_DEFAULT_SPEAKER", "vivian"),
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
