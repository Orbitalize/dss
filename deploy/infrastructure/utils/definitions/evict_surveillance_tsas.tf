variable "evict_surveillance_tsas" {
  type        = bool
  description = <<-EOT
  Set this to true to enable cleanup of surveillance TSAs.

  EOT

  default = true
}
