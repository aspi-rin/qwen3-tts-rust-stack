package openai

import (
	"encoding/json"
	"strings"
	"testing"
)

func decode(t *testing.T, body string) (SpeechRequest, *Error) {
	t.Helper()
	return DecodeSpeechRequest(json.NewDecoder(strings.NewReader(body)))
}

func TestDecodeSpeechRequestDefaults(t *testing.T) {
	req, err := decode(t, `{"model":"gpt-4o-mini-tts","input":"hello","voice":"alloy"}`)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if req.ResponseFormat != "mp3" {
		t.Fatalf("response format default = %q", req.ResponseFormat)
	}
	if req.StreamFormat != "audio" {
		t.Fatalf("stream format default = %q", req.StreamFormat)
	}
}

func TestDecodeSpeechRequestRejectsUnknownField(t *testing.T) {
	_, err := decode(t, `{"model":"gpt-4o-mini-tts","input":"hello","voice":"alloy","seed":42}`)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestDecodeSpeechRequestCustomVoiceID(t *testing.T) {
	req, err := decode(t, `{"model":"gpt-4o-mini-tts","input":"hello","voice":{"id":"vivian"},"response_format":"pcm"}`)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if got := req.UpstreamSpeaker("fallback"); got != "vivian" {
		t.Fatalf("speaker = %q", got)
	}
}
