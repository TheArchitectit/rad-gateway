# A2A Gateway 2026 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Build a production-ready A2A API Gateway on 2026 open standards supporting Google A2A Protocol, MCP, SPIFFE identity, Envoy-based L7 mediation, and Cedar authorization.

**Architecture:** Two-tier data plane (eBPF L4 + Envoy Wasm L7), Kubernetes Gateway API control plane, SPIRE workload identity, Kafka async eventing, OpenTelemetry observability.

**Tech Stack:** Kubernetes Gateway API v1.4, Envoy Proxy, Cilium eBPF, SPIFFE/SPIRE, Rust (Wasm filters), Cedar, Apache Kafka, OpenTelemetry, Go (backend).

---

## Phase 2: Gateway Configuration (Days 4-6) - CONTINUED

### Task 2.3: Configure BackendTrafficPolicy and SecurityPolicy

**Files:**
- Create: `k8s/gateway/backend-traffic-policy.yaml`
- Create: `k8s/gateway/security-policy.yaml`
- Create: `k8s/gateway/reference-grant.yaml`

**Step 1: Write BackendTrafficPolicy for circuit breaking**

```yaml
# k8s/gateway/backend-traffic-policy.yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: a2a-agent-health-policy
  namespace: a2a-agents
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: a2a-protocol-route
  circuitBreaker:
    maxConnections: 1000
    maxPendingRequests: 100
    maxRequests: 1000
    maxRetries: 3
    maxConnectionPools: 100
  healthCheck:
    active:
      type: HTTP
      path: /health
      interval: 10s
      timeout: 5s
      unhealthyThreshold: 3
      healthyThreshold: 1
    passive:
      baseEjectionTime: 30s
      maxEjectionPercent: 50
      consecutive5xxErrors: 5
      interval: 10s
```

**Step 2: Write SecurityPolicy for OAuth 2.1 and CORS**

```yaml
# k8s/gateway/security-policy.yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: SecurityPolicy
metadata:
  name: a2a-security-policy
  namespace: gateway-system
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: a2a-gateway
  cors:
    allowOrigins:
      - "https://a2a.internal.corp"
      - "https://agents.internal.corp"
    allowMethods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
    allowHeaders:
      - Authorization
      - Content-Type
      - X-Request-ID
      - X-A2A-Protocol-Version
      - X-SPIFFE-ID
    allowCredentials: true
    maxAge: "86400s"

  authorization:
    type: JWT
    jwt:
      providers:
        - name: oauth-provider
          issuer: "https://auth.internal.corp"
          audiences:
            - "a2a-gateway"
          remoteJWKS:
            uri: "https://auth.internal.corp/.well-known/jwks.json"
            timeout: "5s"
          claimToHeaders:
            - claim: sub
              header: X-User-ID
            - claim: spiffe_id
              header: X-SPIFFE-ID
```

**Step 3: Write ReferenceGrant for cross-namespace routing**

```yaml
# k8s/gateway/reference-grant.yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: allow-gateway-to-agents
  namespace: a2a-agents
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      namespace: gateway-system
    - group: gateway.envoyproxy.io
      kind: BackendTrafficPolicy
      namespace: gateway-system
    - group: gateway.envoyproxy.io
      kind: SecurityPolicy
      namespace: gateway-system
  to:
    - group: ""
      kind: Service
      name: radgateway-backend
    - group: ""
      kind: Secret
      name: agent-credentials
```

**Step 4: Apply and verify policies**

```bash
kubectl apply -f k8s/gateway/backend-traffic-policy.yaml
kubectl apply -f k8s/gateway/security-policy.yaml
kubectl apply -f k8s/gateway/reference-grant.yaml
kubectl get backendtrafficpolicy a2a-agent-health-policy
kubectl get securitypolicy a2a-security-policy
kubectl get referencegrant allow-gateway-to-agents
```
Expected: All policies show "Accepted" condition.

**Step 5: Commit**

```bash
git add k8s/gateway/*.yaml
git commit -m "gateway: add BackendTrafficPolicy, SecurityPolicy, and ReferenceGrant"
```

---

### Task 2.4: Configure Envoy SDS for SPIFFE Certificate Rotation

**Files:**
- Create: `k8s/envoy/sds-config.yaml`
- Create: `k8s/envoy/envoy-configmap.yaml`

**Step 1: Write Envoy SDS configuration**

```yaml
# k8s/envoy/sds-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-sds-config
  namespace: gateway-system
data:
  sds.yaml: |
    resources:
      - "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret
        name: spiffe://internal.corp
        tls_certificate:
          certificate_chain:
            filename: /run/spire/sockets/agent.sock
          private_key:
            filename: /run/spire/sockets/agent.sock

      - "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret
        name: validation_context
        validation_context:
          trusted_ca:
            filename: /run/spire/bundle/bundle.pem
          match_typed_subject_alt_names:
            - san_type: URI
              matcher:
                exact: "spiffe://internal.corp"
```

**Step 2: Write Envoy bootstrap configuration**

