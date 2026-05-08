package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/vex9z7/qwen3-tts-rust-stack/services/openai-api/internal/audio"
	"github.com/vex9z7/qwen3-tts-rust-stack/services/openai-api/internal/config"
	compat "github.com/vex9z7/qwen3-tts-rust-stack/services/openai-api/internal/openai"
	"github.com/vex9z7/qwen3-tts-rust-stack/services/openai-api/internal/upstream"
)

type Server struct {
	cfg      config.Config
	upstream *upstream.Client
	proxy    *httputil.ReverseProxy
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("qwen3_openai_api - OpenAI-compatible API adapter for Qwen3-TTS-Rust")
		fmt.Println("env: OPENAI_API_LISTEN_ADDR, QWEN3_TTS_UPSTREAM_URL, QWEN3_TTS_DEFAULT_SPEAKER")
		return
	}

	cfg := config.Load()
	up := upstream.New(cfg.UpstreamBaseURL)
	proxy := newReverseProxy(cfg.UpstreamBaseURL)
	s := &Server{cfg: cfg, upstream: up, proxy: proxy}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/models", s.handleModels)
	mux.HandleFunc("POST /v1/audio/speech", s.handleSpeech)

	// Qwen3 upstream-compatible API bypass. This keeps existing /api/tts,
	// /api/tts/stream, and /api/speakers clients working through the adapter.
	mux.Handle("/api/", s.proxy)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("starting OpenAI-compatible Qwen3 TTS adapter", "addr", cfg.ListenAddr, "upstream", cfg.UpstreamBaseURL)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "err", err)
	}
}

func newReverseProxy(upstreamBaseURL string) *httputil.ReverseProxy {
	target, err := url.Parse(upstreamBaseURL)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalHost := r.Host
		originalDirector(r)
		r.Host = target.Host
		r.Header.Set("X-Forwarded-Host", originalHost)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, compat.Upstream("upstream proxy failed: "+err.Error()))
	}
	return proxy
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.upstream.Health(ctx); err != nil {
		writeError(w, compat.Upstream("upstream health check failed: "+err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, compat.ModelList{Object: "list", Data: []compat.Model{
		{ID: "qwen3-tts", Object: "model", Created: 0, OwnedBy: "local"},
	}})
}

func (s *Server) handleSpeech(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			writeError(w, compat.Invalid("Content-Type", "Content-Type must be application/json"))
			return
		}
	}
	defer r.Body.Close()
	req, reqErr := compat.DecodeSpeechRequest(json.NewDecoder(r.Body))
	if reqErr != nil {
		writeError(w, reqErr)
		return
	}
	if req.StreamFormat == "sse" {
		writeError(w, compat.Unsupported("stream_format", "stream_format=sse is not implemented yet"))
		return
	}
	switch req.ResponseFormat {
	case "pcm":
		s.handleSpeechPCM(w, r, req)
	case "wav":
		s.handleSpeechWAV(w, r, req)
	default:
		writeError(w, compat.Unsupported("response_format", "only response_format=pcm and response_format=wav are supported"))
	}
}

func (s *Server) handleSpeechWAV(w http.ResponseWriter, r *http.Request, req compat.SpeechRequest) {
	wav, err := s.upstream.GenerateWAV(r.Context(), upstream.TTSRequest{
		Text:        req.Input,
		Speaker:     req.UpstreamSpeaker(s.cfg.DefaultSpeaker),
		Instruction: req.Instructions,
	})
	if err != nil {
		writeError(w, compat.Upstream(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", intToString(len(wav)))
	_, _ = w.Write(wav)
}

func (s *Server) handleSpeechPCM(w http.ResponseWriter, r *http.Request, req compat.SpeechRequest) {
	stream, err := s.upstream.OpenStream(r.Context(), upstream.TTSRequest{
		Text:        req.Input,
		Speaker:     req.UpstreamSpeaker(s.cfg.DefaultSpeaker),
		Instruction: req.Instructions,
	})
	if err != nil {
		writeError(w, compat.Upstream("open upstream stream: "+err.Error()))
		return
	}
	defer stream.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, compat.Server("response writer does not support streaming"))
		return
	}
	w.Header().Set("Content-Type", "audio/pcm")
	w.Header().Set("X-Audio-Sample-Rate", "24000")
	w.Header().Set("X-Audio-Channels", "1")
	w.Header().Set("X-Audio-Format", "s16le")
	w.WriteHeader(http.StatusOK)

	for {
		chunk, err := stream.Read(r.Context())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || errors.Is(err, context.Canceled) {
				return
			}
			slog.Warn("upstream stream read failed", "err", err)
			return
		}
		if len(chunk.Binary) > 0 {
			if _, err := w.Write(audio.F32LEToS16LE(chunk.Binary)); err != nil {
				return
			}
			flusher.Flush()
			continue
		}
		if chunk.Text == "segment_done" || chunk.Text == "done" {
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, e *compat.Error) {
	writeJSON(w, e.Status, e.Body)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
