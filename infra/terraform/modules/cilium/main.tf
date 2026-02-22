# RAD Gateway A2A - Cilium eBPF Networking Module
# Installs Cilium for eBPF-based networking with mTLS support

terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.16"
    }
  }
}

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "spire_enabled" {
  description = "Enable SPIRE integration for mTLS"
  type        = bool
  default     = true
}

resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io"
  chart      = "cilium"
  version    = "1.18.0"
  namespace  = "kube-system"

  # eBPF dataplane configuration
  set {
    name  = "enableBpfMasquerade"
    value = "true"
  }

  set {
    name  = "enableIdentityMark"
    value = "true"
  }

  set {
    name  = "bpfClockProbe"
    value = "true"
  }

  # kube-proxy replacement
  set {
    name  = "kubeProxyReplacement"
    value = "true"
  }

  # L7 policy engine
  set {
    name  = "l7Proxy"
    value = "true"
  }

  # mTLS configuration
  dynamic "set" {
    for_each = var.spire_enabled ? [1] : []
    content {
      name  = "authentication.enabled"
      value = "true"
    }
  }

  dynamic "set" {
    for_each = var.spire_enabled ? [1] : []
    content {
      name  = "authentication.mutual.spire.enabled"
      value = "true"
    }
  }

  dynamic "set" {
    for_each = var.spire_enabled ? [1] : []
    content {
      name  = "authentication.mutual.spire.serverSocketPath"
      value = "/run/spire/sockets/agent.sock"
    }
  }

  # IPAM mode
  set {
    name  = "ipam.mode"
    value = "kubernetes"
  }

  # Rollout configuration
  set {
    name  = "rollOutCiliumPods"
    value = "true"
  }

  # Cluster name for identity
  set {
    name  = "cluster.name"
    value = var.cluster_name
  }

  timeout = 600
}

output "cilium_namespace" {
  description = "Namespace where Cilium is installed"
  value       = "kube-system"
}

output "cilium_version" {
  description = "Installed Cilium version"
  value       = "1.18.0"
}
