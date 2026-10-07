variable "crdb_liveness_probe" {
  type        = any
  description = <<-EOT
  Liveness probe of the CockroachDB containers, as a Kubernetes probe specification (https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).
  CockroachDB does not recommend using a liveness probe in production: under high load, a failing probe restarts nodes and impacts the availability of the cluster.
  Leave null to use the default of the deployment tool:
  - Tanka: no liveness probe.
  - Helm: the default liveness probe of the CockroachDB Helm chart, which cannot be removed.

  Example:
  ```
  {
    httpGet = {
      path   = "/health"
      port   = "http"
      scheme = "HTTPS"
    }
    initialDelaySeconds = 30
    periodSeconds       = 10
    timeoutSeconds      = 5
    failureThreshold    = 6
  }
  ```
  EOT

  default = null
}
