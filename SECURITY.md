# Security Policy

## Secret Handling

- Never commit secrets, credentials, or tokens.
- Use `.env` for local/runtime secrets.
- Commit only `.env.example` with placeholders.
- Source of truth for secrets: Infisical.
  - `.env` should contain only bootstrap values (e.g. `INFISICAL_API_URL`, `INFISICAL_SERVICE_TOKEN`).
  - Provider keys and runtime secrets should be stored in Infisical and injected at deploy/runtime.
- Rotate any credential immediately if exposure is suspected.

## Reporting a Vulnerability

- Open a private security advisory on GitHub if available.
- Otherwise contact the maintainers directly with:
  - impact summary
  - reproduction steps
  - affected versions/paths

## Scope Notes

- This repository is public-by-default.
- All contributors must verify no secret material is present before push.
