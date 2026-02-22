# RAD Gateway A2A - Cluster Module Outputs

output "gateway_system_namespace" {
  description = "Name of the gateway system namespace"
  value       = kubernetes_namespace.gateway_system.metadata[0].name
}

output "spire_namespace" {
  description = "Name of the SPIRE namespace"
  value       = kubernetes_namespace.spire.metadata[0].name
}

output "a2a_agents_namespace" {
  description = "Name of the A2A agents namespace"
  value       = kubernetes_namespace.a2a_agents.metadata[0].name
}

output "observability_namespace" {
  description = "Name of the observability namespace"
  value       = kubernetes_namespace.observability.metadata[0].name
}

output "kafka_namespace" {
  description = "Name of the Kafka namespace"
  value       = kubernetes_namespace.kafka.metadata[0].name
}

output "radgateway_namespace" {
  description = "Name of the RAD Gateway namespace"
  value       = kubernetes_namespace.radgateway.metadata[0].name
}

output "all_namespaces" {
  description = "List of all created namespaces"
  value = [
    kubernetes_namespace.gateway_system.metadata[0].name,
    kubernetes_namespace.spire.metadata[0].name,
    kubernetes_namespace.a2a_agents.metadata[0].name,
    kubernetes_namespace.observability.metadata[0].name,
    kubernetes_namespace.kafka.metadata[0].name,
    kubernetes_namespace.radgateway.metadata[0].name,
  ]
}
