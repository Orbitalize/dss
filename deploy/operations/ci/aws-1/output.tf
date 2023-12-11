output "generated_files_location" {
  value = module.terraform-commons-dss.generated_files_location
}

output "workspace_location" {
  value = module.terraform-commons-dss.workspace_location
}

output "kubernetes_context" {
  value = module.terraform-aws-kubernetes.kubernetes_context_name
}
