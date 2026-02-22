# RAD Gateway A2A - Kubernetes Cluster Module
# Provisions Kubernetes cluster with required namespaces for A2A Gateway

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
  }
}

# Gateway system namespace - for Envoy Gateway controller
resource "kubernetes_namespace" "gateway_system" {
  metadata {
    name = "gateway-system"
    labels = {
      "gateway-api.envoyproxy.io/managed" = "true"
      "app.kubernetes.io/name"             = "gateway-system"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
    }
  }
}

# SPIRE namespace - for workload identity
resource "kubernetes_namespace" "spire" {
  metadata {
    name = "spire"
    labels = {
      "app.kubernetes.io/name"             = "spire"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
      "spiffe.io/spire-managed-namespace"  = "true"
    }
  }
}

# A2A Agents namespace - where agent workloads run
resource "kubernetes_namespace" "a2a_agents" {
  metadata {
    name = "a2a-agents"
    labels = {
      "app.kubernetes.io/name"             = "a2a-agents"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
      "spiffe.io/spire-managed-namespace"  = "true"
      "gateway-route-access"               = "true"
    }
  }
}

# Observability namespace - for OTel, Jaeger, Prometheus
resource "kubernetes_namespace" "observability" {
  metadata {
    name = "observability"
    labels = {
      "app.kubernetes.io/name"             = "observability"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
    }
  }
}

# Kafka namespace - for async eventing
resource "kubernetes_namespace" "kafka" {
  metadata {
    name = "kafka"
    labels = {
      "app.kubernetes.io/name"             = "kafka"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
    }
  }
}

# RAD Gateway backend namespace
resource "kubernetes_namespace" "radgateway" {
  metadata {
    name = "radgateway"
    labels = {
      "app.kubernetes.io/name"             = "radgateway"
      "app.kubernetes.io/part-of"          = "a2a-gateway"
    }
  }
}
