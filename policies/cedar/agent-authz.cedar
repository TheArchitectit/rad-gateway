// RAD Gateway A2A - Agent Authorization Policies
// Cedar policies for agent-to-agent access control

// =============================================================================
// DEFAULT DENY - All requests denied unless explicitly permitted
// =============================================================================
forbid (principal, action, resource);

// =============================================================================
// PERMIT ALL API ACCESS (Simple Policy)
// Allows any authenticated principal to invoke API endpoints
// =============================================================================
permit (
    principal,
    action == Action::"invoke",
    resource
);

// =============================================================================
// WORKSPACE ISOLATION
// Agents can only access resources in their own workspace
// =============================================================================
permit (
    principal,
    action == Action::"access_resource",
    resource
) when {
    resource.workspace == principal.workspace
};
