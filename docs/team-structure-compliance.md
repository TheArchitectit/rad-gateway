# Team Structure Compliance Guide

**Version**: 1.0
**Date**: 2026-02-16
**Rule**: TEAM-007 (from `.guardrails/team-layout-rules.json`)

---

## TEAM-007: Team Size Compliance (Mandatory)

All teams must have between **4 and 6 members** (inclusive).

```
Minimum: 4 members
Maximum: 6 members
Target: 5-6 members for optimal collaboration
```

---

## Current RAD Gateway Teams (8 Teams, 43 Members)

All teams are compliant with TEAM-007:

### Team Alpha: Architecture & Design (6 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| Chief Architect | Team 2 | 5-year tech vision |
| Solution Architect | Team 2 | Standards mapping |
| Domain Architect | Team 2 | Go/API gateway expertise |
| API Product Manager | Team 8 | API lifecycle |
| Technical Lead | Team 7 | Implementation decisions |
| Standards Lead | Team 2 | Approved Tech List |

**Status**: ✅ Compliant (6 members)

---

### Team Bravo: Core Implementation (6 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| Senior Backend Engineer #1 | Team 7 | Core API logic |
| Senior Backend Engineer #2 | Team 7 | Provider adapters |
| Integration Engineer | Team 8 | Provider connections |
| Messaging Engineer | Team 8 | Kafka/RabbitMQ |
| IAM Specialist | Team 8 | Auth/API keys |
| Technical Writer | Team 7 | Implementation docs |

**Status**: ✅ Compliant (6 members)

---

### Team Charlie: Security Hardening (5 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| Security Architect | Team 9 | Threat model review |
| Vulnerability Researcher | Team 9 | SAST/DAST/SCA |
| Penetration Tester | Team 9 | Security testing |
| DevSecOps Engineer | Team 9 | CI/CD security gates |
| Privacy Engineer | Team 3 | Data masking/PII |

**Status**: ✅ Compliant (5 members)

---

### Team Delta: Quality Assurance (5 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| QA Architect | Team 10 | Testing strategy |
| SDET | Team 10 | Automated tests |
| Performance/Load Engineer | Team 10 | Scale testing |
| Manual QA/UAT Coordinator | Team 10 | Acceptance testing |
| Accessibility (A11y) Expert | Team 7 | WCAG compliance |

**Status**: ✅ Compliant (5 members)

---

### Team Echo: Operations & Observability (5 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| SRE Lead | Team 11 | Error budget/SLA |
| Observability Engineer | Team 11 | Monitoring/logging |
| Chaos Engineer | Team 11 | Resiliency testing |
| Incident Manager | Team 11 | War room leadership |
| Release Manager | Team 12 | Go/No-Go coordination |

**Status**: ✅ Compliant (5 members)

---

### Team Foxtrot: Inspiration Analysis (5 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| Solution Architect | Team 2 | Pattern extraction |
| Senior Backend Engineer | Team 7 | Pattern analysis |
| Integration Engineer | Team 8 | Adapter patterns |
| Domain Architect | Team 2 | Go translation |
| Technical Writer | Team 7 | Findings documentation |

**Status**: ✅ Compliant (5 members)

---

### Team Golf: Documentation & Design (6 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| Lead Technical Writer | Team 7 | Documentation architecture |
| UX Designer | Team 7 | Steampunk interface |
| A2A Protocol Writer | Team 2 | Protocol specs |
| Developer Advocate | Team 8 | Developer experience |
| Information Architect | Team 2 | Content structure |
| Editor-in-Chief | Team 7 | Quality assurance |

**Status**: ✅ Compliant (6 members)

---

### Team Hotel: Deployment & Infrastructure (5 Members)
| Role | Source Team | Responsibility |
|------|-------------|----------------|
| DevOps Lead | Team 11 | Infrastructure orchestration |
| Container Engineer | Team 4 | Podman/Docker mgmt |
| Deployment Engineer | Team 12 | Release automation |
| Infrastructure Architect | Team 2 | Infrastructure design |
| Systems Administrator | Team 4 | Host management |

**Status**: ✅ Compliant (5 members)

---

## Future Team Creation Template

When creating a new team, use this template to ensure TEAM-007 compliance:

```markdown
### Team [Name]: [Purpose] (X Members)
**Purpose**: [One-line description]

| Role | Source Team | Responsibility |
|------|-------------|----------------|
| [Role 1] | Team [N] | [Responsibility] |
| [Role 2] | Team [N] | [Responsibility] |
| [Role 3] | Team [N] | [Responsibility] |
| [Role 4] | Team [N] | [Responsibility] |
| [Role 5] | Team [N] | [Responsibility] |
| [Role 6] | Team [N] | [Responsibility] | *(optional)*

**Status**: [ ] Needs Members | [ ] Compliant
```

---

## TEAM-007 Validation Checklist

Before starting work, verify:

- [ ] Team has at least 4 members (minimum)
- [ ] Team has at most 6 members (maximum)
- [ ] Each member has a defined role
- [ ] Each role has clear responsibilities
- [ ] Source team is documented for each role
- [ ] Team purpose is documented

**Validation Command**:
```bash
python scripts/team_manager.py --project rad-gateway validate-size
```

---

## Role Distribution Guidelines

### Recommended Role Mix by Team Type

**Architecture Teams** (5-6 members):
- 2x Architects (Chief, Solution, or Domain)
- 1x Technical Lead
- 1-2x Specialists (Product Manager, Standards Lead)
- 1x Technical Writer

**Implementation Teams** (5-6 members):
- 2-3x Engineers (Senior Backend/Frontend)
- 1-2x Integration Engineers
- 1x Specialist (Security, IAM, Messaging)
- 1x Technical Writer

**Security Teams** (4-5 members):
- 1x Security Architect
- 1-2x Security Engineers (Vulnerability, PenTest)
- 1x DevSecOps Engineer
- 1x Privacy/Compliance Engineer

**QA Teams** (4-5 members):
- 1x QA Architect
- 1-2x SDETs
- 1x Performance/Load Engineer
- 1x UAT/Manual QA

**Operations Teams** (4-5 members):
- 1x SRE Lead
- 1x Observability Engineer
- 1x Chaos/Resiliency Engineer
- 1x Incident/Release Manager
- 1x DevOps/Infrastructure Engineer

**Deployment Teams** (4-5 members):
- 1x DevOps Lead
- 1-2x Container/Platform Engineers
- 1x Deployment/Release Engineer
- 1x Infrastructure/Systems Engineer

---

## Common Anti-Patterns to Avoid

❌ **Under-staffed teams** (< 4 members):
- Risk: Burnout, single points of failure
- Fix: Merge with related team or add members

❌ **Over-staffed teams** (> 6 members):
- Risk: Coordination overhead, reduced agility
- Fix: Split into focused sub-teams

❌ **Undefined roles**:
- Risk: Unclear ownership, duplicated work
- Fix: Document RACI matrix for all roles

❌ **Missing technical writer**:
- Risk: Documentation debt
- Fix: Add technical writer to all teams

---

## Reference Documents

- `.guardrails/team-layout-rules.json` - Official guardrails
- `docs/guardrails-adoption.md` - Guardrails adoption report
- `docs/analysis/team-*-*.md` - Individual team deliverables

---

**Maintained By**: Team Alpha (Architecture)
**Last Updated**: 2026-02-16
**Next Review**: After team formation for Milestone 2
