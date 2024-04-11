variable "crdb_cluster_name" {
  type        = string
  description = <<-EOT
    A string that specifies a cluster name. This is used together to ensure that all newly created
    nodes join the intended cluster when you are running multiple clusters.

    Example: interuss_us_production
  EOT
}
