package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var AllowedModels = map[string]struct{}{
	"tts-1":                      {},
	"tts-1-hd":                   {},
	"gpt-4o-mini-tts":            {},
	"gpt-4o-mini-tts-2025-12-15": {},
}

var AllowedVoices = map[string]struct{}{
	"alloy": {}, "ash": {}, "ballad": {}, "cedar": {}, "coral": {}, "echo": {}, "fable": {},
	"marin": {}, "nova": {}, "onyx": {}, "sage": {}, "shimmer": {}, "verse": {},
}

var AllowedFormats = map[string]struct{}{
	"mp3": {}, "opus": {}, "aac": {}, "flac": {}, "wav": {}, "pcm": {},
}

type Voice struct {
	Name string
	ID   string
}

func (v *Voice) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		v.Name = name
		v.ID = ""
		return nil
	}
	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if len(obj) != 1 {
		return fmt.Errorf("voice object must contain only id")
	}
	id := strings.TrimSpace(obj["id"])
	if id == "" {
		return fmt.Errorf("voice.id is required")
	}
	v.Name = ""
	v.ID = id
	return nil
}

type SpeechRequest struct {
	Model          string   `json:"model"`
	Input          string   `json:"input"`
	Voice          Voice    `json:"voice"`
	Instructions   string   `json:"instructions,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	Speed          *float64 `json:"speed,omitempty"`
	StreamFormat   string   `json:"stream_format,omitempty"`
}

func DecodeSpeechRequest(dec *json.Decoder) (SpeechRequest, *Error) {
	dec.DisallowUnknownFields()
	var req SpeechRequest
	if err := dec.Decode(&req); err != nil {
		return req, Invalid("body", fmt.Sprintf("invalid JSON request body: %v", err))
	}
	return req, req.Validate()
}

func (r *SpeechRequest) Validate() *Error {
	if r.Model == "" {
		return Invalid("model", "model is required")
	}
	if _, ok := AllowedModels[r.Model]; !ok {
		return Invalid("model", "unsupported model")
	}
	if r.Input == "" {
		return Invalid("input", "input is required")
	}
	if len([]rune(r.Input)) > 4096 {
		return Invalid("input", "input must be at most 4096 characters")
	}
	if len([]rune(r.Instructions)) > 4096 {
		return Invalid("instructions", "instructions must be at most 4096 characters")
	}
	if r.Voice.Name == "" && r.Voice.ID == "" {
		return Invalid("voice", "voice is required")
	}
	if r.Voice.Name != "" {
		if _, ok := AllowedVoices[r.Voice.Name]; !ok {
			return Invalid("voice", "unsupported voice")
		}
	}
	if r.ResponseFormat == "" {
		r.ResponseFormat = "mp3"
	}
	if _, ok := AllowedFormats[r.ResponseFormat]; !ok {
		return Invalid("response_format", "unsupported response_format")
	}
	if r.Speed != nil && (*r.Speed < 0.25 || *r.Speed > 4.0) {
		return Invalid("speed", "speed must be between 0.25 and 4.0")
	}
	if r.StreamFormat == "" {
		r.StreamFormat = "audio"
	}
	if r.StreamFormat != "audio" && r.StreamFormat != "sse" {
		return Invalid("stream_format", "unsupported stream_format")
	}
	return nil
}

func (r SpeechRequest) UpstreamSpeaker(defaultSpeaker string) string {
	if r.Voice.ID != "" {
		return r.Voice.ID
	}
	return defaultSpeaker
}

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    *string `json:"code"`
}

type Error struct {
	Status int
	Body   ErrorEnvelope
}

func Invalid(param, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Body: ErrorEnvelope{Error: ErrorBody{Message: message, Type: "invalid_request_error", Param: &param}}}
}

func Unsupported(param, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Body: ErrorEnvelope{Error: ErrorBody{Message: message, Type: "unsupported_feature_error", Param: &param}}}
}

func Upstream(message string) *Error {
	return &Error{Status: http.StatusBadGateway, Body: ErrorEnvelope{Error: ErrorBody{Message: message, Type: "upstream_error"}}}
}

func Server(message string) *Error {
	return &Error{Status: http.StatusInternalServerError, Body: ErrorEnvelope{Error: ErrorBody{Message: message, Type: "server_error"}}}
}