```yaml
# k8s/envoy/envoy-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-bootstrap-config
  namespace: gateway-system
data:
  envoy.yaml: |
    node:
      id: a2a-gateway-node
      cluster: a2a-gateway-cluster

    static_resources:
      listeners:
        - name: a2a_listener
          address:
            socket_address:
              address: 0.0.0.0
              port_value: 8443
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: a2a_ingress
                    codec_type: AUTO
                    route_config:
                      name: local_route
                      virtual_hosts:
                        - name: a2a_backend
                          domains:
                            - "*"
                          routes:
                            - match:
                                prefix: "/"
                              route:
                                cluster: radgateway_backend
                    http_filters:
                      - name: envoy.filters.http.wasm
                        typed_config:
                          "@type": type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                          config:
                            name: a2a_filter
                            root_id: a2a_filter_root
                            vm_config:
                              vm_id: a2a_filter_vm
                              runtime: envoy.wasm.runtime.v8
                              code:
                                remote:
                                  http_uri:
                                    uri: https://ghcr.io/radgateway/a2a-wasm-filter:latest
                                    cluster: wasm_cluster
                                    timeout: 10s
                      - name: envoy.filters.http.router
                        typed_config:
                          "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router

      clusters:
        - name: radgateway_backend
          connect_timeout: 30s
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: radgateway_backend
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address:
                          address: radgateway-backend.a2a-agents.svc.cluster.local
                          port_value: 8090
          transport_socket:
            name: envoy.transport_sockets.tls
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
              common_tls_context:
                tls_certificate_sds_secret_configs:
                  - name: spiffe://internal.corp
                    sds_config:
                      path: /etc/envoy/sds.yaml
                validation_context_sds_secret_config:
                  name: validation_context
                  sds_config:
                    path: /etc/envoy/sds.yaml
```

**Step 3: Commit**

```bash
git add k8s/envoy/
git commit -m "envoy: add SDS config for SPIFFE certificate rotation"
```

---

## Phase 3: Wasm Filter Development (Days 7-12)

### Task 3.1: Set Up Rust Wasm Filter Project

**Files:**
- Create: `filters/a2a-wasm/Cargo.toml`
- Create: `filters/a2a-wasm/.cargo/config.toml`
- Create: `filters/a2a-wasm/rust-toolchain.toml`

**Step 1: Write Cargo.toml with dependencies**

```toml
# filters/a2a-wasm/Cargo.toml
[package]
name = "a2a-wasm-filter"
version = "0.1.0"
edition = "2021"
authors = ["RAD Gateway Team"]
description = "Envoy Wasm filter for A2A protocol validation"

[lib]
crate-type = ["cdylib"]

[dependencies]
proxy-wasm = "0.2"
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
log = "0.4"
anyhow = "1.0"
thiserror = "1.0"
chrono = { version = "0.4", default-features = false, features = ["wasmbind"] }
regex = { version = "1.10", default-features = false, features = ["std"] }
sha2 = { version = "0.10", default-features = false }
hex = { version = "0.4", default-features = false }

[dev-dependencies]
wasm-bindgen-test = "0.3"

[profile.release]
opt-level = 3
lto = true
panic = "abort"
strip = true
```

**Step 2: Write Rust toolchain configuration**

```toml
# filters/a2a-wasm/.cargo/config.toml
[build]
target = "wasm32-unknown-unknown"

[unstable]
bindeps = true
```

```toml
# filters/a2a-wasm/rust-toolchain.toml
[toolchain]
channel = "stable"
targets = ["wasm32-unknown-unknown"]
components = ["rustfmt", "clippy"]
```

**Step 3: Initialize project and verify build**

```bash
cd filters/a2a-wasm
cargo check
```
Expected: Successful dependency resolution.

**Step 4: Commit**

```bash
git add filters/a2a-wasm/
git commit -m "wasm: initialize Rust project with proxy-wasm SDK"
```

---

### Task 3.2: Implement A2A Protocol Validation Filter

**Files:**
- Create: `filters/a2a-wasm/src/lib.rs`
- Create: `filters/a2a-wasm/src/a2a.rs`
- Create: `filters/a2a-wasm/src/validation.rs`

**Step 1: Write main filter entry point**

```rust
// filters/a2a-wasm/src/lib.rs
use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use serde::{Deserialize, Serialize};
use std::time::Duration;

mod a2a;
mod validation;

use a2a::A2ARequest;
use validation::validate_a2a_payload;

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

struct A2AFilter {
    context_id: u32,
    config: FilterConfig,
}

impl Context for A2AFilter {
    fn on_http_call_response(
        &mut self,
        _token_id: u32,
        _num_headers: usize,
        _body_size: usize,
        _num_trailers: usize,
    ) {
        // Handle async callbacks if needed
    }
}

impl HttpContext for A2AFilter {
    fn on_http_request_headers(&mut self, num_headers: usize, end_of_stream: bool) -> Action {
        let headers = self.get_http_request_headers();

        // Log incoming request
        log::info!("A2A request received: {:?}", headers);

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

        if let Some(body) = self.get_http_request_body(0, body_size) {
            match std::str::from_utf8(&body) {
                Ok(body_str) => {
                    match validate_a2a_payload(body_str) {
                        Ok(result) => {
                            // Add observability headers
                            self.set_http_request_header(
                                "x-a2a-validated",
                                Some("true")
                            );
                            self.set_http_request_header(
                                "x-estimated-tokens",
                                Some(&result.estimated_tokens.to_string())
                            );

                            log::info!(
                                "A2A payload validated: {} tokens",
                                result.estimated_tokens
                            );
                        }
                        Err(e) => {
                            log::error!("A2A validation failed: {}", e);
                            self.send_http_response(
                                400,
                                vec![("content-type", "application/json")],
                                Some(format!(
                                    r#"{{"error": "invalid_a2a_payload", "details": "{}"}}"#,
                                    e
                                ).as_bytes()),
                            );
                            return Action::Pause;
                        }
                    }
                }
                Err(e) => {
                    log::error!("Invalid UTF-8 in request body: {}", e);
                }
            }
        }

        Action::Continue
    }

    fn on_http_response_headers(&mut self, num_headers: usize, end_of_stream: bool) -> Action {
        // Add gateway headers to response
        self.set_http_response_header(
            "x-a2a-gateway-version",
            Some("0.1.0")
        );

        Action::Continue
    }
}
```

