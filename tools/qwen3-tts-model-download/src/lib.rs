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

#[cfg(test)]
mod tests {
    use super::*;

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
}
