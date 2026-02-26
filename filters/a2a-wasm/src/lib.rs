// A2A Wasm Filter for Envoy Proxy
// Main entry point for the proxy-wasm filter

use proxy_wasm::traits::*;
use proxy_wasm::types::*;

mod a2a;
mod validation;

use a2a::A2ARequest;
use validation::{calculate_trust_score, check_token_bucket, estimate_tokens, validate_a2a_payload};

#[no_mangle]
pub fn _start() {
    proxy_wasm::set_log_level(LogLevel::Info);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(A2AFilterRoot)
    });
}

struct A2AFilterRoot;

impl Context for A2AFilterRoot {}

impl RootContext for A2AFilterRoot {
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }

    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(A2AFilter {
            context_id,
            config: FilterConfig::default(),
        }))
    }
}

#[derive(Debug, Default)]
struct FilterConfig {
    max_tokens_per_request: u64,
    trust_decay_constant: f64,
    min_trust_score: f64,
}

impl FilterConfig {
    fn new() -> Self {
        FilterConfig {
            max_tokens_per_request: 100_000,
            trust_decay_constant: 0.1,
            min_trust_score: 0.65,
        }
    }
}

struct A2AFilter {
    context_id: u32,
    config: FilterConfig,
}

impl Context for A2AFilter {}

impl HttpContext for A2AFilter {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Log incoming request
        if let Some(path) = self.get_http_request_header(":path") {
            log::info!("A2A request received: {}", path);
        }

        // Check for A2A protocol headers
        let content_type = self.get_http_request_header("content-type");
        let accept = self.get_http_request_header("accept");

        if let Some(ct) = content_type {
            if ct == "application/agent-task+json" {
                log::info!("Detected A2A task submission");
                return Action::Continue;
            }
        }

        if let Some(acc) = accept {
            if acc == "text/event-stream" {
                log::info!("Detected A2A streaming request");
            }
        }

        Action::Continue
    }

    fn on_http_request_body(&mut self, body_size: usize, end_of_stream: bool) -> Action {
        if !end_of_stream {
            return Action::Pause;
        }

        // Get the request body
        let body_bytes = match self.get_http_request_body(0, body_size) {
            Some(bytes) => bytes,
            None => {
                log::error!("Failed to get request body");
                return Action::Continue;
            }
        };

        let body_str = match std::str::from_utf8(&body_bytes) {
            Ok(s) => s,
            Err(e) => {
                log::error!("Invalid UTF-8 in request body: {}", e);
                self.send_error_response(400, "Invalid UTF-8 encoding");
                return Action::Pause;
            }
        };

        // Validate A2A payload
        match validate_a2a_payload(body_str) {
            Ok(result) => {
                if !result.valid {
                    log::error!("A2A validation failed: {:?}", result.errors);
                    let error_json = format!(
                        r#"{{"error": "invalid_a2a_payload", "details": {:?}}"#,
                        result.errors
                    );
                    self.send_error_response_with_body(400, &error_json);
                    return Action::Pause;
                }

                // Check token limit
                if result.estimated_tokens > self.config.max_tokens_per_request {
                    log::warn!(
                        "Token limit exceeded: {} > {}",
                        result.estimated_tokens,
                        self.config.max_tokens_per_request
                    );
                    self.send_error_response(429, "Token limit exceeded");
                    return Action::Pause;
                }

                // Add observability headers
                self.set_http_request_header("x-a2a-validated", Some("true"));
                self.set_http_request_header(
                    "x-estimated-tokens",
                    Some(&result.estimated_tokens.to_string()),
                );

                log::info!("A2A payload validated: {} tokens", result.estimated_tokens);
            }
            Err(e) => {
                log::error!("A2A validation error: {}", e);
                self.send_error_response(400, &e);
                return Action::Pause;
            }
        }

        // Check trust score from SPIFFE ID header
        if let Some(spiffe_id) = self.get_http_request_header("x-spiffe-id") {
            log::debug!("SPIFFE ID: {}", spiffe_id);
            // Trust score would be checked against a shared state
            // For now, just log it
        }

        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Add gateway headers to response
        self.set_http_response_header("x-a2a-gateway-version", Some("0.1.0"));
        self.set_http_response_header("x-served-by", Some("a2a-wasm-filter"));
        Action::Continue
    }
}

impl A2AFilter {
    fn send_error_response(&self, status: u32, message: &str) {
        let body = format!(r#"{{"error": "{}"}}"#, message);
        self.send_error_response_with_body(status, &body);
    }

    fn send_error_response_with_body(&self, status: u32, body: &str) {
        self.send_http_response(
            status,
            vec![
                ("content-type", "application/json"),
                ("x-a2a-error", "true"),
            ],
            Some(body.as_bytes()),
        );
    }
}