**Step 2: Write A2A protocol structures**

```rust
// filters/a2a-wasm/src/a2a.rs
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct A2ARequest {
    #[serde(rename = "task_id")]
    pub task_id: String,
    #[serde(rename = "message_object")]
    pub message_object: MessageObject,
    pub capabilities: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MessageObject {
    pub role: String,
    pub parts: Vec<MessagePart>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(tag = "type")]
pub enum MessagePart {
    #[serde(rename = "text")]
    Text { text: String },
    #[serde(rename = "file")]
    File { name: String, mime_type: String, uri: Option<String> },
    #[serde(rename = "function_call")]
    FunctionCall { id: String, name: String, args: serde_json::Value },
    #[serde(rename = "function_response")]
    FunctionResponse { id: String, response: serde_json::Value },
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AgentCard {
    pub name: String,
    pub description: String,
    pub url: String,
    pub version: String,
    pub capabilities: AgentCapabilities,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub authentication: Option<AuthenticationInfo>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AgentCapabilities {
    pub streaming: bool,
    #[serde(rename = "pushNotifications")]
    pub push_notifications: bool,
    #[serde(rename = "stateTransition")]
    pub state_transition: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AuthenticationInfo {
    pub schemes: Vec<String>,
}
```

**Step 3: Write validation logic with token counting**

```rust
// filters/a2a-wasm/src/validation.rs
use crate::a2a::A2ARequest;
use serde_json::Value;

#[derive(Debug)]
pub struct ValidationResult {
    pub valid: bool,
    pub estimated_tokens: u64,
    pub errors: Vec<String>,
}

pub fn validate_a2a_payload(body: &str) -> Result<ValidationResult, String> {
    let request: A2ARequest = serde_json::from_str(body)
        .map_err(|e| format!("JSON parse error: {}", e))?;

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

/// Estimate token count using a simple approximation
/// ~4 characters per token on average for English text
fn estimate_tokens(request: &A2ARequest) -> u64 {
    let mut chars = 0u64;

    // Count characters in task_id
    chars += request.task_id.len() as u64;

    // Count characters in message parts
    for part in &request.message_object.parts {
        chars += serde_json::to_string(part).unwrap_or_default().len() as u64;
    }

    // Estimate tokens (4 chars per token)
    (chars + 3) / 4
}

/// Token bucket rate limiting calculation
pub fn check_token_bucket(
    current_tokens: f64,
    max_capacity: f64,
    replenish_rate: f64,
    elapsed_seconds: f64,
    estimated_tokens: f64,
) -> (bool, f64) {
    let available = (current_tokens + replenish_rate * elapsed_seconds)
        .min(max_capacity);

    if available >= estimated_tokens {
        (true, available - estimated_tokens)
    } else {
        (false, available)
    }
}

/// Dynamic trust score calculation
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
        assert_eq!(remaining, 60.0);
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
    }
}
```

**Step 4: Build and verify**

```bash
cd filters/a2a-wasm
cargo build --target wasm32-unknown-unknown --release
cargo test
```
Expected: Build succeeds, tests pass.

**Step 5: Commit**

```bash
git add filters/a2a-wasm/src/
git commit -m "wasm: implement A2A protocol validation with token counting"
```

---

### Task 3.3: Build and Containerize Wasm Filter

**Files:**
- Create: `filters/a2a-wasm/Dockerfile`
- Create: `filters/a2a-wasm/Makefile`

**Step 1: Write Dockerfile for Wasm build**

```dockerfile
# filters/a2a-wasm/Dockerfile
FROM rust:1.75-slim-bookworm as builder

RUN apt-get update && apt-get install -y \
    llvm-15-dev \
    libclang-15-dev \
    clang-15 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY Cargo.toml rust-toolchain.toml ./
COPY .cargo/ ./.cargo/
RUN cargo fetch

COPY src/ ./src/
RUN cargo build --target wasm32-unknown-unknown --release

# Verify the wasm module
RUN ls -la /build/target/wasm32-unknown-unknown/release/*.wasm

# Production stage - just the wasm file
FROM scratch
COPY --from=builder /build/target/wasm32-unknown-unknown/release/a2a_wasm_filter.wasm /filter.wasm
```

**Step 2: Write Makefile**

