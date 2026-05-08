package upstream

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("upstream health returned %s", res.Status)
	}
	return nil
}

type TTSRequest struct {
	Text        string `json:"text"`
	Speaker     string `json:"speaker,omitempty"`
	Instruction string `json:"instruction,omitempty"`
}

type TTSResponse struct {
	Success     bool     `json:"success"`
	Message     *string  `json:"message"`
	AudioBase64 *string  `json:"audio_base64"`
	SampleRate  *uint32  `json:"sample_rate"`
	DurationMS  *float64 `json:"duration_ms"`
}

func (c *Client) GenerateWAV(ctx context.Context, req TTSRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/tts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("upstream /api/tts returned %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	var out TTSResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Success {
		msg := "upstream tts failed"
		if out.Message != nil && *out.Message != "" {
			msg = *out.Message
		}
		return nil, errors.New(msg)
	}
	if out.AudioBase64 == nil || *out.AudioBase64 == "" {
		return nil, fmt.Errorf("upstream response did not include audio_base64")
	}
	wav, err := base64.StdEncoding.DecodeString(*out.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("decode upstream audio_base64: %w", err)
	}
	return wav, nil
}

type StreamChunk struct {
	Binary []byte
	Text   string
}

type Stream struct {
	conn *websocket.Conn
}

func (c *Client) OpenStream(ctx context.Context, req TTSRequest) (*Stream, error) {
	wsURL := strings.TrimPrefix(c.BaseURL, "http://")
	wsURL = strings.TrimPrefix(wsURL, "https://")
	if strings.HasPrefix(c.BaseURL, "https://") {
		wsURL = "wss://" + wsURL
	} else {
		wsURL = "ws://" + wsURL
	}
	wsURL += "/api/tts/stream"

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	if err := wsjson.Write(ctx, conn, req); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "write request failed")
		return nil, err
	}
	return &Stream{conn: conn}, nil
}

func (s *Stream) Read(ctx context.Context) (StreamChunk, error) {
	msgType, data, err := s.conn.Read(ctx)
	if err != nil {
		return StreamChunk{}, err
	}
	switch msgType {
	case websocket.MessageBinary:
		return StreamChunk{Binary: data}, nil
	case websocket.MessageText:
		return StreamChunk{Text: string(data)}, nil
	default:
		return StreamChunk{}, nil
	}
}

func (s *Stream) Close() error {
	return s.conn.Close(websocket.StatusNormalClosure, "done")
}
