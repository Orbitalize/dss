
# This file has been automatically generated by /deploy/infrastructure/utils/generate_terraform_variables.py.
# Please do not modify manually.

variable "aws_region" {
  type        = string
  description = <<-EOT
  AWS region
  List of available regions: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html#concepts-regions
  Currently, the terraform module uses the two first availability zones of the region.

  Example: `eu-west-1`
  EOT
}

variable "aws_instance_type" {
  type        = string
  description = <<-EOT
  AWS EC2 instance type used for the Kubernetes node pool.

  Example: `m6g.xlarge` for production and `t3.medium` for development
  EOT
}

variable "aws_route53_zone_id" {
  type        = string
  description = <<-EOT
  AWS Route 53 Zone ID
  This module can automatically create DNS records in a Route 53 Zone.
  Leave empty to disable record creation.

  Example: `Z0123456789ABCDEFGHIJ`
  EOT
}

variable "aws_iam_permissions_boundary" {
  type        = string
  description = <<-EOT
  AWS IAM Policy ARN to be used for permissions boundaries on created roles.

  Example: `arn:aws:iam::123456789012:policy/GithubCIPermissionBoundaries`
  EOT

  default = ""
}


variable "app_hostname" {
  type        = string
  description = <<-EOT
  Fully-qualified domain name of your HTTPS Gateway ingress endpoint.

  Example: `dss.example.com`
  EOT
}

variable "db_hostname_suffix" {
  type        = string
  description = <<-EOT
  The domain name suffix shared by all of your databases nodes.
  For instance, if your database nodes were addressable at 0.db.example.com,
  1.db.example.com and 2.db.example.com (CockroachDB) or 0.master.db.example.com, 1.tserver.db.example.com (Yugabyte), then the value would be db.example.com.
  Example: db.example.com
  EOT
}


variable "datastore_type" {
  type        = string
  description = <<-EOT
  Type of datastore used

  Supported technologies: cockroachdb, yugabyte
  EOT

  validation {
    condition     = contains(["cockroachdb", "yugabyte"], var.datastore_type)
    error_message = "Supported technologies: cockroachdb, yugabyte"
  }

  default = "cockroachdb"
}


variable "node_count" {
  type        = number
  description = <<-EOT
  Number of Kubernetes nodes which should correspond to the desired CockroachDB nodes.
  Currently, only single node or three nodes deployments are supported.

  Example: `3`
  EOT

  validation {
    condition     = (var.datastore_type == "cockroachdb" && contains([1, 3], var.node_count)) || (var.datastore_type == "yugabyte" && var.node_count > 0)
    error_message = "Currently, only 1 node or 3 nodes deployments are supported for CockroachDB. If you use Yugabyte, you need to have at least one node."
  }
}


variable "cluster_name" {
  type        = string
  description = <<-EOT
  Name of the kubernetes cluster that will host this DSS instance (should generally describe the DSS instance being hosted)

  Example: `dss-che-1`
  EOT
}

variable "kubernetes_version" {
  type        = string
  description = <<-EOT
  Desired version of the Kubernetes cluster control plane and nodes.

  Supported versions: 1.24 to 1.32
  EOT

  validation {
    condition     = contains(["1.24", "1.25", "1.26", "1.27", "1.28", "1.29", "1.30", "1.31", "1.32"], var.kubernetes_version)
    error_message = "Supported versions: 1.24 to 1.32"
  }
}


