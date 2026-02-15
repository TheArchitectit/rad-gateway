# Protocol Stack Decision (A2A, AG-UI, MCP, ACP, ANP)

## Decision Summary

- Implement now: A2A + AG-UI.
- Implement in scoped form: MCP (tools/resources bridge only).
- Defer: ACP + ANP.

This is the near-term recommendation for Brass Relay after parity baseline work.

## Why This Split

### A2A (adopt now)

- Strong interop fit for agent-to-agent workflows.
- Clear task lifecycle and streaming model.
- Active ecosystem and maintained SDKs (including Go).

Primary use in Brass Relay:

- discovery via Agent Card
- task send, subscribe, status, cancel
- cross-agent delegation over HTTP JSON-RPC/SSE

## AG-UI (adopt now)

- Purpose-built frontend/backend protocol for agent UX.
- Event taxonomy maps to real-time UI needs (run lifecycle, text/tool/state deltas).
- Complements A2A by handling user-agent interaction rather than agent-agent transport.

Primary use in Brass Relay:

- stream lifecycle/tool/state events to clients
- maintain session replay and event compaction surfaces

## MCP (adopt selectively)

- Strong fit for tool and resource exposure to models/agents.
- Not a replacement for agent orchestration or UI protocol layers.

Primary use in Brass Relay:

- tool/resource bridge with strict auth, token audience checks, and scope boundaries
- no ownership of routing policy, model/provider selection, or multi-agent orchestration

## ACP (defer)

- ACP repository is archived and includes migration guidance toward A2A direction.
- Implementing ACP now adds integration cost with limited upside versus direct A2A adoption.

Decision: watch only, no production dependency.

## ANP (defer)

- ANP has an ambitious architecture and active project activity, but near-term implementation risk is still higher for this product phase.
- Current Brass Relay roadmap benefits more from mature, directly deployable A2A + AG-UI + scoped MCP.

Decision: monitor maturity and revisit when SDK/runtime integration path is lower risk.

## Product Mapping

- Agent-to-agent transport: A2A
- User-facing agent protocol: AG-UI
- Tool/resource access: MCP
- Legacy/alternative protocol tracks: ACP, ANP (watchlist)

## Sources Used

- A2A repository and spec: `https://github.com/a2aproject/A2A`, `https://a2a-protocol.org/latest/specification/`
- AG-UI docs: `https://docs.ag-ui.com/introduction`, `https://docs.ag-ui.com/concepts/events`, `https://docs.ag-ui.com/agentic-protocols`
- MCP spec/docs: `https://modelcontextprotocol.io/latest/specification/`, `https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices`
- ACP repository and migration notice: `https://github.com/i-am-bee/acp`
- ANP repository: `https://github.com/agent-network-protocol/AgentNetworkProtocol`
