variable "crdb_readiness_probe" {
  type        = any
  description = <<-EOT
  Readiness probe of the CockroachDB containers, as a Kubernetes probe specification (https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).
  Leave null to use the default readiness probe, which checks the `/health?ready=1` endpoint every 5 seconds.

  Example:
  ```
  {
    httpGet = {
      path   = "/health?ready=1"
      port   = "http"
      scheme = "HTTPS"
    }
    initialDelaySeconds = 10
    periodSeconds       = 5
    timeoutSeconds      = 5
    failureThreshold    = 2
  }
  ```
  EOT

  default = null
}