```makefile
# filters/a2a-wasm/Makefile
.PHONY: build test clean image push

FILTER_NAME := a2a-wasm-filter
REGISTRY := ghcr.io/radgateway
TAG := latest

build:
	cargo build --target wasm32-unknown-unknown --release
	cp target/wasm32-unknown-unknown/release/a2a_wasm_filter.wasm dist/

test:
	cargo test

lint:
	cargo clippy --target wasm32-unknown-unknown -- -D warnings
	cargo fmt -- --check

clean:
	cargo clean
	rm -rf dist/

image:
	docker build -t $(REGISTRY)/$(FILTER_NAME):$(TAG) .

push: image
	docker push $(REGISTRY)/$(FILTER_NAME):$(TAG)

all: lint test build
```

**Step 3: Create dist directory and build**

```bash
mkdir -p filters/a2a-wasm/dist
make -C filters/a2a-wasm all
```
Expected: Lint clean, tests pass, wasm file generated.

**Step 4: Commit**

```bash
git add filters/a2a-wasm/Dockerfile filters/a2a-wasm/Makefile
git commit -m "wasm: add build automation and containerization"
```

---

## Phase 4: Cedar Policy Generation (Days 13-15)

### Task 4.1: Create Cedar Policy Schema

**Files:**
- Create: `policies/cedar/schema.cedarschema`
- Create: `policies/cedar/entities.json`

**Step 1: Write Cedar schema**

```cedar
// policies/cedar/schema.cedarschema
namespace A2A {
    type Agent = {
        id: String,
        trustDomain: String,
        capabilities: Set<String>,
    };

    type Task = {
        id: String,
        owner: Agent,
        status: String,
        jurisdiction: String,
    };

    type Action = {
        type: String,
        resource: String,
    };

    entity Agent = {
        spiffeId: String,
        capabilities: Set<String>,
        trustScore: Decimal,
    };

    entity Task in Agent;
    entity Action;
}

// Action types
action submit_task, cancel_task, view_task, stream_events, invoke_capability;
```

**Step 2: Write Cedar policies**

```cedar
// policies/cedar/agent-authz.cedar
// Default deny
forbid (principal, action, resource);

// Allow agents to submit tasks within their trust domain
permit (
    principal == Agent::"any",
    action == Action::"submit_task",
    resource
) when {
    principal.trustScore > 0.65 &&
    resource.jurisdiction in principal.allowedJurisdictions
};

// Allow agents to cancel their own tasks
permit (
    principal == Agent::"any",
    action == Action::"cancel_task",
    resource == Task::"any"
) when {
    resource.owner == principal
};

// Allow viewing tasks within same workspace
permit (
    principal == Agent::"any",
    action == Action::"view_task",
    resource == Task::"any"
) when {
    resource.workspace == principal.workspace
};

// Allow streaming if agent has streaming capability
permit (
    principal == Agent::"any",
    action == Action::"stream_events",
    resource
) when {
    "streaming" in principal.capabilities &&
    principal.trustScore > 0.8
};

// Forbid blocked agents
forbid (
    principal == Agent::"any",
    action,
    resource
) when {
    principal.trustScore < 0.3
};

// Sovereign data compliance
forbid (
    principal == Agent::"any",
    action,
    resource
) when {
    resource.jurisdiction == "EU" &&
    principal.jurisdiction != "EU"
};
```

**Step 3: Write entity definitions**

```json
// policies/cedar/entities.json
{
  "entities": [
    {
      "uid": {
        "type": "A2A::Agent",
        "id": "logistics-optimizer"
      },
      "attrs": {
        "spiffeId": "spiffe://internal.corp/agent/logistics-optimizer",
        "capabilities": ["streaming", "pushNotifications", "stateManagement"],
        "trustScore": 0.95,
        "workspace": "logistics-team",
        "allowedJurisdictions": ["US", "EU", "APAC"],
        "jurisdiction": "US"
      },
      "parents": []
    },
    {
      "uid": {
        "type": "A2A::Agent",
        "id": "finance-analyzer"
      },
      "attrs": {
        "spiffeId": "spiffe://internal.corp/agent/finance-analyzer",
        "capabilities": ["streaming"],
        "trustScore": 0.88,
        "workspace": "finance-team",
        "allowedJurisdictions": ["US"],
        "jurisdiction": "US"
      },
      "parents": []
    }
  ]
}
```

**Step 4: Commit**

```bash
git add policies/cedar/
git commit -m "policies: add Cedar authorization schema and rules"
```

---

### Task 4.2: Create Policy Decision Point (PDP)

**Files:**
- Create: `internal/auth/cedar/pdp.go`
- Create: `internal/auth/cedar/pdp_test.go`

**Step 1: Write Cedar PDP Go implementation**

