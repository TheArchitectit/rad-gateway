# RAD Gateway A2A - Root Variables

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "a2a-gateway-cluster"
}

variable "environment" {
  description = "Environment name (development, staging, production)"
  type        = string
  default     = "development"

  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be one of: development, staging, production."
  }
}

variable "region" {
  description = "Cloud provider region"
  type        = string
  default     = "us-east-1"
}

variable "spire_trust_domain" {
  description = "SPIFFE trust domain for workload identity"
  type        = string
  default     = "internal.corp"

  validation {
    condition     = can(regex("^[a-z0-9._-]+$", var.spire_trust_domain))
    error_message = "Trust domain must be a valid SPIFFE domain (lowercase alphanumeric with dots, hyphens, underscores)."
  }
}

variable "terraform_backend_config" {
  description = "Kubernetes backend configuration for Terraform state"
  type        = map(string)
  default = {
    secret_suffix     = "a2a-gateway-terraform"
    namespace         = "kube-system"
    create_namespace  = true
  }
}
