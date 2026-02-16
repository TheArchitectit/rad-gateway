# RAD Gateway 01 - Common Issues Runbook

**Version**: 1.0
**Last Updated**: 2026-02-16
**Team**: Team Hotel (Deployment & Infrastructure)
**Classification**: Public-safe (no internal IPs)

---

## Quick Reference

| Issue | Command | Exit Code |
|-------|---------|-----------|
| Service down | `sudo systemctl status radgateway01` | - |
| Health check | `/opt/radgateway01/bin/health-check.sh` | 0=healthy, 1=unhealthy |
| View logs | `sudo journalctl -u radgateway01 -n 50` | - |
| Restart | `sudo systemctl restart radgateway01` | - |

---

## Issue: Service Fails to Start

### Symptoms
- `systemctl status radgateway01` shows failed state
- Health endpoint not responding
- Error messages in logs

### Diagnostic Steps

1. **Check service status**:
   ```bash
   sudo systemctl status radgateway01
   ```

2. **View recent logs**:
   ```bash
   sudo journalctl -u radgateway01 -n 100
   ```

3. **Check for port conflicts**:
   ```bash
   sudo ss -tlnp | grep 8090
   ```

4. **Verify Infisical connectivity**:
   ```bash
   curl http://localhost:8080/api/status
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| Port 8090 in use | Stop conflicting service or change port |
| Infisical not running | Start infisical.service first |
| Missing Infisical token | Add token to `/opt/radgateway01/config/infisical-token` |
| Token file permissions | `sudo chmod 600 /opt/radgateway01/config/infisical-token` |
| Container image missing | Build image: `sudo podman build -t radgateway01:latest .` |

---

## Issue: Health Check Failing

### Symptoms
- Health endpoint returns non-200 status
- Container running but unhealthy
- Intermittent failures

### Diagnostic Steps

1. **Run health check**:
   ```bash
   /opt/radgateway01/bin/health-check.sh --verbose
   ```

2. **Check HTTP response**:
   ```bash
   curl -v http://localhost:8090/health
   ```

3. **Check container logs**:
   ```bash
   sudo podman logs radgateway01-app
   ```

4. **Verify process is running**:
   ```bash
   sudo podman exec radgateway01-app ps aux
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| Application starting up | Wait for startup to complete (30-60s) |
| Provider API unavailable | Check provider connectivity |
| Resource exhaustion | Check `df -h` and `free -h` |
| Container health check failing | Restart container: `sudo podman restart radgateway01-app` |

---

## Issue: High Error Rate

### Symptoms
- Increased 5xx responses
- Provider errors in logs
- Degraded service quality

### Diagnostic Steps

1. **Check metrics**:
   ```bash
   curl -s http://localhost:8090/metrics | grep radgateway_requests_total
   ```

2. **View error logs**:
   ```bash
   sudo journalctl -u radgateway01 -p err -n 50
   ```

3. **Check provider health**:
   ```bash
   curl -s http://localhost:8090/metrics | grep radgateway_provider_health
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| Provider API key expired | Update key in Infisical |
| Rate limiting from provider | Reduce request rate or add retry logic |
| Network connectivity issues | Check network status |
| Insufficient resources | Scale up CPU/memory |

---

## Issue: High Latency

### Symptoms
- Slow API responses
- P95 latency > 1200ms
- Timeouts from clients

### Diagnostic Steps

1. **Check current latency**:
   ```bash
   curl -s http://localhost:8090/metrics | grep radgateway_request_duration_seconds
   ```

2. **Monitor resources**:
   ```bash
   sudo podman stats radgateway01-app
   ```

3. **Check for resource limits**:
   ```bash
   sudo systemctl show radgateway01 --property=LimitNOFILE,LimitNPROC
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| CPU throttling | Increase CPU limits or add cores |
| Memory pressure | Increase memory allocation |
| File descriptor limits | Increase `LimitNOFILE` in systemd unit |
| Network latency | Check network path to providers |
| Large request payloads | Implement request size limits |

---

## Issue: Disk Space Full

### Symptoms
- Backup failures
- Cannot write logs
- Service degradation

### Diagnostic Steps

1. **Check disk usage**:
   ```bash
   df -h /opt/radgateway01
   ```

2. **Find large files**:
   ```bash
   du -sh /opt/radgateway01/* | sort -hr
   ```