```go
// internal/auth/cedar/pdp.go
package cedar

import (
    "context"
    "fmt"
    "os"

    "github.com/cedar-policy/cedar-go"
)

// PolicyDecisionPoint evaluates authorization requests
type PolicyDecisionPoint struct {
    policySet *cedar.PolicySet
    schema    *cedar.Schema
}

// NewPDP creates a new policy decision point
func NewPDP(policyPath string) (*PolicyDecisionPoint, error) {
    policyBytes, err := os.ReadFile(policyPath)
    if err != nil {
        return nil, fmt.Errorf("reading policy file: %w", err)
    }

    policySet, err := cedar.ParsePolicies(string(policyBytes))
    if err != nil {
        return nil, fmt.Errorf("parsing policies: %w", err)
    }

    return &PolicyDecisionPoint{
        policySet: policySet,
    }, nil
}

// AuthorizationRequest represents a request to authorize
type AuthorizationRequest struct {
    Principal string            `json:"principal"`
    Action    string            `json:"action"`
    Resource  string            `json:"resource"`
    Context   map[string]any    `json:"context,omitempty"`
}

// AuthorizationDecision represents the result
type AuthorizationDecision struct {
    Decision string   `json:"decision"` // "Allow" or "Deny"
    Reasons  []string `json:"reasons,omitempty"`
}

// Authorize evaluates a request against policies
func (p *PolicyDecisionPoint) Authorize(
    ctx context.Context,
    req AuthorizationRequest,
) (*AuthorizationDecision, error) {
    // Convert to Cedar entities
    principal := cedar.EntityUID{
        Type: "A2A::Agent",
        ID:   cedar.String(req.Principal),
    }

    action := cedar.EntityUID{
        Type: "Action",
        ID:   cedar.String(req.Action),
    }

    resource := cedar.EntityUID{
        Type: "A2A::Task",
        ID:   cedar.String(req.Resource),
    }

    // Build context
    context := cedar.NewRecord(cedar.RecordMap{})
    for k, v := range req.Context {
        context.Set(cedar.String(k), cedar.String(fmt.Sprintf("%v", v)))
    }

    // Evaluate
    result, err := p.policySet.IsAuthorized(
        principal,
        action,
        resource,
        cedar.Context{context},
    )
    if err != nil {
        return nil, fmt.Errorf("policy evaluation error: %w", err)
    }

    decision := "Deny"
    if result.Decision == cedar.Allow {
        decision = "Allow"
    }

    return &AuthorizationDecision{
        Decision: decision,
        Reasons:  result.Reasons,
    }, nil
}
```

**Step 2: Write tests**

```go
// internal/auth/cedar/pdp_test.go
package cedar

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPDP_Authorize(t *testing.T) {
    pdp, err := NewPDP("../../../policies/cedar/agent-authz.cedar")
    require.NoError(t, err)

    ctx := context.Background()

    tests := []struct {
        name       string
        req        AuthorizationRequest
        wantDecision string
    }{
        {
            name: "trusted agent can submit task",
            req: AuthorizationRequest{
                Principal: "logistics-optimizer",
                Action:    "submit_task",
                Resource:  "task-123",
                Context: map[string]any{
                    "jurisdiction": "US",
                },
            },
            wantDecision: "Allow",
        },
        {
            name: "untrusted agent denied",
            req: AuthorizationRequest{
                Principal: "untrusted-agent",
                Action:    "submit_task",
                Resource:  "task-123",
            },
            wantDecision: "Deny",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := pdp.Authorize(ctx, tt.req)
            require.NoError(t, err)
            assert.Equal(t, tt.wantDecision, result.Decision)
        })
    }
}
```

**Step 3: Run tests**

```bash
go test ./internal/auth/cedar/ -v
```
Expected: Tests pass.

**Step 4: Commit**

```bash
git add internal/auth/cedar/
git commit -m "auth: add Cedar policy decision point (PDP)"
```

---

## Phase 5: State Management with Kafka (Days 16-18)

### Task 5.1: Deploy Apache Kafka (Diskless Mode)

**Files:**
- Create: `k8s/kafka/kafka-cluster.yaml`
- Create: `k8s/kafka/topic-a2a-events.yaml`

**Step 1: Write Kafka cluster configuration**

```yaml
# k8s/kafka/kafka-cluster.yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: a2a-kafka
  namespace: kafka
spec:
  kafka:
    version: 3.6.0
    replicas: 3
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: tls
        port: 9093
        type: internal
        tls: true
    authorization:
      type: simple
    config:
      offsets.topic.replication.factor: 3
      transaction.state.log.replication.factor: 3
      transaction.state.log.min.isr: 2
      default.replication.factor: 3
      min.insync.replicas: 2
      log.retention.hours: 168
      log.segment.bytes: 1073741824
      log.retention.check.interval.ms: 300000
    storage:
      type: ephemeral  # Diskless mode
  zookeeper:
    replicas: 3
    storage:
      type: ephemeral  # Diskless mode
  entityOperator:
    topicOperator: {}
    userOperator: {}
```

**Step 2: Write A2A event topics**

```yaml
# k8s/kafka/topic-a2a-events.yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: a2a-task-events
  namespace: kafka
  labels:
    strimzi.io/cluster: a2a-kafka
spec:
  partitions: 12
  replicas: 3
  config:
    retention.ms: 604800000  # 7 days
    cleanup.policy: delete
---
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: a2a-webhook-callbacks
  namespace: kafka
  labels:
    strimzi.io/cluster: a2a-kafka
spec:
  partitions: 6
  replicas: 3
  config:
    retention.ms: 86400000  # 1 day
    cleanup.policy: delete
---
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: a2a-agent-discovery
  namespace: kafka
  labels:
    strimzi.io/cluster: a2a-kafka
spec:
  partitions: 3
  replicas: 3
  config:
    retention.ms: 3600000  # 1 hour
    cleanup.policy: compact  # Compact for discovery cache
```

**Step 3: Commit**

```bash
git add k8s/kafka/
git commit -m "kafka: add diskless cluster and A2A event topics"
```

---

### Task 5.2: Implement Kafka Producer/Consumer

