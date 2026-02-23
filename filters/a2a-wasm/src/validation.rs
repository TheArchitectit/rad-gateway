// A2A Payload Validation and Token Counting
// Implements token estimation and rate limiting logic

use crate::a2a::{A2ARequest, ValidationError};
use serde::{Deserialize, Serialize};

/// ValidationResult contains the outcome of payload validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub valid: bool,
    pub estimated_tokens: u64,
    pub errors: Vec<String>,
}

/// Validate an A2A JSON payload and estimate token count
pub fn validate_a2a_payload(body: &str) -> Result<ValidationResult, String> {
    // Parse the request
    let request: A2ARequest = serde_json::from_str(body)
        .map_err(|e| ValidationError::InvalidJson(e.to_string()))?;

    let mut errors = Vec::new();

    // Validate required fields
    if request.task_id.is_empty() {
        errors.push("task_id is required".to_string());
    }

    if request.message_object.parts.is_empty() {
        errors.push("message_object.parts cannot be empty".to_string());
    }

    // Validate capabilities
    for cap in &request.capabilities {
        if !is_valid_capability(cap) {
            errors.push(format!("invalid capability: {}", cap));
        }
    }

    // Estimate token count
    let estimated_tokens = estimate_tokens(&request);

    Ok(ValidationResult {
        valid: errors.is_empty(),
        estimated_tokens,
        errors,
    })
}

fn is_valid_capability(cap: &str) -> bool {
    const VALID_CAPS: &[&str] = &[
        "a2a",
        "streaming",
        "pushNotifications",
        "stateManagement",
        "artifactSupport",
    ];
    VALID_CAPS.contains(&cap)
}

/// Estimate token count using character-based approximation
/// ~4 characters per token on average for English text
pub fn estimate_tokens(request: &A2ARequest) -> u64 {
    let mut chars = 0u64;

    // Count characters in task_id
    chars += request.task_id.len() as u64;

    // Count characters in message parts
    for part in &request.message_object.parts {
        chars += count_part_characters(part);
    }

    // Estimate tokens (4 chars per token)
    (chars + 3) / 4
}

fn count_part_characters(part: &crate::a2a::MessagePart) -> u64 {
    match part {
        crate::a2a::MessagePart::Text { text } => text.len() as u64,
        crate::a2a::MessagePart::File { name, mime_type, .. } => {
            name.len() as u64 + mime_type.len() as u64
        }
        crate::a2a::MessagePart::Data { data } => {
            serde_json::to_string(data).map(|s| s.len() as u64).unwrap_or(0)
        }
        crate::a2a::MessagePart::FunctionCall { name, args, .. } => {
            name.len() as u64
                + serde_json::to_string(args).map(|s| s.len() as u64).unwrap_or(0)
        }
        crate::a2a::MessagePart::FunctionResponse { call_id, response } => {
            call_id.len() as u64
                + serde_json::to_string(response).map(|s| s.len() as u64).unwrap_or(0)
        }
    }
}

/// Token bucket rate limiting calculation
/// Returns (allowed, remaining_tokens)
pub fn check_token_bucket(
    current_tokens: f64,
    max_capacity: f64,
    replenish_rate: f64,
    elapsed_seconds: f64,
    estimated_tokens: f64,
) -> (bool, f64) {
    let available = (current_tokens + replenish_rate * elapsed_seconds).min(max_capacity);

    if available >= estimated_tokens {
        (true, available - estimated_tokens)
    } else {
        (false, available)
    }
}

/// Dynamic trust score calculation with exponential decay
pub fn calculate_trust_score(
    initial_score: f64,
    decay_constant: f64,
    violations: u32,
) -> f64 {
    initial_score * (-decay_constant * violations as f64).exp()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_token_bucket_allows_valid_request() {
        let (allowed, remaining) = check_token_bucket(100.0, 1000.0, 10.0, 1.0, 50.0);
        assert!(allowed);
        assert!((remaining - 60.0).abs() < 0.001);
    }

    #[test]
    fn test_token_bucket_denies_excessive_request() {
        let (allowed, _) = check_token_bucket(10.0, 1000.0, 1.0, 1.0, 500.0);
        assert!(!allowed);
    }

    #[test]
    fn test_trust_score_decay() {
        let score = calculate_trust_score(1.0, 0.1, 5);
        assert!(score < 1.0);
        assert!(score > 0.0);
        // e^(-0.5) ≈ 0.606
        assert!((score - 0.606).abs() < 0.01);
    }

    #[test]
    fn test_estimate_tokens() {
        let request = A2ARequest {
            task_id: "task-123".to_string(),
            message_object: crate::a2a::MessageObject {
                role: "user".to_string(),
                parts: vec![crate::a2a::MessagePart::Text {
                    text: "Hello, world!".to_string(),
                }],
            },
            capabilities: vec!["a2a".to_string()],
            metadata: None,
        };

        let tokens = estimate_tokens(&request);
        // "task-123" (8) + "Hello, world!" (13) = 21 chars ≈ 5-6 tokens
        assert!(tokens > 3 && tokens < 10);
    }
}
