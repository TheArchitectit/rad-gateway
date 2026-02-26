# RAD Gateway A2A - Root Terraform Configuration
# Orchestrates all infrastructure modules for A2A Gateway deployment

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.16"
    }
  }

  backend "kubernetes" {
    secret_suffix     = "a2a-gateway-terraform"
    namespace         = "kube-system"
    create_namespace  = true
  }
}

# Provider configuration
provider "kubernetes" {
  config_path = "~/.kube/config"
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

# Local variables
locals {
  cluster_name   = var.cluster_name
  environment    = var.environment
  spire_domain   = var.spire_trust_domain
}

# Module: Kubernetes Namespaces
module "cluster" {
  source = "./modules/cluster"

  cluster_name          = local.cluster_name
  environment           = local.environment
  spire_trust_domain    = local.spire_domain
  enable_network_policies = true
}

# Module: Gateway API CRDs
module "gateway_api" {
  source = "./modules/gateway-api"

  depends_on = [module.cluster]
}

# Module: Cilium eBPF Networking
module "cilium" {
  source = "./modules/cilium"

  cluster_name   = local.cluster_name
  spire_enabled  = true

  depends_on = [module.cluster]
}

# Module: SPIRE Workload Identity
module "spire" {
  source = "./modules/spire"

  trust_domain     = local.spire_domain
  spire_namespace  = module.cluster.spire_namespace

  depends_on = [module.cilium]
}

# Outputs
output "namespaces" {
  description = "Created namespaces"
  value = {
    gateway_system  = module.cluster.gateway_system_namespace
    spire           = module.cluster.spire_namespace
    a2a_agents      = module.cluster.a2a_agents_namespace
    observability   = module.cluster.observability_namespace
    kafka           = module.cluster.kafka_namespace
    radgateway      = module.cluster.radgateway_namespace
  }
}

output "gateway_class" {
  description = "GatewayClass name"
  value       = module.gateway_api.gateway_class_name
}

output "spire_socket_path" {
  description = "Path to SPIRE agent socket for SDS integration"
  value       = module.spire.spire_socket_path
}

output "cilium_version" {
  description = "Installed Cilium version"
  value       = module.cilium.cilium_version
}
