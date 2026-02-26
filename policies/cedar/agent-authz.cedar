// RAD Gateway A2A - Agent Authorization Policies
// Cedar policies for agent-to-agent access control

// =============================================================================
// DEFAULT DENY - All requests denied unless explicitly permitted
// =============================================================================
forbid (principal, action, resource);

// =============================================================================
// TRUST-BASED ACCESS
// Agents must have minimum trust score to perform any action
// =============================================================================

// Permit trusted agents (trust > 0.65) to submit tasks
permit (
    principal == A2A::Agent,
    action == A2A::Action::submit_task,
    resource == A2A::Task
) when {
    principal.trustScore > 0.65 &&
    resource.workspace == principal.workspace
};

// Permit agents to cancel their own tasks
permit (
    principal == A2A::Agent,
    action == A2A::Action::cancel_task,
    resource == A2A::Task
) when {
    resource.owner == principal &&
    principal.trustScore > 0.5
};

// Permit agents to view tasks in their workspace
permit (
    principal == A2A::Agent,
    action == A2A::Action::view_task,
    resource == A2A::Task
) when {
    resource.workspace == principal.workspace &&
    principal.trustScore > 0.5
};

// =============================================================================
// STREAMING ACCESS
// Requires higher trust score and streaming capability
// =============================================================================

permit (
    principal == A2A::Agent,
    action == A2A::Action::stream_events,
    resource == A2A::Task
) when {
    "streaming" in principal.capabilities &&
    principal.trustScore > 0.8 &&
    resource.workspace == principal.workspace
};

// =============================================================================
// SOVEREIGN DATA COMPLIANCE
// EU data can only be accessed by EU-based agents
// =============================================================================

forbid (
    principal == A2A::Agent,
    action,
    resource == A2A::Task
) when {
    resource.jurisdiction == "EU" &&
    principal.jurisdiction != "EU"
};

// APAC data residency
forbid (
    principal == A2A::Agent,
    action,
    resource == A2A::Task
) when {
    resource.jurisdiction == "APAC" &&
    !(principal.jurisdiction == "APAC" || "APAC" in principal.allowedJurisdictions)
};

// =============================================================================
// CAPABILITY-BASED ACCESS
// Agents can only invoke capabilities they possess
// =============================================================================

permit (
    principal == A2A::Agent,
    action == A2A::Action::invoke_capability,
    resource == A2A::Agent
) when {
    // Agent must have the capability
    resource.capabilities.contains("invoke") &&
    // And trust score must be sufficient
    principal.trustScore > 0.7
};

// =============================================================================
// BLOCKED AGENTS
// Agents with very low trust scores are completely blocked
// =============================================================================

forbid (
    principal == A2A::Agent,
    action,
    resource
) when {
    principal.trustScore < 0.3
};

// =============================================================================
// WORKSPACE ISOLATION
// Agents can only access resources in their own workspace
// =============================================================================

permit (
    principal == A2A::Agent,
    action == A2A::Action::access_resource,
    resource == A2A::Workspace
) when {
    resource.id == principal.workspace
};
