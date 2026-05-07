variable "evict_surveillance_subscriptions" {
  type        = bool
  description = <<-EOT
  Set this to true to enable cleanup of surveillance subscriptions.

  EOT

  default = true
}