3. **Check log sizes**:
   ```bash
   find /opt/radgateway01/logs -type f -size +100M
   ```

### Solutions

1. **Clean up old logs**:
   ```bash
   sudo find /opt/radgateway01/logs -name "*.log" -mtime +7 -delete
   ```

2. **Clean up old backups**:
   ```bash
   sudo find /backup/radgateway01 -name "*.tar.gz" -mtime +7 -delete
   ```

3. **Force log rotation**:
   ```bash
   sudo logrotate -f /etc/logrotate.d/radgateway01
   ```

4. **Expand storage** (if necessary)

---

## Issue: Backup Failures

### Symptoms
- No new backups in `/backup/radgateway01/`
- Backup script errors
- Corrupt backup files

### Diagnostic Steps

1. **Run backup manually**:
   ```bash
   sudo /opt/radgateway01/bin/backup.sh
   ```

2. **Check backup directory permissions**:
   ```bash
   ls -la /backup/
   ```

3. **Verify disk space**:
   ```bash
   df -h /backup
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| Insufficient disk space | Clean up old backups or expand storage |
| Permission denied | Ensure radgateway user can write to /backup |
| Corrupt data directory | Check filesystem integrity |
| Missing backup directory | Create: `sudo mkdir -p /backup/radgateway01` |

---

## Issue: Container Won't Start

### Symptoms
- `podman ps` shows no running containers
- Container exits immediately
- Image pull errors

### Diagnostic Steps

1. **Check container logs**:
   ```bash
   sudo podman logs radgateway01-app
   ```

2. **Verify image exists**:
   ```bash
   sudo podman images | grep radgateway01
   ```

3. **Check pod status**:
   ```bash
   sudo podman pod ps
   ```

### Common Causes & Solutions

| Cause | Solution |
|-------|----------|
| Image not found | Build: `sudo podman build -t radgateway01:latest .` |
| Pod not created | Recreate: `sudo podman pod create --name radgateway01 --publish 8090:8090` |
| Volume missing | Create: `sudo podman volume create radgateway01-data` |
| Port conflict | Stop service using port 8090 |

---

## Issue: Authentication Failures

### Symptoms
- API requests return 401 Unauthorized
- Token validation errors
- Cannot access protected endpoints

### Diagnostic Steps

1. **Check API keys in Infisical**:
   - Verify RAD_API_KEYS secret exists
   - Check key format (comma-separated)

2. **Test with valid key**:
   ```bash
   curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8090/health
   ```

3. **Check startup logs for key loading**:
   ```bash
   sudo journalctl -u radgateway01 | grep -i "api_keys\|secret"
   ```

### Solutions

| Cause | Solution |
|-------|----------|
| Missing RAD_API_KEYS | Add to Infisical |
| Invalid key format | Use comma-separated list without spaces |
| Infisical connectivity | Verify Infisical is running |
| Token file permissions | Ensure 600 permissions on token file |

---

## Recovery Procedures

### Complete Service Recovery

If the service is completely down:

1. **Stop all components**:
   ```bash
   sudo systemctl stop radgateway01
   sudo podman stop radgateway01-app
   sudo podman rm radgateway01-app
   ```

2. **Clear state** (if needed):
   ```bash
   sudo podman pod rm radgateway01
   sudo podman volume rm radgateway01-data
   ```

3. **Reinstall if necessary**:
   ```bash
   cd /mnt/ollama/git/RADAPI01/deploy
   sudo ./install.sh
   ```

4. **Restore from backup** (if needed):
   ```bash
   # Stop service
   sudo systemctl stop radgateway01

   # Restore data
   sudo tar -xzf /backup/radgateway01/<timestamp>.tar.gz -C /tmp/
   sudo cp -r /tmp/<timestamp>/data/* /opt/radgateway01/data/

   # Restart
   sudo systemctl start radgateway01
   ```

5. **Verify recovery**:
   ```bash
   /opt/radgateway01/bin/health-check.sh --verbose
   ```

---

## Contact & Escalation

| Severity | Response Time | Action |
|----------|--------------|--------|
| Critical | 15 minutes | Page on-call engineer |
| High | 1 hour | Create incident ticket |
| Medium | 4 hours | Schedule fix |
| Low | 24 hours | Add to backlog |

---

**Note**: For issues not covered here, refer to the main [RUNBOOK.md](RUNBOOK.md) or contact the Operations team.
