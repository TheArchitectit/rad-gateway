# RAD Gateway - Team System Guide

**Version**: 1.0
**Date**: 2026-02-16
**Status**: Active

---

## Team Structure (TEAM-007 Compliant)

All teams have 4-6 members per guardrails TEAM-007.

| Team | Purpose | Members | Status |
|------|---------|---------|--------|
| Team Alpha | Architecture & Design | 6 | 🟢 Active |
| Team Bravo | Core Implementation | 6 | 🟢 Active |
| Team Charlie | Security Hardening | 5 | 🟢 Active |
| Team Delta | Quality Assurance | 5 | 🟢 Active |
| Team Echo | Operations & Observability | 5 | 🟢 Active |
| Team Foxtrot | Inspiration Analysis | 5 | ✅ Complete |
| Team Golf | Documentation & Design | 6 | 🟢 Active |
| Team Hotel | Deployment & Infrastructure | 5 | 🟢 Deploying |

**Total**: 43 members across 8 teams

---

## Team Management Commands

### Create a New Team

```bash
# Create team (must not be leading another team)
TeamCreate --team_name <name> --description <desc> --agent_type <type>

# Example:
TeamCreate --team_name "hotel-deployment" --description "Team Hotel - Deployment" --agent_type "devops-engineer"
```

### Spawn Team Members

```bash
# Spawn individual team members as separate agents
Task --subagent_type <type> --team_name <team> --name <member-name> --prompt <instructions>

# Example - Team Hotel members:
Task --subagent_type "devops-engineer" --team_name "hotel-deployment" --name "devops-lead" --prompt "..."
Task --subagent_type "devops-engineer" --team_name "hotel-deployment" --name "container-engineer" --prompt "..."
Task --subagent_type "devops-engineer" --team_name "hotel-deployment" --name "deployment-engineer" --prompt "..."
Task --subagent_type "devops-engineer" --team_name "hotel-deployment" --name "infrastructure-architect" --prompt "..."
Task --subagent_type "devops-engineer" --team_name "hotel-deployment" --name "systems-administrator" --prompt "..."
```

### Monitor Team Tasks

```bash
# List all tasks for current team
TaskList

# Get specific task details
TaskGet --taskId <id>

# Update task status
TaskUpdate --taskId <id> --status "completed"
```

### Delete Team (When Done)

```bash
# Must shut down all team members first
TeamDelete
```

---

## Team Hotel - Current Deployment

### Members

| Member | Role | Current Task | Status |
|--------|------|--------------|--------|
| devops-lead | DevOps Lead | Verify deployment success | in_progress |
| container-engineer | Container Engineer | Verify container health | in_progress |
| deployment-engineer | Deployment Engineer | Review deployment scripts | in_progress |
| infrastructure-architect | Infrastructure Architect | Validate infrastructure | in_progress |
| systems-administrator | Systems Administrator | Check system health | in_progress |

### Deployment Status

**radgateway01 on 172.16.30.45**:
- ✅ Container Image: Built
- ✅ Podman Pod: Created
- ✅ Container: Running
- ✅ Systemd Service: Active
- ✅ Firewall: Port 8090 open
- ✅ Health: Responding ({"status":"ok"})

---

## Communication Protocol

### Team Lead Responsibilities
1. Create team with TeamCreate
2. Spawn team members with Task tool
3. Monitor progress via TaskList
4. Collect final reports from members
5. Delete team when complete

### Team Member Responsibilities
1. Accept task assignment
2. Execute assigned work
3. Report status to team lead
4. Complete task via TaskUpdate

### Inter-Team Communication
- Use SendMessage tool for direct messages
- Use broadcast sparingly for team-wide announcements
- Document decisions in shared memory files

---

## Memory Files

Team information is persisted in:

- `/home/user001/.claude/projects/-mnt-ollama-git-RADAPI01/memory/MEMORY.md` - Main memory
- `/home/user001/.claude/projects/-mnt-ollama-git-RADAPI01/memory/teams.md` - Team rosters
- `/home/user001/.claude/projects/-mnt-ollama-git-RADAPI01/memory/deployment.md` - Deployment info
- `/home/user001/.claude/projects/-mnt-ollama-git-RADAPI01/memory/architecture.md` - Architecture decisions

---

## Task Tracking with MCP

```bash
# Create todos for team tasks
mcp__radical-mcp__todowrite --todos [
  {"id": "hotel-1", "content": "Verify deployment", "status": "in_progress", "priority": "high"},
  {"id": "hotel-2", "content": "Check containers", "status": "pending", "priority": "high"}
]

# Read current todos
mcp__radical-mcp__todoread
```

---

## Best Practices

1. **Always use TeamCreate** before spawning team members
2. **Delete old teams** before creating new ones (one team per leader)
3. **Spawn members in parallel** for faster startup
4. **Use descriptive names** for team members (role-based)
5. **Track tasks with TaskList** to monitor progress
6. **Save results to memory** for cross-session persistence
7. **Use TEAM-007 compliance**: 4-6 members per team

---

## Example Workflow

```bash
# 1. Create team
TeamCreate --team_name "hotel-deployment" --description "Team Hotel" --agent_type "devops-engineer"

# 2. Spawn members (parallel)
Task --team_name "hotel-deployment" --name "devops-lead" --subagent_type "devops-engineer" --prompt "..."
Task --team_name "hotel-deployment" --name "container-engineer" --subagent_type "devops-engineer" --prompt "..."
Task --team_name "hotel-deployment" --name "deployment-engineer" --subagent_type "devops-engineer" --prompt "..."

# 3. Monitor progress
TaskList

# 4. Collect results
SendMessage --type "message" --recipient "devops-lead" --content "Report deployment status"

# 5. Clean up
TeamDelete
```

---

**Maintained By**: Team Alpha (Architecture)
**Last Updated**: 2026-02-16
