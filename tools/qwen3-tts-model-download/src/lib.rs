use std::fs;
use std::io;
use std::path::Path;

pub const SUPPORTED_QUANTS: &[&str] = &["none", "q5_k_m", "q8_0"];

pub fn validate_quant(quant: &str) -> Result<(), String> {
    if SUPPORTED_QUANTS.contains(&quant) {
        Ok(())
    } else {
        Err(format!(
            "unsupported quant '{quant}'. supported values: {}",
            SUPPORTED_QUANTS.join(", ")
        ))
    }
}

pub fn quant_model_dir(quant: &str) -> &'static str {
    match quant {
        "q5_k_m" => "gguf_q5_k_m",
        "q8_0" => "gguf_q8_0",
        _ => "gguf",
    }
}

/// Keep upstream's expected model layout without patching upstream download.rs.
///
/// Upstream currently downloads `qwen3_assets.gguf` into the selected quant
/// directory, but `TtsEngine::new` always loads assets from `gguf/` because
/// assets must stay F32. The stack-level download service normalizes that
/// layout after calling upstream download code.
pub fn normalize_assets_layout(model_dir: &Path, quant: &str) -> io::Result<bool> {
    if quant_model_dir(quant) == "gguf" {
        return Ok(false);
    }

    let target = model_dir.join("gguf").join("qwen3_assets.gguf");
    if target.exists() {
        return Ok(false);
    }

    let source = model_dir
        .join(quant_model_dir(quant))
        .join("qwen3_assets.gguf");
    if !source.exists() {
        return Ok(false);
    }

    if let Some(parent) = target.parent() {
        fs::create_dir_all(parent)?;
    }

    match fs::hard_link(&source, &target) {
        Ok(()) => Ok(true),
        Err(_) => {
            fs::copy(&source, &target)?;
            Ok(true)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn accepts_supported_quants() {
        for quant in SUPPORTED_QUANTS {
            assert!(validate_quant(quant).is_ok(), "{quant} should be valid");
        }
    }

    #[test]
    fn rejects_unknown_quant() {
        let err = validate_quant("q4_0").expect_err("q4_0 should be rejected");
        assert!(err.contains("q4_0"));
        assert!(err.contains("q5_k_m"));
    }

    #[test]
    fn maps_quant_to_upstream_model_dirs() {
        assert_eq!(quant_model_dir("none"), "gguf");
        assert_eq!(quant_model_dir("q5_k_m"), "gguf_q5_k_m");
        assert_eq!(quant_model_dir("q8_0"), "gguf_q8_0");
    }

    #[test]
    fn normalizes_quantized_assets_to_unquantized_gguf_dir() {
        let root = std::env::temp_dir().join(format!(
            "qwen3-tts-model-download-{}",
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let source_dir = root.join("gguf_q5_k_m");
        fs::create_dir_all(&source_dir).unwrap();
        fs::write(source_dir.join("qwen3_assets.gguf"), b"assets").unwrap();

        let changed = normalize_assets_layout(&root, "q5_k_m").unwrap();
        assert!(changed);
        assert_eq!(fs::read(root.join("gguf/qwen3_assets.gguf")).unwrap(), b"assets");

        let changed = normalize_assets_layout(&root, "q5_k_m").unwrap();
        assert!(!changed, "second run should be idempotent");

        fs::remove_dir_all(root).unwrap();
    }
}