**Files:**
- Create: `internal/messaging/kafka/producer.go`
- Create: `internal/messaging/kafka/consumer.go`
- Create: `internal/messaging/kafka/types.go`

**Step 1: Write Kafka types**

```go
// internal/messaging/kafka/types.go
package kafka

import (
    "encoding/json"
    "time"
)

// TaskEvent represents an A2A task event
type TaskEvent struct {
    EventID   string          `json:"event_id"`
    TaskID    string          `json:"task_id"`
    EventType string          `json:"event_type"` // "created", "updated", "completed", "failed"
    Status    string          `json:"status"`
    AgentID   string          `json:"agent_id"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   json.RawMessage `json:"payload,omitempty"`
}

// WebhookCallback represents an async webhook notification
type WebhookCallback struct {
    CallbackID string          `json:"callback_id"`
    TaskID     string          `json:"task_id"`
    WebhookURL string          `json:"webhook_url"`
    Payload    json.RawMessage `json:"payload"`
    RetryCount int             `json:"retry_count"`
    MaxRetries int             `json:"max_retries"`
    CreatedAt  time.Time       `json:"created_at"`
}

// AgentDiscoveryEvent represents agent registration/deregistration
type AgentDiscoveryEvent struct {
    EventID   string    `json:"event_id"`
    AgentID   string    `json:"agent_id"`
    Action    string    `json:"action"` // "registered", "deregistered", "updated"
    AgentCard []byte    `json:"agent_card"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Step 2: Write Kafka producer**

```go
// internal/messaging/kafka/producer.go
package kafka

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/IBM/sarama"
)

// Producer wraps sarama async producer
type Producer struct {
    producer sarama.AsyncProducer
    brokers  []string
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string) (*Producer, error) {
    config := sarama.NewConfig()
    config.Producer.Return.Successes = true
    config.Producer.Return.Errors = true
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Retry.Max = 3

    producer, err := sarama.NewAsyncProducer(brokers, config)
    if err != nil {
        return nil, fmt.Errorf("creating producer: %w", err)
    }

    return &Producer{
        producer: producer,
        brokers:  brokers,
    }, nil
}

// SendTaskEvent sends a task event to Kafka
func (p *Producer) SendTaskEvent(ctx context.Context, event TaskEvent) error {
    msg, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshaling event: %w", err)
    }

    p.producer.Input() <- &sarama.ProducerMessage{
        Topic: "a2a-task-events",
        Key:   sarama.StringEncoder(event.TaskID),
        Value: sarama.ByteEncoder(msg),
        Headers: []sarama.RecordHeader{
            {Key: []byte("event-type"), Value: []byte(event.EventType)},
            {Key: []byte("agent-id"), Value: []byte(event.AgentID)},
        },
    }

    return nil
}

// SendWebhookCallback sends a webhook callback to the queue
func (p *Producer) SendWebhookCallback(ctx context.Context, callback WebhookCallback) error {
    msg, err := json.Marshal(callback)
    if err != nil {
        return fmt.Errorf("marshaling callback: %w", err)
    }

    p.producer.Input() <- &sarama.ProducerMessage{
        Topic: "a2a-webhook-callbacks",
        Key:   sarama.StringEncoder(callback.CallbackID),
        Value: sarama.ByteEncoder(msg),
    }

    return nil
}

// Close shuts down the producer
func (p *Producer) Close() error {
    return p.producer.Close()
}
```

**Step 3: Write Kafka consumer**

```go
// internal/messaging/kafka/consumer.go
package kafka

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"

    "github.com/IBM/sarama"
)

// HandlerFunc processes consumed messages
type HandlerFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// Consumer wraps sarama consumer group
type Consumer struct {
    group   sarama.ConsumerGroup
    topics  []string
    handler HandlerFunc
}

// NewConsumer creates a new consumer group
func NewConsumer(brokers []string, groupID string, topics []string) (*Consumer, error) {
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetOldest

    group, err := sarama.NewConsumerGroup(brokers, groupID, config)
    if err != nil {
        return nil, fmt.Errorf("creating consumer group: %w", err)
    }

    return &Consumer{
        group:  group,
        topics: topics,
    }, nil
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context, handler HandlerFunc) error {
    c.handler = handler

    for {
        if err := c.group.Consume(ctx, c.topics, c); err != nil {
            return fmt.Errorf("consume error: %w", err)
        }

        if ctx.Err() != nil {
            return ctx.Err()
        }
    }
}

