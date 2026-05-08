#!/usr/bin/env python3
"""Minimal stdlib WebSocket smoke client for Qwen3-TTS streaming.

It sends one upstream-compatible /api/tts/stream request, records received
binary f32le PCM chunks, writes raw PCM and WAV files, and prints TTFB/RTF
metrics. No websocat or third-party Python packages are required.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import socket
import ssl
import struct
import sys
import time
import wave
from dataclasses import dataclass
from typing import BinaryIO
from urllib.parse import urlparse

GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
SAMPLE_RATE = 24_000


@dataclass
class Frame:
    opcode: int
    payload: bytes


class WebSocketClient:
    def __init__(self, url: str, timeout: float) -> None:
        parsed = urlparse(url)
        if parsed.scheme not in {"ws", "wss"}:
            raise ValueError("URL must start with ws:// or wss://")
        self.parsed = parsed
        self.timeout = timeout
        self.sock: socket.socket | ssl.SSLSocket | None = None

    def connect(self) -> None:
        host = self.parsed.hostname or "127.0.0.1"
        port = self.parsed.port or (443 if self.parsed.scheme == "wss" else 80)
        path = self.parsed.path or "/"
        if self.parsed.query:
            path += "?" + self.parsed.query

        raw_sock = socket.create_connection((host, port), timeout=self.timeout)
        raw_sock.settimeout(self.timeout)
        if self.parsed.scheme == "wss":
            sock: socket.socket | ssl.SSLSocket = ssl.create_default_context().wrap_socket(
                raw_sock, server_hostname=host
            )
        else:
            sock = raw_sock

        key = base64.b64encode(os.urandom(16)).decode("ascii")
        request = (
            f"GET {path} HTTP/1.1\r\n"
            f"Host: {host}:{port}\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Key: {key}\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            "\r\n"
        )
        sock.sendall(request.encode("ascii"))
        response = self._read_http_response(sock)
        if b" 101 " not in response.split(b"\r\n", 1)[0]:
            raise RuntimeError(response.decode("latin1", errors="replace"))
        expected = base64.b64encode(hashlib.sha1((key + GUID).encode("ascii")).digest())
        if expected.lower() not in response.lower():
            raise RuntimeError("WebSocket handshake response did not include expected accept key")
        self.sock = sock

    @staticmethod
    def _read_http_response(sock: socket.socket | ssl.SSLSocket) -> bytes:
        data = b""
        while b"\r\n\r\n" not in data:
            chunk = sock.recv(4096)
            if not chunk:
                break
            data += chunk
            if len(data) > 64 * 1024:
                raise RuntimeError("HTTP upgrade response too large")
        return data

    def send_text(self, text: str) -> None:
        self._send_frame(0x1, text.encode("utf-8"))

    def close(self) -> None:
        try:
            self._send_frame(0x8, b"")
        finally:
            if self.sock is not None:
                self.sock.close()
                self.sock = None

    def _send_frame(self, opcode: int, payload: bytes) -> None:
        if self.sock is None:
            raise RuntimeError("WebSocket is not connected")
        # Clients must mask frames.
        mask = os.urandom(4)
        header = bytearray([0x80 | opcode])
        length = len(payload)
        if length < 126:
            header.append(0x80 | length)
        elif length < (1 << 16):
            header.extend([0x80 | 126])
            header.extend(struct.pack("!H", length))
        else:
            header.extend([0x80 | 127])
            header.extend(struct.pack("!Q", length))
        masked = bytes(byte ^ mask[i % 4] for i, byte in enumerate(payload))
        self.sock.sendall(bytes(header) + mask + masked)

    def recv_frame(self) -> Frame:
        if self.sock is None:
            raise RuntimeError("WebSocket is not connected")
        first = self._read_exact(2)
        b0, b1 = first
        opcode = b0 & 0x0F
        masked = bool(b1 & 0x80)
        length = b1 & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._read_exact(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._read_exact(8))[0]
        mask = self._read_exact(4) if masked else b""
        payload = self._read_exact(length)
        if masked:
            payload = bytes(byte ^ mask[i % 4] for i, byte in enumerate(payload))
        if opcode == 0x9:  # ping
            self._send_frame(0xA, payload)
            return self.recv_frame()
        return Frame(opcode=opcode, payload=payload)

    def _read_exact(self, n: int) -> bytes:
        if self.sock is None:
            raise RuntimeError("WebSocket is not connected")
        data = b""
        while len(data) < n:
            chunk = self.sock.recv(n - len(data))
            if not chunk:
                raise EOFError("WebSocket closed while reading")
            data += chunk
        return data


def float32le_to_s16le(chunk: bytes) -> bytes:
    if len(chunk) % 4 != 0:
        raise ValueError(f"PCM f32 chunk size is not divisible by 4: {len(chunk)}")
    out = bytearray(len(chunk) // 2)
    for i, (sample,) in enumerate(struct.iter_unpack("<f", chunk)):
        sample = max(-1.0, min(1.0, sample))
        struct.pack_into("<h", out, i * 2, int(sample * 32767.0))
    return bytes(out)


def write_wav(path: str, pcm_f32le_path: str, sample_rate: int) -> None:
    with open(pcm_f32le_path, "rb") as src, wave.open(path, "wb") as dst:
        dst.setnchannels(1)
        dst.setsampwidth(2)
        dst.setframerate(sample_rate)
        while True:
            chunk = src.read(64 * 1024)
            if not chunk:
                break
            dst.writeframes(float32le_to_s16le(chunk))


def run(args: argparse.Namespace) -> int:
    req = {
        "text": args.text,
        "speaker": args.speaker,
        "seed": args.seed,
    }
    if args.instruction:
        req["instruction"] = args.instruction

    client = WebSocketClient(args.url, args.timeout)
    started = time.monotonic()
    client.connect()
    connected = time.monotonic()
    client.send_text(json.dumps(req, ensure_ascii=False))

    first_binary_at: float | None = None
    segment_done_at: float | None = None
    chunk_count = 0
    total_bytes = 0
    text_messages: list[str] = []

    with open(args.output_pcm, "wb") as pcm:
        while True:
            frame = client.recv_frame()
            now = time.monotonic()
            if frame.opcode == 0x1:  # text
                message = frame.payload.decode("utf-8", errors="replace")
                text_messages.append(message)
                if message == "segment_done":
                    segment_done_at = now
                    break
                if message.startswith('{"error"'):
                    raise RuntimeError(message)
            elif frame.opcode == 0x2:  # binary f32le PCM
                if first_binary_at is None:
                    first_binary_at = now
                chunk_count += 1
                total_bytes += len(frame.payload)
                pcm.write(frame.payload)
            elif frame.opcode == 0x8:  # close
                break

    client.send_text("end")
    client.close()

    finished = time.monotonic()
    total_samples = total_bytes // 4
    audio_seconds = total_samples / SAMPLE_RATE
    generation_seconds = (segment_done_at or finished) - connected
    ttfb_seconds = None if first_binary_at is None else first_binary_at - connected
    rtf = None if audio_seconds <= 0 else generation_seconds / audio_seconds

    if args.output_wav:
        write_wav(args.output_wav, args.output_pcm, SAMPLE_RATE)

    result = {
        "url": args.url,
        "speaker": args.speaker,
        "text_chars": len(args.text),
        "connect_seconds": round(connected - started, 3),
        "ttfb_seconds": None if ttfb_seconds is None else round(ttfb_seconds, 3),
        "generation_seconds": round(generation_seconds, 3),
        "audio_seconds": round(audio_seconds, 3),
        "rtf": None if rtf is None else round(rtf, 3),
        "chunk_count": chunk_count,
        "total_samples": total_samples,
        "output_pcm": args.output_pcm,
        "output_wav": args.output_wav,
        "text_messages": text_messages,
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if chunk_count > 0 and total_samples > 0 else 1


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Smoke-test Qwen3-TTS WebSocket streaming")
    parser.add_argument("--url", default="ws://127.0.0.1:9746/api/tts/stream")
    parser.add_argument("--text", default="你好，这是本地流式语音合成测试。")
    parser.add_argument("--speaker", default="vivian")
    parser.add_argument("--seed", type=int, default=123)
    parser.add_argument("--instruction")
    parser.add_argument("--timeout", type=float, default=120.0)
    parser.add_argument("--output-pcm", default="/tmp/qwen3-tts-stream.f32le.pcm")
    parser.add_argument("--output-wav", default="/tmp/qwen3-tts-stream.wav")
    return parser.parse_args()


if __name__ == "__main__":
    try:
        raise SystemExit(run(parse_args()))
    except KeyboardInterrupt:
        raise SystemExit(130)
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
