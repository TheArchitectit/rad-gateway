# RAD Gateway A2A - Cluster Module Variables

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "a2a-gateway-cluster"
}

variable "environment" {
  description = "Environment name (development, staging, production)"
  type        = string
  default     = "development"
}

variable "region" {
  description = "Cloud provider region"
  type        = string
  default     = "us-east-1"
}

variable "enable_network_policies" {
  description = "Enable network policies for namespace isolation"
  type        = bool
  default     = true
}

variable "spire_trust_domain" {
  description = "SPIFFE trust domain for workload identity"
  type        = string
  default     = "internal.corp"
}