// Close shuts down the consumer
func (c *Consumer) Close() error {
    return c.group.Close()
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerHandler struct {
    handler HandlerFunc
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(
    session sarama.ConsumerGroupSession,
    claim sarama.ConsumerGroupClaim,
) error {
    ctx := context.Background()

    for msg := range claim.Messages() {
        slog.Info("received message",
            "topic", msg.Topic,
            "partition", msg.Partition,
            "offset", msg.Offset,
        )

        if err := c.handler(ctx, msg); err != nil {
            slog.Error("handler error", "error", err)
            // Don't commit on error - will retry
            continue
        }

        session.MarkMessage(msg, "")
    }

    return nil
}

// ProcessWebhookCallback handles webhook delivery
func ProcessWebhookCallback(ctx context.Context, msg *sarama.ConsumerMessage) error {
    var callback WebhookCallback
    if err := json.Unmarshal(msg.Value, &callback); err != nil {
        return fmt.Errorf("unmarshaling callback: %w", err)
    }

    // Implement webhook delivery logic
    slog.Info("processing webhook",
        "callback_id", callback.CallbackID,
        "task_id", callback.TaskID,
        "url", callback.WebhookURL,
    )

    // TODO: Implement actual HTTP POST to webhookURL

    return nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/messaging/kafka/ -v
```
Expected: Tests pass (or skipped if no Kafka running).

**Step 5: Commit**

```bash
git add internal/messaging/kafka/
git commit -m "messaging: add Kafka producer/consumer for A2A events"
```

---

## Phase 6: OpenTelemetry Observability (Days 19-20)

### Task 6.1: Configure OTel Collector and Instrumentation

**Files:**
- Create: `k8s/otel/otel-collector.yaml`
- Create: `internal/observability/tracer.go`

**Step 1: Write OTel collector configuration**

```yaml
# k8s/otel/otel-collector.yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: a2a-otel
  namespace: observability
spec:
  mode: deployment
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318

    processors:
      batch:
        timeout: 1s
        send_batch_size: 1024
      resource:
        attributes:
          - key: service.name
            value: a2a-gateway
            action: upsert
          - key: deployment.environment
            value: production
            action: upsert

    exporters:
      prometheus:
        endpoint: 0.0.0.0:8889
      otlp/jaeger:
        endpoint: jaeger-collector.observability.svc.cluster.local:4317
        tls:
          insecure: true

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [resource, batch]
          exporters: [otlp/jaeger]
        metrics:
          receivers: [otlp]
          processors: [resource, batch]
          exporters: [prometheus]
```

**Step 2: Write Go tracer initialization**

```go
// internal/observability/tracer.go
package observability

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "go.opentelemetry.io/otel/trace"
)

// InitTracer initializes OpenTelemetry tracer
func InitTracer(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
    ctx := context.Background()

    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("creating exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String("0.1.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("creating resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}

// Tracer returns the global tracer
func Tracer() trace.Tracer {
    return otel.Tracer("a2a-gateway")
}

// AgentAttributes creates OTel attributes for agent context
func AgentAttributes(agentID, taskID string, promptTokens, completionTokens int) []attribute.KeyValue {
    return []attribute.KeyValue{
        attribute.String("agent.id", agentID),
        attribute.String("task.id", taskID),
        attribute.Int("prompt.tokens", promptTokens),
        attribute.Int("completion.tokens", completionTokens),
    }
}
```

**Step 3: Commit**

```bash
git add k8s/otel/ internal/observability/
git commit -m "observability: add OpenTelemetry collector and Go instrumentation"
```

---

## Phase 7: Integration and Testing (Days 21)

### Task 7.1: End-to-End Integration Tests

**Files:**
- Create: `tests/e2e/a2a_gateway_test.go`
- Create: `tests/e2e/setup_test.go`

**Step 1: Write E2E test suite**

```go
// tests/e2e/a2a_gateway_test.go
package e2e

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestA2A_Gateway_FullFlow(t *testing.T) {
    ctx := context.Background()

    // Setup: Create SPIFFE identity
    agentID, err := createTestAgentIdentity("logistics-optimizer")
    require.NoError(t, err)
    defer cleanupAgentIdentity(agentID)

    t.Run("agent_discovery", func(t *testing.T) {
        // Request agent card
        resp, err := http.Get("https://a2a.internal.corp/.well-known/agent.json")
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 200, resp.StatusCode)
        assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
        assert.Contains(t, resp.Header.Get("Cache-Control"), "max-age=300")
    })

    t.Run("synchronous_task", func(t *testing.T) {
        task := map[string]any{
            "task_id": "task-sync-" + time.Now().Format("20060102150405"),
            "message_object": map[string]any{
                "role": "user",
                "parts": []map[string]any{
                    {"type": "text", "text": "What is the current temperature?"},
                },
            },
            "capabilities": []string{"a2a", "streaming"},
        }

        body, _ := json.Marshal(task)
        req, _ := http.NewRequestWithContext(ctx, "POST",
            "https://a2a.internal.corp/a2a/tasks",
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/agent-task+json")
        req.Header.Set("X-SPIFFE-ID", agentID)

        client := &http.Client{Timeout: 30 * time.Second}
        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 200, resp.StatusCode)
        assert.NotEmpty(t, resp.Header.Get("X-A2A-Validated"))
        assert.NotEmpty(t, resp.Header.Get("X-Estimated-Tokens"))
    })

    t.Run("asynchronous_task_with_webhook", func(t *testing.T) {
        task := map[string]any{
            "task_id": "task-async-" + time.Now().Format("20060102150405"),
            "message_object": map[string]any{
                "role": "user",
                "parts": []map[string]any{
                    {"type": "text", "text": "Analyze this 500-page document"},
                },
            },
            "capabilities": []string{"a2a"},
            "webhook": map[string]string{
                "url": "https://callback.example.com/webhook",
            },
        }

        body, _ := json.Marshal(task)
        req, _ := http.NewRequestWithContext(ctx, "POST",
            "https://a2a.internal.corp/a2a/tasks",
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/agent-task+json")
        req.Header.Set("X-SPIFFE-ID", agentID)

        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 202, resp.StatusCode)

        var result map[string]any
        json.NewDecoder(resp.Body).Decode(&result)
        assert.NotEmpty(t, result["task_id"])
    })

    t.Run("sovereign_data_compliance", func(t *testing.T) {
        // EU agent attempting to access US resource
        task := map[string]any{
            "task_id": "task-eu-" + time.Now().Format("20060102150405"),
            "message_object": map[string]any{
                "role": "user",
                "parts": []map[string]any{
                    {"type": "text", "text": "Process PII data"},
                },
            },
            "metadata": map[string]string{
                "jurisdiction": "EU",
            },
        }

        body, _ := json.Marshal(task)
        req, _ := http.NewRequestWithContext(ctx, "POST",
            "https://a2a.internal.corp/a2a/tasks",
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/agent-task+json")
        req.Header.Set("X-SPIFFE-ID", agentID) // US agent

        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Should be denied by Cedar policy
        assert.Equal(t, 403, resp.StatusCode)
    })

    t.Run("trust_decay_blocking", func(t *testing.T) {
        // Create low-trust agent
        lowTrustAgentID, _ := createLowTrustAgent("suspicious-agent")

        task := map[string]any{
            "task_id": "task-sus-" + time.Now().Format("20060102150405"),
            "message_object": map[string]any{
                "role": "user",
                "parts": []map[string]any{
                    {"type": "text", "text": "Hello"},
                },
            },
        }

        body, _ := json.Marshal(task)
        req, _ := http.NewRequestWithContext(ctx, "POST",
            "https://a2a.internal.corp/a2a/tasks",
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/agent-task+json")
        req.Header.Set("X-SPIFFE-ID", lowTrustAgentID)

        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Low trust score should result in 403
        assert.Equal(t, 403, resp.StatusCode)
    })

    t.Run("token_rate_limiting", func(t *testing.T) {
        // Request with excessive token estimate
        task := map[string]any{
            "task_id": "task-limit-" + time.Now().Format("20060102150405"),
            "message_object": map[string]any{
                "role": "user",
                "parts": []map[string]any{
                    {"type": "text", "text": string(make([]byte, 1000000))}, // Large payload
                },
            },
        }

        body, _ := json.Marshal(task)
        req, _ := http.NewRequestWithContext(ctx, "POST",
            "https://a2a.internal.corp/a2a/tasks",
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/agent-task+json")
        req.Header.Set("X-SPIFFE-ID", agentID)

        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        // Should be rate limited
        assert.Equal(t, 429, resp.StatusCode)
    })
}
```

**Step 2: Write test setup**

```go
// tests/e2e/setup_test.go
package e2e

import (
    "fmt"
    "os"
    "testing"
)

var testConfig struct {
    GatewayURL string
    SPIREAddr  string
}

func TestMain(m *testing.M) {
    testConfig.GatewayURL = getEnv("A2A_GATEWAY_URL", "https://a2a.internal.corp")
    testConfig.SPIREAddr = getEnv("SPIRE_SERVER_ADDR", "spire-server.spire:8081")

    // Run tests
    code := m.Run()
    os.Exit(code)
}

func getEnv(key, defaultValue string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultValue
}

func createTestAgentIdentity(name string) (string, error) {
    // Mock implementation - integrate with SPIRE SDK
    return fmt.Sprintf("spiffe://internal.corp/agent/%s", name), nil
}

func cleanupAgentIdentity(agentID string) error {
    // Mock implementation
    return nil
}

func createLowTrustAgent(name string) (string, error) {
    // Create agent with trust score below threshold
    return fmt.Sprintf("spiffe://internal.corp/agent/%s", name), nil
}
```

**Step 3: Commit**

```bash
git add tests/e2e/
git commit -m "e2e: add comprehensive A2A gateway integration tests"
```

---

## Daily Execution Commands

### Morning Routine
```bash
git pull origin feature/a2a-gateway-2026
cd /mnt/ollama/git/RADAPI01
make check
```

### End of Day
```bash
make test
make lint
git add .
git commit -m "feat: [description of day's work]"
git push origin feature/a2a-gateway-2026
```

---

## Success Criteria

| Phase | Criteria |
|-------|----------|
| **Phase 1** | Kubernetes cluster with Gateway API, Cilium eBPF, SPIRE installed |
| **Phase 2** | Gateway routes, HTTPRoutes, policies configured and validated |
| **Phase 3** | Wasm filter builds, validates A2A payloads, counts tokens |
| **Phase 4** | Cedar policies evaluate authorization correctly |
| **Phase 5** | Kafka cluster running, producer/consumer working |
| **Phase 6** | OTel traces visible in Jaeger, metrics in Prometheus |
| **Phase 7** | All E2E tests pass |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| SPIFFE mTLS complexity | Test with mock identities before full SPIRE integration |
| Wasm filter performance | Benchmark token counting, optimize if >1ms latency |
| Kafka diskless stability | Monitor pod restarts, have persistent fallback |
| Cedar policy complexity | Start with simple allow/deny, add conditions incrementally |
| eBPF compatibility | Test on target kernel versions (5.15+) |

---

**Plan complete and saved to `docs/plans/a2a-gateway-2026-implementation.md`.**

**Two execution options:**

1. **Subagent-Driven (this session)** - Dispatch fresh subagent per task, review between tasks, fast iteration
2. **Parallel Session (separate)** - Open new session with `superpowers:executing-plans`, batch execution with checkpoints

**Which approach?**