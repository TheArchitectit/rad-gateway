# RAD Gateway A2A - Gateway API CRDs Module
# Installs Gateway API v1.4+ CRDs for ingress and mesh routing

terraform {
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
}

# Install Gateway API CRDs via Helm
resource "helm_release" "gateway_api_crds" {
  name       = "gateway-api-crds"
  repository = "https://kubernetes-sigs.github.io/gateway-api"
  chart      = "gateway-api-crds"
  version    = "v1.4.0"
  namespace  = "gateway-system"

  # Wait for CRDs to be ready before proceeding
  wait = true
  timeout = 300
}

# Gateway API Policy Attachment CRDs (for BackendTrafficPolicy, SecurityPolicy)
resource "helm_release" "envoy_gateway_crds" {
  name       = "envoy-gateway-crds"
  repository = "https://gateway.envoyproxy.io"
  chart      = "envoy-gateway-crds"
  version    = "v1.2.0"
  namespace  = "gateway-system"

  depends_on = [helm_release.gateway_api_crds]
  wait       = true
  timeout    = 300
}

# Create GatewayClass for Envoy Gateway
resource "kubernetes_manifest" "gateway_class" {
  manifest = {
    apiVersion = "gateway.networking.k8s.io/v1"
    kind       = "GatewayClass"
    metadata = {
      name = "a2a-gateway-class"
      annotations = {
        "gateway.envoyproxy.io/gatewayclass-controller-name" = "envoy-gateway-controller"
      }
    }
    spec = {
      controllerName = "gateway.envoyproxy.io/gatewayclass-controller"
      description    = "A2A Gateway with Envoy proxy for AI agent communication"
    }
  }

  depends_on = [helm_release.envoy_gateway_crds]
}

# Output the GatewayClass name for reference
output "gateway_class_name" {
  description = "Name of the created GatewayClass"
  value       = kubernetes_manifest.gateway_class.manifest.metadata.name
}

output "gateway_class_controller" {
  description = "Controller name for the GatewayClass"
  value       = kubernetes_manifest.gateway_class.manifest.spec.controllerName
}
