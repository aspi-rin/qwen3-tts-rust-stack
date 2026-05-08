use clap::Parser;
use qwen3_tts::TtsEngine;
use qwen3_tts_prepare::validate_quant;
use std::path::PathBuf;

#[derive(Parser, Debug)]
#[command(author, version, about = "Prepare Qwen3-TTS model files", long_about = None)]
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

    println!("=== Qwen3-TTS Prepare ===");
    println!("Model Dir: {:?}", args.model_dir);
    println!("Quant:     {}", args.quant);
    println!("Checking and downloading model files...");

    TtsEngine::download_models(&args.model_dir, &args.quant)
        .await
        .map_err(|e| format!("Model preparation failed: {}", e))?;

    println!("Model files are ready.");
    Ok(())
}
