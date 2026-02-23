# RAD Gateway A2A - SPIRE Server and Agent Module
# Installs SPIFFE/SPIRE for workload identity with X.509 SVIDs

terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.16"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
  }
}

variable "trust_domain" {
  description = "SPIFFE trust domain"
  type        = string
  default     = "internal.corp"
}

variable "spire_namespace" {
  description = "Namespace for SPIRE components"
  type        = string
  default     = "spire"
}

# SPIRE Server
resource "helm_release" "spire_server" {
  name       = "spire-server"
  repository = "https://spiffe.github.io/helm-charts"
  chart      = "spire-server"
  version    = "1.10.0"
  namespace  = var.spire_namespace

  set {
    name  = "spire_server.config.trustDomain"
    value = var.trust_domain
  }

  # DataStore plugin - SQLite for development, PostgreSQL for production
  set {
    name  = "spire_server.config.plugins.DataStore.sql.database_type"
    value = "sqlite3"
  }

  set {
    name  = "spire_server.config.plugins.DataStore.sql.database_name"
    value = "/run/spire/data/datastore.sqlite3"
  }

  # NodeAttestor plugin - Kubernetes
  set {
    name  = "spire_server.config.plugins.NodeAttestor.k8s.clusters.k8s-cluster.useTokenReviewAPIValidation"
    value = "true"
  }

  # KeyManager plugin - disk
  set {
    name  = "spire_server.config.plugins.KeyManager.disk.keys_path"
    value = "/run/spire/data/keys.json"
  }

  # CredentialComposer plugin - k8s-workload
  set {
    name  = "spire_server.config.plugins.CredentialComposer.k8s-workload.cluster"
    value = "k8s-cluster"
  }

  # Prometheus scraping annotation
  set {
    name  = "spire_server.annotations.prometheus\\.io/scrape"
    value = "true"
  }

  timeout = 300
}

# SPIRE Agent - runs as DaemonSet on each node
resource "helm_release" "spire_agent" {
  name       = "spire-agent"
  repository = "https://spiffe.github.io/helm-charts"
  chart      = "spire-agent"
  version    = "1.10.0"
  namespace  = var.spire_namespace

  set {
    name  = "spire_agent.config.trustDomain"
    value = var.trust_domain
  }

  set {
    name  = "spire_agent.config.server.address"
    value = "spire-server.${var.spire_namespace}"
  }

  set {
    name  = "spire_agent.config.server.port"
    value = "8081"
  }

  # Host socket for workload attestation
  set {
    name  = "hostSocket"
    value = "/run/spire/sockets"
  }

  # SVID store for Envoy SDS integration
  set {
    name  = "svidStore.enabled"
    value = "true"
  }

  # Extra volume mounts for Envoy sidecars
  set {
    name  = "extraVolumeMounts[0].name"
    value = "spire-socket"
  }

  set {
    name  = "extraVolumeMounts[0].mountPath"
    value = "/run/spire/sockets"
  }

  set {
    name  = "extraVolumeMounts[0].readOnly"
    value = "true"
  }

  depends_on = [helm_release.spire_server]
  timeout    = 300
}

output "spire_server_namespace" {
  description = "Namespace where SPIRE Server is installed"
  value       = var.spire_namespace
}

output "spire_agent_namespace" {
  description = "Namespace where SPIRE Agent is installed"
  value       = var.spire_namespace
}

output "trust_domain" {
  description = "Configured SPIFFE trust domain"
  value       = var.trust_domain
}

output "spire_socket_path" {
  description = "Path to SPIRE agent socket for SDS integration"
  value       = "/run/spire/sockets/agent.sock"
}
