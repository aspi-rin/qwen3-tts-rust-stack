#!/usr/bin/env python3
import argparse, json, os, subprocess, sys, time, wave
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_TEXTS = [
    "你好，我是本地语音合成服务。",
    "今天天气不错，我们来测试一下中文语音合成的自然度和实时性。",
    "Qwen three TTS Rust 使用 GGUF 和 llama.cpp backend，希望在 Vulkan 上获得更低的 RTF。",
]

def wav_duration(path: Path) -> float:
    with wave.open(str(path), "rb") as w:
        return w.getnframes() / float(w.getframerate())

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--rounds", type=int, default=3)
    ap.add_argument("--speaker", default=os.environ.get("SPEAKER", "vivian"))
    ap.add_argument("--quant", default=os.environ.get("QUANT", "q5_k_m"))
    ap.add_argument("--threads", default=os.environ.get("THREADS", "4"))
    ap.add_argument("--text", action="append", help="Override/add test text; can be repeated")
    ap.add_argument("--json-out", default=str(ROOT / "data/outputs/benchmark.json"))
    args = ap.parse_args()

    texts = args.text or DEFAULT_TEXTS
    out_dir = ROOT / "data/outputs"
    out_dir.mkdir(parents=True, exist_ok=True)

    subprocess.run([str(ROOT / "scripts/build_cli.sh")], check=True)

    results = []
    for i in range(args.rounds):
        text = texts[i % len(texts)]
        out = out_dir / f"bench_{i+1:02d}.wav"
        cmd = [
            "cargo", "run", "--manifest-path", str(ROOT / "upstream/Cargo.toml"),
            "--release", "--features", os.environ.get("FEATURES", "vulkan"),
            "--bin", "qwen3_tts", "--",
            "--model-dir", str(ROOT / "data/models"),
            "--quant", args.quant,
            "--threads", args.threads,
            "--speakers-dir", str(ROOT / "upstream/speakers"),
            "--speaker", args.speaker,
            "--text", text,
            "--output", str(out),
        ]
        start = time.perf_counter()
        proc = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        elapsed = time.perf_counter() - start
        if proc.returncode != 0:
            print(proc.stdout)
            raise SystemExit(proc.returncode)
        dur = wav_duration(out)
        rtf = elapsed / dur if dur > 0 else float("inf")
        row = {"round": i+1, "text": text, "output": str(out), "elapsed_sec": elapsed, "audio_sec": dur, "rtf": rtf}
        results.append(row)
        print(f"round={i+1} elapsed={elapsed:.3f}s audio={dur:.3f}s rtf={rtf:.3f} output={out}")

    avg = sum(r["rtf"] for r in results) / len(results)
    payload = {"avg_rtf": avg, "results": results}
    Path(args.json_out).write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"avg_rtf={avg:.3f}")
    print(f"json={args.json_out}")

if __name__ == "__main__":
    main()
