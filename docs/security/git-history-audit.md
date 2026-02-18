# Git History Security Audit Report

**Date**: 2026-02-18
**Scope**: Full git history scan for exposed secrets
**Commits Scanned**: 57

---

## CRITICAL FINDINGS

### 1. Infisical Service Tokens (CRITICAL)

**Status**: ✅ Removed from current HEAD but **EXISTS IN HISTORY**

**Location**: `.env` file (lines 11-12)
**Token Pattern**: `st.227fc651-7fe7-4cb2-8b93-3bbc0bf72469...`

**Timeline**:
- First introduced: Unknown (prior to commit `2a4dad2`)
- Removed: Commit `2a4dad2` (security: Remove exposed Infisical tokens from .env)
- **STILL IN GIT HISTORY**: Anyone can access via `git log -p`

**Risk**: Full access to Infisical secrets management system

### 2. Hardcoded Database Passwords (HIGH)

**Location**: `config/env.example` and `config/env.local`

**Password Found**:
```
radgateway_secure_password_2024
```

**Usage**:
```
RAD_DB_DSN=postgresql://radgateway_user:radgateway_secure_password_2024@localhost:5432/radgateway?sslmode=disable
```

**Risk**: Production database password exposed in example files

### 3. Other Exposed Credentials

| Type | Location | Risk |
|------|----------|------|
| API Keys | `.env` (historical) | MEDIUM |
| JWT Fallback Secrets | `internal/auth/jwt.go` | MEDIUM (runtime generated) |
| Redis Password | `.env` (historical) | MEDIUM |

---

## Git History Evidence

### Commits with Secrets

| Commit | Description | Secrets |
|--------|-------------|---------|
| `2a4dad2` | Remove tokens | Token removal (still in parent commits) |
| `9efc2b5` | Infisical integration | Likely introduced tokens |
| `b3f18c1` | Add internal/secrets | Related to secrets |
| Various | env.example/env.local | Hardcoded passwords |

---

## REQUIRED ACTIONS

### Immediate (Within 24 hours)

1. **REVOKE Infisical Token**
   ```bash
   # In Infisical dashboard:
   # 1. Go to Project Settings > Service Tokens
   # 2. Revoke: st.227fc651-7fe7-4cb2-8b93-3bbc0bf72469...
   # 3. Generate new token
   # 4. Store securely on host ONLY (not in git)
   ```

2. **Change Database Password**
   ```bash
   # On PostgreSQL host:
   ALTER USER radgateway_user WITH PASSWORD 'NEW_RANDOM_PASSWORD';
   ```

3. **Clean Git History**
   ```bash
   # Use git-filter-repo (recommended) or BFG
   # Remove .env from entire history
   # Remove config/env.example and config/env.local passwords
   ```

### Short-term (This Week)

4. **Scan for Other Secrets**
   ```bash
   # Install and run gitleaks
   gitleaks detect --source . --verbose

   # Or truffleHog
   truffleHog git file://. --only-verified
   ```

5. **Force Push Cleaned History**
   ```bash
   # After history rewrite
   git push origin main --force-with-lease

   # Notify all collaborators
   ```

6. **Update .gitignore**
   ```
   # Already present, but verify:
   .env
   .env.*
   *.token
   *.key
   secrets/
   config/*.local
   ```

---

## History Cleaning Commands

### Option 1: Remove .env entirely from history
```bash
# Using git-filter-repo
pip install git-filter-repo
git filter-repo --path .env --invert-paths
```

### Option 2: Remove specific secrets
```bash
# Using BFG Repo-Cleaner
java -jar bfg.jar --delete-files .env .
java -jar bfg.jar --replace-text passwords.txt .
```

### Option 3: Interactive rebase (small repos)
```bash
git rebase -i --root
# Edit commits that introduced secrets
```

---

## Prevention Measures

1. **Pre-commit hooks**
   ```bash
   # Install pre-commit with gitleaks
   pre-commit install
   ```

2. **CI/CD scanning**
   ```yaml
   # Add to GitHub Actions
   - uses: gitleaks/gitleaks-action@v2
   ```

3. **Secret management policy**
   - All secrets via Infisical/Vault
   - No exceptions for "temporary" or "example" files
   - Regular rotation schedule

---

## Verification Commands

```bash
# Check if token still in history
git log --all -p | grep "st.227fc651"

# Find commits with .env
git log --all --full-history -- .env

# Search for password pattern
git log --all -p | grep "radgateway_secure_password"

# List all files ever committed
git log --all --name-only --pretty=format: | sort -u
```

---

## Conclusion

**Current Status**: ⚠️ **VULNERABLE**
- Secrets removed from HEAD ✅
- Secrets still in history ❌
- Git history needs cleaning ❌

**Next Steps**:
1. Revoke exposed tokens immediately
2. Clean git history
3. Force push
4. Notify team
5. Implement pre-commit hooks

**Estimated Time**: 2-4 hours
