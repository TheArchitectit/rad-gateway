# RAD Gateway 01 Monitoring Procedures

**Version**: 1.0
**Last Updated**: 2026-02-16
**Team**: Team Hotel (Deployment & Infrastructure)
**Classification**: Public-safe (no internal IPs or credentials)

---

## Overview

This document provides monitoring procedures for RAD Gateway 01 that are safe for public documentation and external monitoring systems.

## Service Level Objectives (SLOs)

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| API Availability | 99.5% | < 99.0% |
| Error Rate (5xx/total) | < 1.0% | > 2.0% |
| P95 Latency (non-stream) | < 1000ms | > 1200ms |

## Health Endpoint

### Primary Health Check

```bash
curl -s http://<gateway-host>:8090/health
```

**Expected Response**:
```json
{
  "status": "healthy",
  "version": "0.1.0-alpha"
}
```

**HTTP Status Codes**:
- `200 OK` - Service is healthy
- `503 Service Unavailable` - Service is starting or unhealthy

### Health Check Script

For automated monitoring, use the health check script:

```bash
/opt/radgateway01/bin/health-check.sh
```

Exit codes:
- `0` - All checks passed
- `1` - One or more checks failed

For detailed output:
```bash
/opt/radgateway01/bin/health-check.sh --verbose
```

## Metrics Endpoint

Prometheus-compatible metrics are available at:

```bash
curl -s http://<gateway-host>:8090/metrics
```

### Key Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `radgateway_requests_total` | Counter | Total API requests |
| `radgateway_request_duration_seconds` | Histogram | Request latency |
| `radgateway_provider_health` | Gauge | Provider health status |
| `radgateway_usage_tokens_total` | Counter | Token usage by provider |

## Monitoring Commands

### Service Status

```bash
# Check systemd service
sudo systemctl status radgateway01

# Check if service is active
sudo systemctl is-active radgateway01

# View recent logs
sudo journalctl -u radgateway01 -n 50

# Follow logs in real-time
sudo journalctl -u radgateway01 -f
```

### Container Status

```bash
# List running containers
sudo podman ps --pod

# View container logs
sudo podman logs radgateway01-app

# Follow container logs
sudo podman logs -f radgateway01-app
```

### Resource Monitoring

```bash
# Container resource usage
sudo podman stats radgateway01-app

# System resource usage
df -h /opt/radgateway01
du -sh /opt/radgateway01/data
```

## Alerting Conditions

### Critical Alerts (Immediate Response)

1. **Service Down**
   - Condition: Health endpoint returns non-200 for 2 minutes
   - Action: Check systemd status, restart if necessary

2. **High Error Rate**
   - Condition: Error rate > 2% for 10 minutes
   - Action: Check application logs, verify provider connectivity

3. **High Latency**
   - Condition: P95 latency > 1200ms for 15 minutes
   - Action: Check resource usage, verify network connectivity

### Warning Alerts (Investigation Required)

1. **Disk Space Low**
   - Condition: Disk usage > 80%
   - Action: Run backup cleanup, expand storage if needed

2. **Memory Usage High**
   - Condition: Memory usage > 85%
   - Action: Monitor for leaks, restart if necessary

## Backup Monitoring

### Automated Backups

Backups run automatically via cron. Verify backup status:

```bash
# List recent backups
ls -la /backup/radgateway01/

# Check backup manifest
cat /backup/radgateway01/<timestamp>/manifest.txt

# Verify backup integrity
cd /backup/radgateway01/<timestamp> && sha256sum -c checksums.sha256
```

### Manual Backup

```bash
# Create manual backup
/opt/radgateway01/bin/backup.sh
```

## Log Retention

| Log Type | Location | Retention |
|----------|----------|-----------|
| Application | `/opt/radgateway01/logs/` | 14 days |
| Usage Data | `/opt/radgateway01/data/usage/` | 30 days |
| Systemd Journal | `journalctl -u radgateway01` | System default |

## External Monitoring Integration

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'radgateway01'
    static_configs:
      - targets: ['gateway-host:8090']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Nagios/Icinga Check

```bash
# NRPE check command
define command {
    command_name check_radgateway01
    command_line /opt/radgateway01/bin/health-check.sh
}
```

### Uptime Kuma

- Monitor Type: HTTP(s)
- URL: `http://gateway-host:8090/health`
- Expected Status Code: 200
- Heartbeat Interval: 60 seconds

## Runbook References

For troubleshooting procedures, see:
- [RUNBOOK.md](RUNBOOK.md) - Operations runbook
- [deployment-radgateway01.md](deployment-radgateway01.md) - Deployment specification

---

**Note**: Replace `<gateway-host>` with your actual gateway hostname or IP address as appropriate for your environment.
