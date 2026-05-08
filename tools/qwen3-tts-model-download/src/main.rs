use clap::Parser;
use qwen3_tts::TtsEngine;
use qwen3_tts_model_download::{normalize_assets_layout, validate_quant};
use std::path::PathBuf;

#[derive(Parser, Debug)]
#[command(author, version, about = "Download Qwen3-TTS model files", long_about = None)]
struct Args {
    #[arg(long, default_value = "models")]
    model_dir: PathBuf,

    #[arg(long, default_value = "q5_k_m")]
    quant: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();

    validate_quant(&args.quant)?;

    println!("=== Qwen3-TTS Model Download ===");
    println!("Model Dir: {:?}", args.model_dir);
    println!("Quant:     {}", args.quant);
    println!("Checking and downloading model files...");

    TtsEngine::download_models(&args.model_dir, &args.quant)
        .await
        .map_err(|e| format!("Model download failed: {}", e))?;

    if normalize_assets_layout(&args.model_dir, &args.quant)? {
        println!(
            "Normalized qwen3_assets.gguf into {:?}",
            args.model_dir.join("gguf")
        );
    }

    println!("Model download completed. Model files are ready.");
    Ok(())
}
