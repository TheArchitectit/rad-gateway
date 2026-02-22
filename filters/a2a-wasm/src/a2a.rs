// A2A Protocol Data Structures
// Defines the JSON schema for A2A request/response payloads

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// A2ARequest represents an incoming agent-to-agent task submission
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

/// MessageObject contains the agent message content
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MessageObject {
    pub role: String,
    pub parts: Vec<MessagePart>,
}

/// MessagePart is a polymorphic type for different message content types
#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(tag = "type")]
pub enum MessagePart {
    #[serde(rename = "text")]
    Text { text: String },

    #[serde(rename = "file")]
    File {
        name: String,
        #[serde(rename = "mimeType")]
        mime_type: String,
        uri: Option<String>,
    },

    #[serde(rename = "data")]
    Data { data: HashMap<String, serde_json::Value> },

    #[serde(rename = "function_call")]
    FunctionCall {
        id: String,
        name: String,
        args: HashMap<String, serde_json::Value>,
    },

    #[serde(rename = "function_response")]
    FunctionResponse {
        #[serde(rename = "id")]
        call_id: String,
        response: HashMap<String, serde_json::Value>,
    },
}

/// AgentCard represents the agent capabilities document
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AgentCard {
    pub name: String,
    pub description: String,
    pub url: String,
    pub version: String,
    pub capabilities: AgentCapabilities,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub authentication: Option<AuthenticationInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub skills: Option<Vec<Skill>>,
}

/// AgentCapabilities defines what the agent supports
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AgentCapabilities {
    pub streaming: bool,
    #[serde(rename = "pushNotifications")]
    pub push_notifications: bool,
    #[serde(rename = "stateTransition")]
    pub state_transition: bool,
}

/// AuthenticationInfo describes supported auth schemes
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AuthenticationInfo {
    pub schemes: Vec<String>,
}

/// Skill represents a capability exposed by the agent
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Skill {
    pub id: String,
    pub name: String,
    pub description: String,
    pub tags: Option<Vec<String>>,
    pub examples: Option<Vec<String>>,
}

impl A2ARequest {
    /// Validate the request has required fields
    pub fn validate(&self) -> Result<(), ValidationError> {
        if self.task_id.is_empty() {
            return Err(ValidationError::MissingTaskId);
        }

        if self.message_object.parts.is_empty() {
            return Err(ValidationError::EmptyMessageParts);
        }

        for cap in &self.capabilities {
            if !Self::is_valid_capability(cap) {
                return Err(ValidationError::InvalidCapability(cap.clone()));
            }
        }

        Ok(())
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
}

#[derive(Debug, thiserror::Error)]
pub enum ValidationError {
    #[error("task_id is required")]
    MissingTaskId,
    #[error("message_object.parts cannot be empty")]
    EmptyMessageParts,
    #[error("invalid capability: {0}")]
    InvalidCapability(String),
    #[error("invalid JSON: {0}")]
    InvalidJson(String),
}
