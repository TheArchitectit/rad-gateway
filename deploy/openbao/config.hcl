# OpenBao Configuration for Golden Stack Cold Vault
# This configuration provides long-term secret storage with PostgreSQL backend

# Storage backend configuration - PostgreSQL
storage "postgresql" {
  # Connection string will be constructed from environment variables
  # Format: postgres://username:password@host:port/database?sslmode=mode
  connection_url = "${BAO_PG_CONNECTION_URL}"

  # Table name for OpenBao data
  table = "openbao_kv_store"

  # Maximum number of parallel connections
  max_parallel = 16

  # Connection pool settings
  max_idle_connections = 4
  max_connection_lifetime = "30m"

  # HA settings (for high availability mode)
  # ha_enabled = true
  # ha_table = "openbao_ha_locks"
}

# Listener configuration - TCP
listener "tcp" {
  # Listen address
  address = "0.0.0.0:8200"

  # TLS configuration (disabled for internal network, enable in production)
  # tls_cert_file = "/openbao/config/tls.crt"
  # tls_key_file = "/openbao/config/tls.key"
  # tls_min_version = "tls13"

  # Proxy protocol support (if behind load balancer)
  # proxy_protocol_behavior = "allow_authorized"
  # proxy_protocol_authorized_addrs = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

  # HTTP response headers
  x_forwarded_for_authorized_addrs = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
  x_forwarded_for_hop_skips = 0
  x_forwarded_for_reject_not_authorized = true
  x_forwarded_for_reject_not_present = false
}

# API and UI configuration
api_addr = "${BAO_API_ADDR}"
cluster_addr = "${BAO_CLUSTER_ADDR}"

# Enable UI
disable_mlock = false
ui = true

# Logging configuration
log_level = "${BAO_LOG_LEVEL}"
log_format = "json"

# Audit log configuration
audit_file {
  path = "/openbao/logs/audit.log"
  log_raw = false
  hmac_accessor = true
  mode = "0750"
  format = "json"
}

# Telemetry configuration (optional - for monitoring)
telemetry {
  statsite_address = ""
  statsd_address = ""
  disable_hostname = false
  enable_hostname_label = true
  enable_runtime_metrics = true
  usage_gauge_period = "10m"
  maximum_gauge_cardinality = 500
  lease_metrics_epsilon = "1h"
  num_lease_metrics_buckets = 2
  lease_metrics_name_resolution = "false"
  lease_metrics_name_resolution = "schema"
}

# Cold vault specific settings
# These settings optimize for long-term retention
max_lease_ttl = "87600h"      # 10 years max lease TTL
default_lease_ttl = "43800h"  # 5 years default lease TTL

# Plugin directory (if using custom plugins)
# plugin_directory = "/openbao/plugins"

# Enable raw endpoint for disaster recovery (disable in production)
# raw_storage_endpoint = true

# Cluster configuration (if using HA)
# cluster_name = "golden-stack-cold-vault"

# Performance standby nodes (if using enterprise features)
# disable_performance_standby = false

# Seals configuration (for auto-unseal, recommended for production)
# seal "awskms" {
#   region     = "us-east-1"
#   kms_key_id = "arn:aws:kms:us-east-1:..."
# }

# seal "transit" {
#   address = "https://primary-openbao:8200"
#   token   = "s.xxx"
#   key_name = "unseal-key"
# }
